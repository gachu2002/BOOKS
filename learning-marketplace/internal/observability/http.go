package observability

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsInFlight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "learning_marketplace_http_requests_in_flight",
		Help: "Current number of in-flight HTTP requests.",
	}, []string{"method"})

	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "learning_marketplace_http_requests_total",
		Help: "Total number of processed HTTP requests.",
	}, []string{"method", "route", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "learning_marketplace_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route", "status"})
)

func InstrumentHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		start := time.Now()
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		httpRequestsInFlight.WithLabelValues(method).Inc()
		defer httpRequestsInFlight.WithLabelValues(method).Dec()

		next.ServeHTTP(wrapped, r)

		status := strconv.Itoa(wrapped.Status())
		route := routePattern(r)
		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(method, route, status).Inc()
		httpRequestDuration.WithLabelValues(method, route, status).Observe(duration)
	})
}

func LogRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(wrapped, r)

		slog.Info("http request completed",
			"method", r.Method,
			"route", routePattern(r),
			"path", r.URL.Path,
			"status", wrapped.Status(),
			"bytes", wrapped.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}

func routePattern(r *http.Request) string {
	ctx := chi.RouteContext(r.Context())
	if ctx == nil {
		return fallbackPath(r.URL.Path)
	}

	pattern := ctx.RoutePattern()
	if strings.TrimSpace(pattern) == "" {
		return fallbackPath(r.URL.Path)
	}

	return pattern
}

func fallbackPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return "unknown"
	}

	return path
}
