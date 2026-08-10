// Package metrics provides the Prometheus instrumentation shared by every
// OMS service: a common registry, RED metrics for inbound HTTP traffic, and
// middleware that records them.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds the collectors for a single service. Each service builds one
// during startup and passes it to the middleware wrapping its handlers.
type Metrics struct {
	registry *prometheus.Registry

	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge
}

// New builds the collector set for a service and registers it against a
// private registry. Using a private registry rather than the default one keeps
// each service's exposition free of collectors registered by transitive
// dependencies that we did not opt into.
//
// The service label is applied as a constant label so that a single Grafana
// dashboard can break metrics down by service without every query needing to
// know which job scraped them.
func New(service string) *Metrics {
	labels := prometheus.Labels{"service": service}
	registry := prometheus.NewRegistry()

	m := &Metrics{
		registry: registry,
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "http_requests_total",
				Help:        "Total number of HTTP requests handled, by route, method and status code.",
				ConstLabels: labels,
			},
			[]string{"method", "route", "code"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "http_request_duration_seconds",
				Help:        "HTTP request latency in seconds, by route and method.",
				ConstLabels: labels,
				// Default buckets are tuned for sub-second web traffic, which
				// is the right shape for these services.
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route"},
		),
		requestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "http_requests_in_flight",
				Help:        "Number of HTTP requests currently being served.",
				ConstLabels: labels,
			},
		),
	}

	registry.MustRegister(
		m.requestsTotal,
		m.requestDuration,
		m.requestsInFlight,
		// Go runtime and process metrics give us memory, goroutine and CPU
		// series for free, which the service dashboard graphs alongside RED.
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return m
}

// Registry exposes the underlying registry so a service can register its own
// domain collectors (queue depth, DB pool stats) alongside the shared ones.
func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

// Handler returns the HTTP handler that exposes the registry in the Prometheus
// text exposition format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		// Surface collection problems in the scrape itself rather than
		// silently returning a partial body.
		ErrorHandling: promhttp.HTTPErrorOnError,
	})
}

// Middleware wraps a handler so every request it serves is counted and timed.
//
// The route argument is the templated path (for example "/orders/{id}") rather
// than the concrete URL. Recording the raw path would mint a new time series
// per distinct URL, which is the classic way to blow up Prometheus cardinality.
func (m *Metrics) Middleware(route string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.requestsInFlight.Inc()
		defer m.requestsInFlight.Dec()

		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(recorder, r)

		elapsed := time.Since(start).Seconds()
		m.requestDuration.WithLabelValues(r.Method, route).Observe(elapsed)
		m.requestsTotal.WithLabelValues(r.Method, route, strconv.Itoa(recorder.status)).Inc()
	})
}

// statusRecorder captures the status code written by a handler so the
// middleware can label metrics with it. net/http gives no way to read back a
// status once written, so we have to intercept the write.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.status = status
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	// An implicit 200 happens when a handler writes a body without ever
	// calling WriteHeader; mark the header as written so a later explicit
	// call cannot overwrite the status we already recorded.
	r.wroteHeader = true
	return r.ResponseWriter.Write(b)
}
