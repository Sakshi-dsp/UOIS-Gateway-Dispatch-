package middleware

import (
	"context"
	"encoding/base64"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"uois-gateway/internal/models"
	"uois-gateway/pkg/errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const ClientContextKey = "client"

type AuthService interface {
	AuthenticateClient(ctx context.Context, clientID, clientSecret, clientIP string) (*models.Client, error)
}

type RateLimitService interface {
	CheckRateLimit(ctx context.Context, clientID string) (allowed bool, remaining int64, resetAt time.Time, err error)
	GetRateLimitError(ctx context.Context, clientID string) error
}

type TrustedProxyChecker interface {
	IsTrustedProxy(remoteAddr string) bool
}

type AuthMiddlewareConfig struct {
	TrustedProxyChecker TrustedProxyChecker
}

func AuthMiddleware(authService AuthService, rateLimitService RateLimitService, logger *zap.Logger) gin.HandlerFunc {
	return AuthMiddlewareWithConfig(authService, rateLimitService, logger, AuthMiddlewareConfig{})
}

func AuthMiddlewareWithConfig(authService AuthService, rateLimitService RateLimitService, logger *zap.Logger, config AuthMiddlewareConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID, clientSecret, ok := extractCredentials(c)
		if !ok {
			respondError(c, logger, http.StatusUnauthorized, errors.NewDomainError(65002, "authentication failed", "missing credentials"))
			c.Abort()
			return
		}

		clientIP := extractClientIP(c, config.TrustedProxyChecker)
		client, err := authService.AuthenticateClient(c.Request.Context(), clientID, clientSecret, clientIP)
		if err != nil {
			httpStatus := errors.GetHTTPStatus(err)
			respondError(c, logger, httpStatus, err)
			c.Abort()
			return
		}

		allowed, remaining, resetAt, err := rateLimitService.CheckRateLimit(c.Request.Context(), clientID)
		if err != nil {
			httpStatus := errors.GetHTTPStatus(err)
			respondError(c, logger, httpStatus, err)
			c.Abort()
			return
		}

		if !allowed {
			rateLimitErr := rateLimitService.GetRateLimitError(c.Request.Context(), clientID)
			setRateLimitHeaders(c, 0, resetAt)
			respondError(c, logger, http.StatusTooManyRequests, rateLimitErr)
			c.Abort()
			return
		}

		setRateLimitHeaders(c, remaining, resetAt)
		c.Set(ClientContextKey, client)
		c.Next()
	}
}

func extractCredentials(c *gin.Context) (clientID, clientSecret string, ok bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", "", false
	}

	if strings.HasPrefix(authHeader, "Basic ") {
		return extractBasicAuth(authHeader)
	}

	if strings.HasPrefix(authHeader, "Bearer ") {
		return extractBearerToken(authHeader)
	}

	return "", "", false
}

func extractBasicAuth(authHeader string) (clientID, clientSecret string, ok bool) {
	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}

func extractBearerToken(authHeader string) (clientID, clientSecret string, ok bool) {
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return "", "", false
	}

	// Bearer tokens are treated as opaque secrets
	// For clientID:secret format, use Basic Auth instead
	return token, "", true
}

func extractClientIP(c *gin.Context, proxyChecker TrustedProxyChecker) string {
	remoteAddr := c.Request.RemoteAddr
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	// Only trust proxy headers if request comes from a trusted proxy
	if proxyChecker != nil && proxyChecker.IsTrustedProxy(host) {
		if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}

		if xrip := c.GetHeader("X-Real-IP"); xrip != "" {
			return xrip
		}
	}

	// Fallback: use RemoteAddr (already extracted as host)
	return host
}

func setRateLimitHeaders(c *gin.Context, remaining int64, resetAt time.Time) {
	if remaining >= 0 {
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
	}

	if !resetAt.IsZero() {
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
		retryAfter := int(time.Until(resetAt).Seconds())
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
	}
}

func respondError(c *gin.Context, logger *zap.Logger, statusCode int, err error) {
	logger.Warn("request rejected by middleware",
		zap.Int("status_code", statusCode),
		zap.Error(err),
	)

	// Sanitize error response - don't leak internal details
	c.JSON(statusCode, gin.H{
		"error": "request rejected",
	})
}

func GetClientFromContext(c *gin.Context) *models.Client {
	client, exists := c.Get(ClientContextKey)
	if !exists {
		return nil
	}

	clientModel, ok := client.(*models.Client)
	if !ok {
		return nil
	}

	return clientModel
}
