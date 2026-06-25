package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	redirectRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redirect_requests_total",
			Help: "Total number of redirect requests.",
		},
		[]string{"method", "status_code"},
	)
	redirectRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redirect_request_duration_seconds",
			Help:    "Duration of redirect requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status_code"},
	)
)

func init() {
	prometheus.MustRegister(redirectRequestsTotal, redirectRequestDuration)
}

type redirectHandler struct {
	destination *url.URL
	statusCode  int
}

func newRedirectHandler(dest string, statusCode int) (*redirectHandler, error) {
	u, err := url.Parse(dest)
	if err != nil {
		return nil, fmt.Errorf("failed to parse destination URL: %w", err)
	}
	return &redirectHandler{destination: u, statusCode: statusCode}, nil
}

func (h *redirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/health":
		h.handleHealth(w, r)
	case "/ready":
		h.handleReady(w, r)
	case "/metrics":
		h.handleMetrics(w, r)
	default:
		h.handleRedirect(w, r)
	}
}

func (h *redirectHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		slog.Error("failed to write health response", "error", err)
	}
}

func (h *redirectHandler) handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ready"}); err != nil {
		slog.Error("failed to write ready response", "error", err)
	}
}

func (h *redirectHandler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

func (h *redirectHandler) handleRedirect(w http.ResponseWriter, r *http.Request) {
	statusStr := fmt.Sprintf("%d", h.statusCode)
	timer := prometheus.NewTimer(redirectRequestDuration.WithLabelValues(r.Method, statusStr))
	defer timer.ObserveDuration()

	dest := h.buildRedirectURL(r)
	redirectRequestsTotal.WithLabelValues(r.Method, statusStr).Inc()

	slog.Info("redirect",
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"destination", dest,
		"status", h.statusCode,
	)

	w.Header().Set("Location", dest)
	w.WriteHeader(h.statusCode)
}

func (h *redirectHandler) buildRedirectURL(r *http.Request) string {
	destPath := h.destination.Path
	if destPath == "" {
		destPath = "/"
	}

	joinedPath := joinURLPath(destPath, r.URL.Path)

	result := url.URL{
		Scheme:   h.destination.Scheme,
		Host:     h.destination.Host,
		Path:     joinedPath,
		RawQuery: r.URL.RawQuery,
	}
	return result.String()
}

// joinURLPath joins two URL path segments, normalizing slashes.
func joinURLPath(base, rel string) string {
	base = strings.TrimRight(base, "/")
	if !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}
	if base == "" {
		return rel
	}
	return base + rel
}

// responseWriterWrapper wraps http.ResponseWriter to capture the status code.
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// loggingMiddleware logs every request with method, path, status, and duration.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapper := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapper, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapper.statusCode,
			"duration", time.Since(start).String(),
		)
	})
}
