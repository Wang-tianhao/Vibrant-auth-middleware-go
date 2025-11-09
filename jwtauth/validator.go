package jwtauth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// parseAndValidateJWT parses and validates a JWT token string
func parseAndValidateJWT(tokenString string, cfg *Config) (*Claims, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the algorithm and get the appropriate signing key
		signingKey, err := validateAlgorithm(token, cfg)
		if err != nil {
			return nil, err
		}
		return signingKey, nil
	})

	if err != nil {
		// Check if error is already a ValidationError (from validateAlgorithm)
		// The JWT library may wrap our error, so we need to unwrap it
		if valErr, ok := err.(*ValidationError); ok {
			return nil, valErr
		}

		// Unwrap error to check if the underlying error is a ValidationError
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			return nil, valErr
		}

		// Check for specific JWT library error types
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, NewValidationError(ErrExpired, "token has expired", err)
		}
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			return nil, NewValidationError(ErrInvalidSignature, "invalid signature", err)
		}

		// Check error message for signature-related failures
		errMsg := err.Error()
		if containsAny(errMsg, []string{"signature", "invalid"}) {
			return nil, NewValidationError(ErrInvalidSignature, "signature verification failed", err)
		}

		return nil, NewValidationError(ErrMalformed, "malformed token", err)
	}

	if !token.Valid {
		return nil, NewValidationError(ErrInvalidSignature, "token is invalid", nil)
	}

	// Extract claims
	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, NewValidationError(ErrMalformed, "invalid claims format", nil)
	}

	// Validate and convert claims
	claims, err := mapJWTClaimsToClaims(mapClaims, cfg)
	if err != nil {
		return nil, err
	}

	// Validate time-based claims with clock skew
	if err := validateClaims(claims, cfg); err != nil {
		return nil, err
	}

	// Validate required claims
	if err := validateRequiredClaims(mapClaims, cfg); err != nil {
		return nil, err
	}

	return claims, nil
}

// validateAlgorithm ensures the token uses a configured algorithm and returns the appropriate signing key
func validateAlgorithm(token *jwt.Token, cfg *Config) (interface{}, error) {
	// Extract algorithm from token header
	alg, ok := token.Header["alg"].(string)
	if !ok {
		// Check if alg field exists but is not a string
		if _, exists := token.Header["alg"]; exists {
			return nil, NewValidationError(ErrMalformedAlgorithmHeader, "algorithm header must be a string", nil)
		}
		return nil, NewValidationError(ErrMalformed, "missing algorithm in token header", nil)
	}

	// Reject "none" algorithm explicitly (case-insensitive check)
	if alg == "none" || alg == "None" || alg == "NONE" {
		return nil, NewValidationError(ErrNoneAlgorithm, "none algorithm not allowed", nil)
	}

	// Look up validator for this algorithm (case-sensitive)
	validator, exists := cfg.getValidator(alg)
	if !exists {
		availableAlgs := cfg.AvailableAlgorithms()
		return nil, NewValidationError(
			ErrUnsupportedAlgorithm,
			fmt.Sprintf("algorithm %s not supported (available: %s)", alg, joinStrings(availableAlgs)),
			nil,
		)
	}

	// Verify token's signing method matches the validator's expected signing method
	// This prevents algorithm confusion attacks
	if token.Method.Alg() != validator.signingMethod.Alg() {
		return nil, NewValidationError(
			ErrInvalidSignature,
			fmt.Sprintf("algorithm confusion detected: token method %s does not match expected method %s",
				token.Method.Alg(), validator.signingMethod.Alg()),
			nil,
		)
	}

	// Return the signing key for this algorithm
	return validator.signingKey, nil
}

// joinStrings joins a string slice with commas
func joinStrings(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += ", " + strs[i]
	}
	return result
}

// containsAny checks if a string contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// mapJWTClaimsToClaims converts jwt.MapClaims to our Claims struct
func mapJWTClaimsToClaims(mapClaims jwt.MapClaims, cfg *Config) (*Claims, error) {
	claims := &Claims{
		Custom: make(map[string]interface{}),
	}

	// Extract standard claims
	if sub, ok := mapClaims["sub"].(string); ok {
		claims.Subject = sub
	}
	if iss, ok := mapClaims["iss"].(string); ok {
		claims.Issuer = iss
	}
	if aud, ok := mapClaims["aud"].(string); ok {
		claims.Audience = aud
	}
	if jti, ok := mapClaims["jti"].(string); ok {
		claims.JWTID = jti
	}

	// Extract time-based claims
	if exp, err := mapClaims.GetExpirationTime(); err == nil && exp != nil {
		claims.ExpiresAt = exp.Time
	}
	if nbf, err := mapClaims.GetNotBefore(); err == nil && nbf != nil {
		claims.NotBefore = nbf.Time
	}
	if iat, err := mapClaims.GetIssuedAt(); err == nil && iat != nil {
		claims.IssuedAt = iat.Time
	}

	// Copy custom claims
	standardClaims := map[string]bool{
		"sub": true, "iss": true, "aud": true, "exp": true,
		"nbf": true, "iat": true, "jti": true,
	}
	for key, value := range mapClaims {
		if !standardClaims[key] {
			claims.Custom[key] = value
		}
	}

	return claims, nil
}

// validateClaims validates time-based claims with clock skew tolerance
func validateClaims(claims *Claims, cfg *Config) error {
	now := time.Now()
	skew := cfg.ClockSkewLeeway()

	// Validate expiration time
	if !claims.ExpiresAt.IsZero() {
		if now.After(claims.ExpiresAt.Add(skew)) {
			return NewValidationError(
				ErrExpired,
				fmt.Sprintf("token expired at %v", claims.ExpiresAt),
				nil,
			)
		}
	}

	// Validate not-before time
	if !claims.NotBefore.IsZero() {
		if now.Before(claims.NotBefore.Add(-skew)) {
			return NewValidationError(
				ErrExpired,
				fmt.Sprintf("token not valid until %v", claims.NotBefore),
				nil,
			)
		}
	}

	return nil
}

// validateRequiredClaims ensures all required claims are present
func validateRequiredClaims(mapClaims jwt.MapClaims, cfg *Config) error {
	for _, claimName := range cfg.RequiredClaims() {
		if _, ok := mapClaims[claimName]; !ok {
			return NewValidationError(
				ErrMalformed,
				fmt.Sprintf("required claim missing: %s", claimName),
				nil,
			)
		}
	}
	return nil
}
