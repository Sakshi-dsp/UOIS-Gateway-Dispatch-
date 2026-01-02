package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type MetricsRecorder interface {
	RecordRequest(endpoint, clientID, protocol, status string)
	RecordRequestDuration(endpoint, status string, duration time.Duration)
}

func MetricsMiddleware(metrics MetricsRecorder, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = c.Request.URL.Path
		}

		c.Next()

		duration := time.Since(start)
		status := getStatusLabel(c.Writer.Status())
		clientID := getClientID(c)

		metrics.RecordRequest(endpoint, clientID, "http", status)
		metrics.RecordRequestDuration(endpoint, status, duration)
	}
}

func getStatusLabel(statusCode int) string {
	if statusCode >= 200 && statusCode < 300 {
		return "success"
	}
	if statusCode >= 400 && statusCode < 500 {
		return "client_error"
	}
	if statusCode >= 500 {
		return "server_error"
	}
	return "unknown"
}

func getClientID(c *gin.Context) string {
	client := GetClientFromContext(c)
	if client != nil {
		return client.ID
	}
	return ""
}
