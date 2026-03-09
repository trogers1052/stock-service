package api

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/trogers1052/stock-alert-system/internal/metrics"
)

// APIKeyAuth returns middleware that validates the X-API-Key header.
// If apiKey is empty, authentication is disabled (development mode).
func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			provided := r.Header.Get("X-API-Key")
			if provided == "" {
				http.Error(w, `{"error":"missing X-API-Key header"}`, http.StatusUnauthorized)
				return
			}

			if subtle.ConstantTimeCompare([]byte(provided), []byte(apiKey)) != 1 {
				http.Error(w, `{"error":"invalid API key"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// statusRecorder wraps http.ResponseWriter so we can capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// PrometheusMiddleware records HTTP request count and latency for every request.
func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		sr := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(sr, r)

		elapsed := time.Since(start).Seconds()

		// Use the matched route template (e.g. "/api/v1/stocks/{symbol}")
		// so we don't get high-cardinality label values from path params.
		endpoint := r.URL.Path
		if route := mux.CurrentRoute(r); route != nil {
			if tpl, err := route.GetPathTemplate(); err == nil {
				endpoint = tpl
			}
		}

		method := r.Method
		code := strconv.Itoa(sr.statusCode)

		metrics.HTTPRequests.WithLabelValues(method, endpoint, code).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(elapsed)
	})
}
