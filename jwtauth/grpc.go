package jwtauth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a gRPC unary server interceptor for JWT authentication
func UnaryServerInterceptor(cfg *Config) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		startTime := time.Now()

		// Generate request ID for correlation
		requestID := uuid.New().String()

		// Extract metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logAuthFailureGRPC(cfg, requestID, "", NewValidationError(ErrMissingToken, "metadata not found", nil), time.Since(startTime))
			return nil, status.Error(codes.Unauthenticated, "metadata not found")
		}

		// Extract token from metadata
		token, err := extractTokenFromMetadata(md)
		if err != nil {
			logAuthFailureGRPC(cfg, requestID, token, err, time.Since(startTime))
			return nil, status.Error(codes.Unauthenticated, getErrorCode(err))
		}

		// Validate token
		claims, err := parseAndValidateJWT(token, cfg)
		if err != nil {
			logAuthFailureGRPC(cfg, requestID, token, err, time.Since(startTime))
			return nil, status.Error(codes.Unauthenticated, getErrorCode(err))
		}

		// Inject claims and request ID into context
		ctx = WithClaims(ctx, claims)
		ctx = WithRequestID(ctx, requestID)

		// Log successful authentication
		logAuthSuccessGRPC(cfg, requestID, claims, token, time.Since(startTime))

		// Call the handler with enriched context
		return handler(ctx, req)
	}
}

// logAuthSuccessGRPC logs a successful gRPC authentication event
func logAuthSuccessGRPC(cfg *Config, requestID string, claims *Claims, token string, latency time.Duration) {
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

// logAuthFailureGRPC logs a failed gRPC authentication event
func logAuthFailureGRPC(cfg *Config, requestID string, token string, err error, latency time.Duration) {
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
