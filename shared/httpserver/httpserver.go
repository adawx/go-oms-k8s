// Package httpserver wires up the HTTP surface every OMS service exposes: an
// application listener for real traffic and a separate admin listener carrying
// metrics and health probes.
package httpserver

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-oms/shared/metrics"
)

// Server runs a service's application and admin listeners together and shuts
// both down cleanly on SIGTERM.
//
// Metrics and health live on their own port so they are not exposed through
// the api-gateway LoadBalancer and cannot be reached from outside the cluster.
// Prometheus scrapes pods directly, so it reaches the admin port regardless.
type Server struct {
	app   *http.Server
	admin *http.Server
}

// New builds a Server. appAddr carries application routes; adminAddr carries
// /metrics, /healthz and /readyz.
func New(appAddr, adminAddr string, appHandler http.Handler, m *metrics.Metrics, ready func() error) *Server {
	adminMux := http.NewServeMux()
	adminMux.Handle("/metrics", m.Handler())

	// Liveness answers "is the process running": if it can serve this, it is
	// alive. It deliberately does not check dependencies, otherwise a database
	// blip would make Kubernetes restart healthy pods.
	adminMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Readiness answers "should this pod receive traffic", so it does check
	// dependencies. Services with none pass a nil ready func.
	adminMux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if ready != nil {
			if err := ready(); err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})

	return &Server{
		app: &http.Server{
			Addr:              appAddr,
			Handler:           appHandler,
			ReadHeaderTimeout: 10 * time.Second,
		},
		admin: &http.Server{
			Addr:              adminAddr,
			Handler:           adminMux,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

// Run starts both listeners and blocks until the process is signalled or
// either listener fails. It returns once both have shut down.
func (s *Server) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)

	go func() {
		log.Printf("admin listener on %s (/metrics, /healthz, /readyz)", s.admin.Addr)
		if err := s.admin.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	go func() {
		log.Printf("app listener on %s", s.app.Addr)
		if err := s.app.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	var runErr error
	select {
	case <-ctx.Done():
		log.Print("shutdown signal received")
	case runErr = <-errCh:
		log.Printf("listener failed: %v", runErr)
	}

	// Bound shutdown so a hung connection cannot outlive the pod's grace
	// period and turn a graceful stop into a SIGKILL.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.app.Shutdown(shutdownCtx); err != nil {
		log.Printf("app shutdown: %v", err)
	}
	if err := s.admin.Shutdown(shutdownCtx); err != nil {
		log.Printf("admin shutdown: %v", err)
	}

	return runErr
}

// ExitOnError logs and exits non-zero when Run returns a listener failure, so
// a crashed listener surfaces as a pod restart rather than a silent exit 0.
func ExitOnError(err error) {
	if err != nil {
		log.Printf("server exited with error: %v", err)
		os.Exit(1)
	}
}
