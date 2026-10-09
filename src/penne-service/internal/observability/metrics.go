package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HttpRequestsTotal counts total HTTP requests processed by endpoint and status
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "penne_http_requests_total",
			Help: "Total number of HTTP requests processed by penne-service",
		},
		[]string{"method", "path", "status"},
	)

	// HttpRequestDuration records response latency
	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "penne_http_request_duration_seconds",
			Help:    "Histogram of response latency for HTTP requests",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"method", "path"},
	)
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// MetricsMiddleware records request count and duration per route pattern
func MetricsMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			duration := time.Since(start).Seconds()

			routePath := r.URL.Path
			if route := mux.CurrentRoute(r); route != nil {
				if pathTpl, err := route.GetPathTemplate(); err == nil {
					routePath = pathTpl
				}
			}

			HttpRequestsTotal.WithLabelValues(r.Method, routePath, strconv.Itoa(sw.status)).Inc()
			HttpRequestDuration.WithLabelValues(r.Method, routePath).Observe(duration)
		})
	}
}
