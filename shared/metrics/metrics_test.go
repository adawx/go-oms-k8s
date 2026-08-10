package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMiddlewareRecordsStatusAndCount(t *testing.T) {
	m := New("test-service")

	handler := m.Middleware("/orders", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	for range 3 {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/orders", nil))
	}

	expected := `
# HELP http_requests_total Total number of HTTP requests handled, by route, method and status code.
# TYPE http_requests_total counter
http_requests_total{code="201",method="POST",route="/orders",service="test-service"} 3
`
	if err := testutil.CollectAndCompare(m.requestsTotal, strings.NewReader(expected), "http_requests_total"); err != nil {
		t.Fatalf("unexpected request counter state: %v", err)
	}
}

func TestMiddlewareDefaultsToStatusOK(t *testing.T) {
	m := New("test-service")

	// A handler that writes a body without calling WriteHeader implicitly
	// returns 200; the recorder must report that rather than a zero status.
	handler := m.Middleware("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	got := testutil.ToFloat64(m.requestsTotal.WithLabelValues(http.MethodGet, "/", "200"))
	if got != 1 {
		t.Fatalf("expected one request recorded with status 200, got %v", got)
	}
}

func TestMiddlewareReleasesInFlightGauge(t *testing.T) {
	m := New("test-service")

	handler := m.Middleware("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mid-request the gauge should be held.
		if got := testutil.ToFloat64(m.requestsInFlight); got != 1 {
			t.Errorf("expected in-flight gauge of 1 during request, got %v", got)
		}
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if got := testutil.ToFloat64(m.requestsInFlight); got != 0 {
		t.Fatalf("expected in-flight gauge released to 0 after request, got %v", got)
	}
}

func TestHandlerExposesRegisteredMetrics(t *testing.T) {
	m := New("api-gateway")

	handler := m.Middleware("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from metrics handler, got %d", rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{
		`http_requests_total{code="200",method="GET",route="/",service="api-gateway"}`,
		"http_request_duration_seconds_bucket",
		"go_goroutines", // proves the Go runtime collector is registered
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
}
