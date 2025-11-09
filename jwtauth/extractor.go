package jwtauth

import (
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
)

// extractTokenFromHeader extracts JWT token from Authorization header
// Expected format: "Authorization: Bearer <token>"
func extractTokenFromHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", NewValidationError(ErrMissingToken, "authorization header not found", nil)
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", NewValidationError(ErrMalformed, "invalid authorization header format, expected 'Bearer <token>'", nil)
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", NewValidationError(ErrMissingToken, "token is empty", nil)
	}

	return token, nil
}

// extractTokenFromCookie extracts JWT token from a cookie
func extractTokenFromCookie(r *http.Request, cookieName string) (string, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", NewValidationError(ErrMissingToken, "cookie not found", err)
	}

	token := strings.TrimSpace(cookie.Value)
	if token == "" {
		return "", NewValidationError(ErrMissingToken, "cookie value is empty", nil)
	}

	return token, nil
}

// extractToken extracts JWT token from HTTP request
// Checks Authorization header first, then falls back to cookie if configured
func extractToken(r *http.Request, cfg *Config) (string, error) {
	// Try header first
	token, err := extractTokenFromHeader(r)
	if err == nil {
		return token, nil
	}

	// If cookie is configured, try it as fallback
	if cfg.CookieName() != "" {
		token, cookieErr := extractTokenFromCookie(r, cfg.CookieName())
		if cookieErr == nil {
			return token, nil
		}
	}

	// Return the original header error
	return "", err
}

// extractTokenFromMetadata extracts JWT token from gRPC metadata
func extractTokenFromMetadata(md metadata.MD) (string, error) {
	values := md.Get("authorization")
	if len(values) == 0 {
		return "", NewValidationError(ErrMissingToken, "authorization metadata not found", nil)
	}

	authHeader := values[0]
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", NewValidationError(ErrMalformed, "invalid authorization format, expected 'Bearer <token>'", nil)
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", NewValidationError(ErrMissingToken, "token is empty", nil)
	}

	return token, nil
}
