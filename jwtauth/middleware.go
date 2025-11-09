package jwtauth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// JWTAuth returns a Gin middleware handler for JWT authentication
func JWTAuth(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Generate or extract request ID for correlation
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Extract token from request
		token, err := extractToken(c.Request, cfg)
		if err != nil {
			logAuthFailure(cfg, requestID, token, err, time.Since(startTime))
			c.AbortWithStatusJSON(401, buildErrorResponse(err))
			return
		}

		// Validate token
		claims, err := parseAndValidateJWT(token, cfg)
		if err != nil {
			logAuthFailure(cfg, requestID, token, err, time.Since(startTime))
			c.AbortWithStatusJSON(401, buildErrorResponse(err))
			return
		}

		// Inject claims and request ID into context
		ctx := WithClaims(c.Request.Context(), claims)
		ctx = WithRequestID(ctx, requestID)
		c.Request = c.Request.WithContext(ctx)

		// Log successful authentication
		logAuthSuccess(cfg, requestID, claims, token, time.Since(startTime))

		// Continue to next handler
		c.Next()
	}
}

// logAuthSuccess logs a successful authentication event
func logAuthSuccess(cfg *Config, requestID string, claims *Claims, token string, latency time.Duration) {
	if cfg.Logger() == nil {
		return
	}

	event := SecurityEvent{
		EventType:    "success",
		Timestamp:    time.Now(),
		RequestID:    requestID,
		UserID:       claims.Subject,
		Algorithm:    extractAlgorithmFromToken(token),
		TokenPreview: token,
		Latency:      latency,
	}

	logSecurityEvent(cfg.Logger(), event)
}

// logAuthFailure logs a failed authentication event
func logAuthFailure(cfg *Config, requestID string, token string, err error, latency time.Duration) {
	if cfg.Logger() == nil {
		return
	}

	event := SecurityEvent{
		EventType:     "failure",
		Timestamp:     time.Now(),
		RequestID:     requestID,
		Algorithm:     extractAlgorithmFromToken(token),
		FailureReason: getErrorCode(err),
		TokenPreview:  token,
		Latency:       latency,
	}

	logSecurityEvent(cfg.Logger(), event)
}

// getErrorCode extracts the error code from a validation error
func getErrorCode(err error) string {
	if valErr, ok := err.(*ValidationError); ok {
		return string(valErr.Code)
	}
	return "UNKNOWN"
}

// buildErrorResponse constructs error response with optional message field
// For UNSUPPORTED_ALGORITHM and MALFORMED errors, includes helpful message from ValidationError
func buildErrorResponse(err error) gin.H {
	response := gin.H{
		"error":  "unauthorized",
		"reason": getErrorCode(err),
	}

	// Add message field for specific error types (US3 requirement)
	if valErr, ok := err.(*ValidationError); ok {
		// Include message for UNSUPPORTED_ALGORITHM (lists available algorithms)
		// and MALFORMED errors (helps debugging)
		if valErr.Code == ErrUnsupportedAlgorithm || valErr.Code == ErrMalformedAlgorithmHeader {
			if valErr.Message != "" {
				response["message"] = valErr.Message
			}
		}
	}

	return response
}

// extractAlgorithmFromToken extracts the algorithm from a JWT token header
// Returns empty string if extraction fails (token will be logged as invalid anyway)
func extractAlgorithmFromToken(token string) string {
	// JWT format: header.payload.signature
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "MALFORMED"
	}

	// Decode header (first part)
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "MALFORMED"
	}

	// Parse header JSON
	var header map[string]interface{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return "MALFORMED"
	}

	// Extract alg field
	if alg, ok := header["alg"].(string); ok {
		return alg
	}

	return "MALFORMED"
}
