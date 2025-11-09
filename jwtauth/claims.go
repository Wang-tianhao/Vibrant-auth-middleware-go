package jwtauth

import "time"

// Claims represents parsed and validated JWT claims
type Claims struct {
	Subject   string                 // User identifier (sub claim)
	Issuer    string                 // Token issuer (iss claim)
	Audience  string                 // Intended audience (aud claim)
	ExpiresAt time.Time              // Expiration time (exp claim)
	NotBefore time.Time              // Not-before time (nbf claim)
	IssuedAt  time.Time              // Issue time (iat claim)
	JWTID     string                 // JWT ID (jti claim)
	Custom    map[string]interface{} // Custom application-specific claims
}
