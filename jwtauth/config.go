package jwtauth

import (
	"crypto/rsa"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// algorithmValidator holds signing key and method for a specific algorithm
type algorithmValidator struct {
	signingKey    interface{}       // []byte for HS256, *rsa.PublicKey for RS256
	signingMethod jwt.SigningMethod // jwt.SigningMethodHS256 or jwt.SigningMethodRS256
}

// Config holds immutable configuration for JWT validation
type Config struct {
	validators       map[string]algorithmValidator // "HS256" -> validator, "RS256" -> validator
	clockSkewLeeway  time.Duration
	cookieName       string
	requiredClaims   []string
	logger           *slog.Logger
	contextKeyPrefix string
}

// ConfigOption is a functional option for configuring the middleware
type ConfigOption func(*Config) error

// NewConfig creates a new immutable configuration with the given options
func NewConfig(opts ...ConfigOption) (*Config, error) {
	cfg := &Config{
		validators:       make(map[string]algorithmValidator),
		clockSkewLeeway:  60 * time.Second, // Default 60 seconds
		contextKeyPrefix: "jwtauth",
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, NewValidationError(ErrConfigError, fmt.Sprintf("configuration error: %v", err), err)
		}
	}

	// Validate required fields
	if len(cfg.validators) == 0 {
		return nil, NewValidationError(ErrConfigError, "at least one algorithm must be configured (use WithHS256 or WithRS256)", nil)
	}

	// Reject "none" algorithm variants
	for alg := range cfg.validators {
		if alg == "none" || alg == "None" || alg == "NONE" {
			return nil, NewValidationError(ErrConfigError, "none algorithm is prohibited", nil)
		}
	}

	// Validate each validator
	for alg, validator := range cfg.validators {
		if validator.signingKey == nil {
			return nil, NewValidationError(ErrConfigError, fmt.Sprintf("signing key for %s cannot be nil", alg), nil)
		}
		if validator.signingMethod == nil {
			return nil, NewValidationError(ErrConfigError, fmt.Sprintf("signing method for %s cannot be nil", alg), nil)
		}
	}

	return cfg, nil
}

// WithHS256 configures HMAC-SHA256 validation with the given secret
func WithHS256(secret []byte) ConfigOption {
	return func(c *Config) error {
		if len(secret) < 32 {
			return fmt.Errorf("HS256 secret must be at least 32 bytes (256 bits), got %d bytes", len(secret))
		}
		c.validators["HS256"] = algorithmValidator{
			signingKey:    secret,
			signingMethod: jwt.SigningMethodHS256,
		}
		return nil
	}
}

// WithRS256 configures RSA-SHA256 validation with the given public key
func WithRS256(publicKey *rsa.PublicKey) ConfigOption {
	return func(c *Config) error {
		if publicKey == nil {
			return fmt.Errorf("RS256 public key cannot be nil")
		}
		c.validators["RS256"] = algorithmValidator{
			signingKey:    publicKey,
			signingMethod: jwt.SigningMethodRS256,
		}
		return nil
	}
}

// WithClockSkew sets the clock skew tolerance for exp/nbf validation
func WithClockSkew(skew time.Duration) ConfigOption {
	return func(c *Config) error {
		if skew < 0 {
			return fmt.Errorf("clock skew must be non-negative, got %v", skew)
		}
		c.clockSkewLeeway = skew
		return nil
	}
}

// WithCookie enables token extraction from a cookie with the given name
func WithCookie(cookieName string) ConfigOption {
	return func(c *Config) error {
		c.cookieName = cookieName
		return nil
	}
}

// WithLogger sets a structured logger for security events
func WithLogger(logger *slog.Logger) ConfigOption {
	return func(c *Config) error {
		c.logger = logger
		return nil
	}
}

// WithRequiredClaims specifies claim names that must be present in the JWT
func WithRequiredClaims(claims ...string) ConfigOption {
	return func(c *Config) error {
		c.requiredClaims = append(c.requiredClaims, claims...)
		return nil
	}
}

// Getter methods for internal use

// AvailableAlgorithms returns a sorted list of configured algorithm names
func (c *Config) AvailableAlgorithms() []string {
	algs := make([]string, 0, len(c.validators))
	for alg := range c.validators {
		algs = append(algs, alg)
	}
	sort.Strings(algs)
	return algs
}

// getValidator retrieves the validator for a given algorithm (unexported, for internal use)
func (c *Config) getValidator(alg string) (algorithmValidator, bool) {
	validator, exists := c.validators[alg]
	return validator, exists
}

// Algorithm returns the first algorithm in sorted order (deprecated, for backward compatibility)
// Deprecated: Use AvailableAlgorithms() for multi-algorithm configurations
func (c *Config) Algorithm() string {
	algs := c.AvailableAlgorithms()
	if len(algs) > 0 {
		return algs[0]
	}
	return ""
}

// SigningKey returns the signing key of the first validator (deprecated, for backward compatibility)
// Deprecated: Use getValidator() to retrieve algorithm-specific keys
func (c *Config) SigningKey() interface{} {
	algs := c.AvailableAlgorithms()
	if len(algs) > 0 {
		validator, _ := c.getValidator(algs[0])
		return validator.signingKey
	}
	return nil
}

func (c *Config) ClockSkewLeeway() time.Duration {
	return c.clockSkewLeeway
}

func (c *Config) CookieName() string {
	return c.cookieName
}

func (c *Config) RequiredClaims() []string {
	return c.requiredClaims
}

func (c *Config) Logger() *slog.Logger {
	return c.logger
}
