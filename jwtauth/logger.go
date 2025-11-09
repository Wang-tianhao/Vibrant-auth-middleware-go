package jwtauth

import (
	"log/slog"
	"time"
)

// SecurityEvent represents a structured security log entry
type SecurityEvent struct {
	EventType     string        // "success" or "failure"
	Timestamp     time.Time     // Event timestamp
	RequestID     string        // Correlation ID
	UserID        string        // Subject from claims (empty on failure)
	Algorithm     string        // Algorithm used (HS256, RS256) or attempted
	FailureReason string        // Error code (on failure)
	TokenPreview  string        // Redacted token preview
	Latency       time.Duration // Validation latency
}

// LogValue implements slog.LogValuer for structured logging with redaction
func (e SecurityEvent) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("event", e.EventType),
		slog.Time("timestamp", e.Timestamp),
		slog.String("request_id", e.RequestID),
		slog.String("user_id", e.UserID),
		slog.String("algorithm", e.Algorithm),
		slog.String("failure_reason", e.FailureReason),
		slog.String("token", redactToken(e.TokenPreview)),
		slog.Duration("latency", e.Latency),
	)
}

// redactToken redacts sensitive token data
func redactToken(token string) string {
	if len(token) == 0 {
		return ""
	}
	if len(token) <= 8 {
		return "***"
	}
	return token[:8] + "..."
}

// logSecurityEvent emits a security event via the configured logger
func logSecurityEvent(logger *slog.Logger, event SecurityEvent) {
	if logger == nil {
		return // Logging disabled
	}

	if event.EventType == "failure" {
		logger.Warn("authentication failed", "auth_event", event)
	} else {
		logger.Info("authentication succeeded", "auth_event", event)
	}
}
