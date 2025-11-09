package jwtauth

import "context"

// contextKey is an unexported type for context keys to prevent collisions
type contextKey string

const (
	claimsContextKey    contextKey = "github.com/user/vibrant-auth-middleware-go/jwtauth:claims"
	requestIDContextKey contextKey = "github.com/user/vibrant-auth-middleware-go/jwtauth:request_id"
)

// WithClaims stores validated JWT claims in the request context.
// Claims are immutable and should not be modified by downstream handlers.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// GetClaims retrieves validated JWT claims from the request context.
// Returns nil, false if claims are not present or have wrong type.
// Always check the ok return value before using claims.
func GetClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	return claims, ok
}

// MustGetClaims retrieves claims from context and panics if not present.
// Use only when you're certain claims exist (e.g., after middleware validation).
func MustGetClaims(ctx context.Context) *Claims {
	claims, ok := GetClaims(ctx)
	if !ok {
		panic("jwtauth: claims not found in context")
	}
	return claims
}

// WithRequestID stores a request ID in context for correlation
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDContextKey).(string)
	return id, ok
}
