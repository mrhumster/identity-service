package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// RED-style HTTP metrics shared by identity-service and stream-service
// (the latter imports this package from the identity module).
var (
	httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests served.",
	}, []string{"method", "endpoint", "status"})

	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "endpoint", "status"})
)

func init() {
	prometheus.MustRegister(httpRequests, httpDuration)
}

// MetricsMiddleware records request count and latency per
// method / route template / status code.
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		labels := prometheus.Labels{
			"method":   c.Request.Method,
			"endpoint": endpoint,
			"status":   status,
		}
		httpRequests.With(labels).Inc()
		httpDuration.With(labels).Observe(time.Since(start).Seconds())
	}
}

// CountHTTPResult is a convenience for business counters keyed on an
// additional result label alongside the request route.
func HTTPRequests() *prometheus.CounterVec { return httpRequests }