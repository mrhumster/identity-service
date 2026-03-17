package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func StructuredLog() gin.HandlerFunc {
	skipPath := map[string]struct{}{
		"/stream/health": {},
		"/auth/health":   {},
		"/metrics":       {},
	}
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()
		if _, ok := skipPath[path]; ok {
			return
		}
		end := time.Now()
		latency := end.Sub(start)
		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() {
				slog.Error("HTTP Request Error",
					"status", c.Writer.Status(),
					"method", c.Request.Method,
					"path", path,
					"error", e)
			}
		} else {
			slog.Info("HTTP Request",
				"status", c.Writer.Status(),
				"method", c.Request.Method,
				"path", path,
				"query", query,
				"ip", c.ClientIP(),
				"latency", latency,
				"user_agent", c.Request.UserAgent())
		}
	}
}
