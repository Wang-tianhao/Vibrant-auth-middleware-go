package jwtauth

import (
	"testing"
)

// TestBuildErrorResponse_MessageField tests that buildErrorResponse includes message field for specific errors
func TestBuildErrorResponse_MessageField(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectMessage   bool
		expectedMessage string
	}{
		{
			name: "UNSUPPORTED_ALGORITHM includes message with available algorithms",
			err: NewValidationError(
				ErrUnsupportedAlgorithm,
				"algorithm ES256 not supported (available: HS256, RS256)",
				nil,
			),
			expectMessage:   true,
			expectedMessage: "algorithm ES256 not supported (available: HS256, RS256)",
		},
		{
			name: "MALFORMED_ALGORITHM_HEADER includes message",
			err: NewValidationError(
				ErrMalformedAlgorithmHeader,
				"algorithm header must be a string",
				nil,
			),
			expectMessage:   true,
			expectedMessage: "algorithm header must be a string",
		},
		{
			name: "INVALID_SIGNATURE does not include message",
			err: NewValidationError(
				ErrInvalidSignature,
				"invalid signature",
				nil,
			),
			expectMessage: false,
		},
		{
			name: "EXPIRED does not include message",
			err: NewValidationError(
				ErrExpired,
				"token expired",
				nil,
			),
			expectMessage: false,
		},
		{
			name: "NONE_ALGORITHM does not include message",
			err: NewValidationError(
				ErrNoneAlgorithm,
				"none algorithm not allowed",
				nil,
			),
			expectMessage: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := buildErrorResponse(tt.err)

			// Verify error and reason fields always present
			if response["error"] != "unauthorized" {
				t.Errorf("Expected error=unauthorized, got %v", response["error"])
			}

			reason := getErrorCode(tt.err)
			if response["reason"] != reason {
				t.Errorf("Expected reason=%s, got %v", reason, response["reason"])
			}

			// Verify message field based on error type
			message, hasMessage := response["message"]
			if tt.expectMessage {
				if !hasMessage {
					t.Errorf("Expected message field for %s, but it was missing", tt.name)
				} else if message != tt.expectedMessage {
					t.Errorf("Expected message=%q, got %q", tt.expectedMessage, message)
				}
			} else {
				if hasMessage {
					t.Errorf("Did not expect message field for %s, but got: %v", tt.name, message)
				}
			}
		})
	}
}

// TestBuildErrorResponse_Format tests the overall response format
func TestBuildErrorResponse_Format(t *testing.T) {
	err := NewValidationError(
		ErrUnsupportedAlgorithm,
		"algorithm HS384 not supported (available: HS256, RS256)",
		nil,
	)

	response := buildErrorResponse(err)

	// Verify it returns a map type
	if response == nil {
		t.Fatal("Expected non-nil response")
	}

	// Verify all required fields
	if response["error"] != "unauthorized" {
		t.Errorf("Missing or incorrect 'error' field")
	}
	if response["reason"] != "UNSUPPORTED_ALGORITHM" {
		t.Errorf("Missing or incorrect 'reason' field")
	}
	if response["message"] != "algorithm HS384 not supported (available: HS256, RS256)" {
		t.Errorf("Missing or incorrect 'message' field")
	}
}

// TestErrorCodeSeparation_UnitLevel verifies error codes are distinct
func TestErrorCodeSeparation_UnitLevel(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode string
	}{
		{
			name:         "UNSUPPORTED_ALGORITHM",
			err:          NewValidationError(ErrUnsupportedAlgorithm, "test", nil),
			expectedCode: "UNSUPPORTED_ALGORITHM",
		},
		{
			name:         "INVALID_SIGNATURE",
			err:          NewValidationError(ErrInvalidSignature, "test", nil),
			expectedCode: "INVALID_SIGNATURE",
		},
		{
			name:         "EXPIRED",
			err:          NewValidationError(ErrExpired, "test", nil),
			expectedCode: "EXPIRED",
		},
		{
			name:         "MALFORMED",
			err:          NewValidationError(ErrMalformed, "test", nil),
			expectedCode: "MALFORMED",
		},
		{
			name:         "MALFORMED_ALGORITHM_HEADER",
			err:          NewValidationError(ErrMalformedAlgorithmHeader, "test", nil),
			expectedCode: "MALFORMED_ALGORITHM_HEADER",
		},
		{
			name:         "NONE_ALGORITHM",
			err:          NewValidationError(ErrNoneAlgorithm, "test", nil),
			expectedCode: "NONE_ALGORITHM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := getErrorCode(tt.err)
			if code != tt.expectedCode {
				t.Errorf("Expected code=%s, got code=%s", tt.expectedCode, code)
			}

			// Verify each code is distinct (not overlapping)
			for _, other := range tests {
				if other.name != tt.name {
					otherCode := getErrorCode(other.err)
					if code == otherCode && tt.name != other.name {
						t.Errorf("Error code collision: %s and %s both return %s", tt.name, other.name, code)
					}
				}
			}
		})
	}
}
