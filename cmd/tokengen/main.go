package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	var (
		secret  = flag.String("secret", "your-256-bit-secret-key-min-32-bytes-here-for-demo!", "Secret key (minimum 32 bytes)")
		subject = flag.String("sub", "user123", "Subject (user ID)")
		email   = flag.String("email", "user@example.com", "Email address")
		role    = flag.String("role", "user", "User role")
		hours   = flag.Int("hours", 1, "Token validity in hours")
	)

	flag.Parse()

	if len(*secret) < 32 {
		log.Fatal("Secret must be at least 32 bytes")
	}

	// Create claims
	claims := jwt.MapClaims{
		"sub":   *subject,
		"email": *email,
		"role":  *role,
		"exp":   time.Now().Add(time.Duration(*hours) * time.Hour).Unix(),
		"nbf":   time.Now().Unix(),
		"iat":   time.Now().Unix(),
	}

	// Create and sign token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(*secret))
	if err != nil {
		log.Fatalf("Failed to sign token: %v", err)
	}

	fmt.Println("\n=== JWT Token Generated ===")
	fmt.Printf("\nToken: %s\n\n", tokenString)
	fmt.Println("Claims:")
	fmt.Printf("  Subject: %s\n", *subject)
	fmt.Printf("  Email:   %s\n", *email)
	fmt.Printf("  Role:    %s\n", *role)
	fmt.Printf("  Expires: %s\n\n", time.Now().Add(time.Duration(*hours)*time.Hour).Format(time.RFC3339))
	fmt.Println("Usage:")
	fmt.Printf("  curl -H 'Authorization: Bearer %s' http://localhost:8080/api/profile\n\n", tokenString)
}
