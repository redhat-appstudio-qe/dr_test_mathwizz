package main

// This file contains unit tests for JWT middleware functions, rate limiting, and CORS.
// Tests JWT secret initialization, validation logic, rate limiting behavior, and CORS origin validation.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("InitJWTSecret", func() {
	// Store original environment variable to restore after tests
	var originalJWTSecret string

	BeforeEach(func() {
		originalJWTSecret = os.Getenv("JWT_SECRET")
		// Clear jwtSecret variable before each test
		jwtSecret = nil
	})

	AfterEach(func() {
		// Restore original environment variable
		if originalJWTSecret != "" {
			os.Setenv("JWT_SECRET", originalJWTSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
		// Clear jwtSecret variable after each test
		jwtSecret = nil
	})

	When("JWT_SECRET environment variable is valid", func() {
		It("should initialize successfully with exactly 32 characters", func() {
			os.Setenv("JWT_SECRET", "12345678901234567890123456789012") // Exactly 32 chars
			Expect(InitJWTSecret()).Should(Succeed())
			Expect(jwtSecret).ShouldNot(BeNil())
			Expect(jwtSecret).Should(Equal([]byte("12345678901234567890123456789012")))
		})

		It("should initialize successfully with more than 32 characters", func() {
			os.Setenv("JWT_SECRET", "this-is-a-very-secure-secret-key-that-is-longer-than-32-characters")
			Expect(InitJWTSecret()).Should(Succeed())
			Expect(jwtSecret).ShouldNot(BeNil())
			Expect(string(jwtSecret)).Should(Equal("this-is-a-very-secure-secret-key-that-is-longer-than-32-characters"))
		})

		It("should initialize successfully with 64 character base64 string", func() {
			os.Setenv("JWT_SECRET", "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODkwYWJjZGVmZ2hpamts")
			Expect(InitJWTSecret()).Should(Succeed())
			Expect(jwtSecret).ShouldNot(BeNil())
		})
	})

	When("JWT_SECRET environment variable is invalid", func() {
		It("should fail when JWT_SECRET is not set", func() {
			os.Unsetenv("JWT_SECRET")
			err := InitJWTSecret()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("JWT_SECRET environment variable is required"))
			Expect(jwtSecret).Should(BeNil())
		})

		It("should fail when JWT_SECRET is empty string", func() {
			os.Setenv("JWT_SECRET", "")
			err := InitJWTSecret()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("JWT_SECRET environment variable is required"))
			Expect(jwtSecret).Should(BeNil())
		})

		DescribeTable("should fail when JWT_SECRET is too short",
			func(secret string) {
				os.Setenv("JWT_SECRET", secret)
				err := InitJWTSecret()
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("JWT_SECRET must be at least 32 characters long"))
				Expect(jwtSecret).Should(BeNil())
			},
			Entry("1 character", "a"),
			Entry("10 characters", "1234567890"),
			Entry("31 characters (one short)", "1234567890123456789012345678901"),
		)
	})

	When("verifying secret security requirements", func() {
		It("should accept exactly the minimum required length", func() {
			minSecret := "12345678901234567890123456789012" // Exactly 32 chars
			os.Setenv("JWT_SECRET", minSecret)
			Expect(InitJWTSecret()).Should(Succeed())
			Expect(len(jwtSecret)).Should(Equal(32))
		})

		It("should preserve the exact secret value including special characters", func() {
			specialSecret := "!@#$%^&*()_+-=[]{}|;':,.<>?/~`1234567890ABCDEF"
			os.Setenv("JWT_SECRET", specialSecret)
			Expect(InitJWTSecret()).Should(Succeed())
			Expect(string(jwtSecret)).Should(Equal(specialSecret))
		})

		It("should handle unicode characters in secret", func() {
			unicodeSecret := "こんにちは世界-this-is-32-chars-min-12345678"
			os.Setenv("JWT_SECRET", unicodeSecret)
			Expect(InitJWTSecret()).Should(Succeed())
			Expect(string(jwtSecret)).Should(Equal(unicodeSecret))
		})
	})

	When("verifying secret is used for token generation and validation", func() {
		It("should generate different tokens when using different secrets", func() {
			// Generate token with first secret
			os.Setenv("JWT_SECRET", "first-secret-key-at-least-32-characters-long")
			Expect(InitJWTSecret()).Should(Succeed())
			token1, err1 := GenerateToken(123, "user@example.com")
			Expect(err1).ShouldNot(HaveOccurred())

			// Generate token with second secret for same user
			os.Setenv("JWT_SECRET", "second-secret-key-at-least-32-characters-long-different")
			Expect(InitJWTSecret()).Should(Succeed())
			token2, err2 := GenerateToken(123, "user@example.com")
			Expect(err2).ShouldNot(HaveOccurred())

			// Tokens should be different even for same user data
			Expect(token1).ShouldNot(Equal(token2))
		})

		It("should fail validation when token signed with different secret", func() {
			// Generate token with first secret
			os.Setenv("JWT_SECRET", "original-secret-key-at-least-32-characters-long")
			Expect(InitJWTSecret()).Should(Succeed())
			token, err := GenerateToken(123, "user@example.com")
			Expect(err).ShouldNot(HaveOccurred())

			// Try to validate with different secret
			os.Setenv("JWT_SECRET", "wrong-secret-key-at-least-32-characters-long-different")
			Expect(InitJWTSecret()).Should(Succeed())
			claims, err := ValidateToken(token)

			// Validation should fail with wrong secret
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to parse token"))
			Expect(claims).Should(BeNil())
		})

		It("should successfully validate token when using same secret", func() {
			// Generate and validate token with same secret
			secret := "consistent-secret-key-at-least-32-characters-long"
			os.Setenv("JWT_SECRET", secret)
			Expect(InitJWTSecret()).Should(Succeed())

			userID := 456
			email := "test@example.com"
			token, err := GenerateToken(userID, email)
			Expect(err).ShouldNot(HaveOccurred())

			// Validation should succeed with same secret
			claims, err := ValidateToken(token)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(claims).ShouldNot(BeNil())
			Expect(claims.UserID).Should(Equal(userID))
			Expect(claims.Email).Should(Equal(email))
		})

		It("should prevent token forgery by rejecting tokens from different secret", func() {
			// Attacker has their own secret and tries to create token
			os.Setenv("JWT_SECRET", "attacker-secret-key-at-least-32-characters-long")
			Expect(InitJWTSecret()).Should(Succeed())
			forgedToken, err := GenerateToken(999, "admin@example.com")
			Expect(err).ShouldNot(HaveOccurred())

			// Server uses legitimate secret
			os.Setenv("JWT_SECRET", "legitimate-server-secret-at-least-32-characters")
			Expect(InitJWTSecret()).Should(Succeed())

			// Server should reject forged token
			claims, err := ValidateToken(forgedToken)
			Expect(err).Should(HaveOccurred())
			Expect(claims).Should(BeNil())
		})
	})
})

var _ = Describe("GenerateToken", func() {
	BeforeEach(func() {
		// Initialize JWT secret before each test
		os.Setenv("JWT_SECRET", "test-secret-key-at-least-32-characters-long")
		InitJWTSecret()
	})

	When("generating tokens with valid inputs", func() {
		It("should successfully generate token with valid userID and email", func() {
			token, err := GenerateToken(123, "user@example.com")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(token).ShouldNot(BeEmpty())
			// JWT format: header.payload.signature (3 parts separated by dots)
			Expect(token).Should(MatchRegexp(`^[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+$`))
		})

		It("should generate token with 24-hour expiration", func() {
			token, err := GenerateToken(456, "test@example.com")
			Expect(err).ShouldNot(HaveOccurred())

			// Validate and extract claims
			claims, err := ValidateToken(token)
			Expect(err).ShouldNot(HaveOccurred())

			// Verify expiration is approximately 24 hours from now
			expectedExpiration := time.Now().Add(24 * time.Hour)
			actualExpiration := claims.ExpiresAt.Time
			timeDiff := actualExpiration.Sub(expectedExpiration)
			Expect(timeDiff).Should(BeNumerically("<", 5*time.Second)) // Within 5 seconds tolerance
		})

		It("should include userID and email in token claims", func() {
			userID := 789
			email := "claim@example.com"

			token, err := GenerateToken(userID, email)
			Expect(err).ShouldNot(HaveOccurred())

			claims, err := ValidateToken(token)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(claims.UserID).Should(Equal(userID))
			Expect(claims.Email).Should(Equal(email))
		})

		It("should generate unique tokens for same user at different times", func() {
			userID := 100
			email := "same@example.com"

			token1, err1 := GenerateToken(userID, email)
			Expect(err1).ShouldNot(HaveOccurred())

			// Wait 1+ second to ensure different IssuedAt timestamp (timestamps are in seconds)
			time.Sleep(1100 * time.Millisecond)

			token2, err2 := GenerateToken(userID, email)
			Expect(err2).ShouldNot(HaveOccurred())

			// Tokens should be different due to different timestamps
			Expect(token1).ShouldNot(Equal(token2))
		})
	})

	When("generating tokens with edge case inputs", func() {
		It("should handle userID of zero", func() {
			token, err := GenerateToken(0, "zero@example.com")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(token).ShouldNot(BeEmpty())

			claims, err := ValidateToken(token)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(claims.UserID).Should(Equal(0))
		})

		It("should handle negative userID", func() {
			token, err := GenerateToken(-1, "negative@example.com")
			Expect(err).ShouldNot(HaveOccurred())

			claims, err := ValidateToken(token)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(claims.UserID).Should(Equal(-1))
		})

		It("should handle empty email string", func() {
			token, err := GenerateToken(123, "")
			Expect(err).ShouldNot(HaveOccurred())

			claims, err := ValidateToken(token)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(claims.Email).Should(Equal(""))
		})

		It("should handle very long email addresses", func() {
			longEmail := "very.long.email.address.with.many.characters.and.subdomains@some.very.long.domain.name.example.com"
			token, err := GenerateToken(999, longEmail)
			Expect(err).ShouldNot(HaveOccurred())

			claims, err := ValidateToken(token)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(claims.Email).Should(Equal(longEmail))
		})

		It("should handle special characters in email", func() {
			specialEmail := "user+tag@example.com"
			token, err := GenerateToken(111, specialEmail)
			Expect(err).ShouldNot(HaveOccurred())

			claims, err := ValidateToken(token)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(claims.Email).Should(Equal(specialEmail))
		})

		It("should handle very large userID values", func() {
			largeUserID := 2147483647 // Max int32
			token, err := GenerateToken(largeUserID, "large@example.com")
			Expect(err).ShouldNot(HaveOccurred())

			claims, err := ValidateToken(token)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(claims.UserID).Should(Equal(largeUserID))
		})
	})

	When("verifying token signing", func() {
		It("should use HS256 signing algorithm", func() {
			token, err := GenerateToken(123, "user@example.com")
			Expect(err).ShouldNot(HaveOccurred())

			// Parse token without validation to inspect header
			parsedToken, _, err := jwt.NewParser().ParseUnverified(token, &Claims{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(parsedToken.Method.Alg()).Should(Equal("HS256"))
		})

		It("should produce valid signature that can be verified", func() {
			token, err := GenerateToken(456, "verify@example.com")
			Expect(err).ShouldNot(HaveOccurred())

			// Validation succeeds with correct secret
			claims, err := ValidateToken(token)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(claims).ShouldNot(BeNil())
		})
	})
})

var _ = Describe("ValidateToken", func() {
	var validToken string
	var userID int
	var email string

	BeforeEach(func() {
		// Initialize JWT secret
		os.Setenv("JWT_SECRET", "test-secret-key-at-least-32-characters-long")
		InitJWTSecret()

		// Generate valid token for testing
		userID = 123
		email = "test@example.com"
		var err error
		validToken, err = GenerateToken(userID, email)
		Expect(err).ShouldNot(HaveOccurred())
	})

	When("validating tokens with valid format and signature", func() {
		It("should successfully validate valid token", func() {
			claims, err := ValidateToken(validToken)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(claims).ShouldNot(BeNil())
			Expect(claims.UserID).Should(Equal(userID))
			Expect(claims.Email).Should(Equal(email))
		})

		It("should extract correct claims from valid token", func() {
			claims, err := ValidateToken(validToken)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(claims.UserID).Should(Equal(123))
			Expect(claims.Email).Should(Equal("test@example.com"))
			Expect(claims.ExpiresAt).ShouldNot(BeNil())
			Expect(claims.IssuedAt).ShouldNot(BeNil())
		})

		It("should validate token is not expired", func() {
			claims, err := ValidateToken(validToken)
			Expect(err).ShouldNot(HaveOccurred())

			// Expiration should be in the future
			Expect(claims.ExpiresAt.Time).Should(BeTemporally(">", time.Now()))
		})

		It("should validate token issued at time is in the past", func() {
			claims, err := ValidateToken(validToken)
			Expect(err).ShouldNot(HaveOccurred())

			// IssuedAt should be in the past or now
			Expect(claims.IssuedAt.Time).Should(BeTemporally("<=", time.Now().Add(1*time.Second)))
		})
	})

	When("validating tokens with invalid format", func() {
		It("should reject empty token string", func() {
			Expect(ValidateToken("")).Error().Should(HaveOccurred())
		})

		It("should reject malformed token with only 2 parts", func() {
			malformedToken := "header.payload" // Missing signature
			claims, err := ValidateToken(malformedToken)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to parse token"))
			Expect(claims).Should(BeNil())
		})

		It("should reject token with invalid base64 encoding", func() {
			invalidToken := "invalid!!!.base64!!!.encoding!!!"
			claims, err := ValidateToken(invalidToken)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to parse token"))
			Expect(claims).Should(BeNil())
		})

		It("should reject token with corrupted header", func() {
			// Replace first part of token with invalid base64
			parts := strings.Split(validToken, ".")
			if len(parts) == 3 {
				corruptedToken := "XXXXX." + parts[1] + "." + parts[2]
				claims, err := ValidateToken(corruptedToken)
				Expect(err).Should(HaveOccurred())
				Expect(claims).Should(BeNil())
			}
		})

		It("should reject random string as token", func() {
			claims, err := ValidateToken("this-is-not-a-jwt-token")
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to parse token"))
			Expect(claims).Should(BeNil())
		})
	})

	When("validating tokens with invalid signature", func() {
		It("should reject token signed with different secret", func() {
			// Generate token with different secret
			os.Setenv("JWT_SECRET", "different-secret-key-at-least-32-characters")
			InitJWTSecret()
			differentSecretToken, _ := GenerateToken(999, "attacker@example.com")

			// Try to validate with original secret
			os.Setenv("JWT_SECRET", "test-secret-key-at-least-32-characters-long")
			InitJWTSecret()

			claims, err := ValidateToken(differentSecretToken)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to parse token"))
			Expect(claims).Should(BeNil())
		})

		It("should reject token with tampered payload", func() {
			// Split token and modify payload
			parts := strings.Split(validToken, ".")
			if len(parts) == 3 {
				// Change one character in the payload
				tamperedPayload := parts[1][:len(parts[1])-1] + "X"
				tamperedToken := parts[0] + "." + tamperedPayload + "." + parts[2]

				claims, err := ValidateToken(tamperedToken)
				Expect(err).Should(HaveOccurred())
				Expect(claims).Should(BeNil())
			}
		})

		It("should reject token with tampered signature", func() {
			// Split token and completely replace signature
			parts := strings.Split(validToken, ".")
			Expect(parts).Should(HaveLen(3))

			// Replace signature with obviously invalid value
			tamperedToken := parts[0] + "." + parts[1] + ".completely-invalid-signature"

			claims, err := ValidateToken(tamperedToken)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to parse token"))
			Expect(claims).Should(BeNil())
		})
	})

	When("validating expired tokens", func() {
		It("should reject token that expired 1 hour ago", func() {
			// Create token that expired 1 hour ago
			claims := Claims{
				UserID: 123,
				Email:  "expired@example.com",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now().Add(-25 * time.Hour)),
				},
			}
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			expiredToken, _ := token.SignedString(jwtSecret)

			validatedClaims, err := ValidateToken(expiredToken)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to parse token"))
			Expect(validatedClaims).Should(BeNil())
		})

		It("should reject token that expired just now", func() {
			// Create token that expired 1 second ago
			claims := Claims{
				UserID: 456,
				Email:  "justexpired@example.com",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Second)),
					IssuedAt:  jwt.NewNumericDate(time.Now().Add(-24*time.Hour - 1*time.Second)),
				},
			}
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			expiredToken, _ := token.SignedString(jwtSecret)

			validatedClaims, err := ValidateToken(expiredToken)
			Expect(err).Should(HaveOccurred())
			Expect(validatedClaims).Should(BeNil())
		})

		It("should accept token that expires in 1 second (not yet expired)", func() {
			// Create token that will expire in 1 second
			claims := Claims{
				UserID: 789,
				Email:  "almostexpired@example.com",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Second)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
				},
			}
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			almostExpiredToken, _ := token.SignedString(jwtSecret)

			validatedClaims, err := ValidateToken(almostExpiredToken)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(validatedClaims).ShouldNot(BeNil())
			Expect(validatedClaims.UserID).Should(Equal(789))
		})
	})

	When("validating tokens with wrong signing algorithm", func() {
		It("should reject token signed with RS256 instead of HS256", func() {
			// Note: This test documents expected behavior
			// In practice, creating RS256 token requires RSA key pair
			// ValidateToken should reject non-HMAC signing methods

			// For now, we verify the algorithm check exists by checking error message
			// when wrong algorithm is detected
			claims := Claims{
				UserID: 123,
				Email:  "test@example.com",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
				},
			}

			// Try to sign with "none" algorithm (which should be rejected)
			token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
			noneToken, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

			validatedClaims, err := ValidateToken(noneToken)
			Expect(err).Should(HaveOccurred())
			Expect(validatedClaims).Should(BeNil())
		})
	})

})

var _ = Describe("GetUserIDFromContext", func() {
	When("extracting userID from context with valid value", func() {
		It("should successfully extract userID from context", func() {
			userID := 123
			ctx := context.WithValue(context.Background(), userIDKey, userID)

			extractedID, err := GetUserIDFromContext(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(extractedID).Should(Equal(userID))
		})

		It("should extract userID of zero", func() {
			ctx := context.WithValue(context.Background(), userIDKey, 0)

			extractedID, err := GetUserIDFromContext(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(extractedID).Should(Equal(0))
		})

		It("should extract negative userID", func() {
			ctx := context.WithValue(context.Background(), userIDKey, -1)

			extractedID, err := GetUserIDFromContext(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(extractedID).Should(Equal(-1))
		})

		It("should extract very large userID", func() {
			largeUserID := 2147483647 // Max int32
			ctx := context.WithValue(context.Background(), userIDKey, largeUserID)

			extractedID, err := GetUserIDFromContext(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(extractedID).Should(Equal(largeUserID))
		})
	})

	When("extracting userID from context with missing or invalid value", func() {
		It("should return error when userID not in context", func() {
			ctx := context.Background()

			extractedID, err := GetUserIDFromContext(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("user ID not found in context"))
			Expect(extractedID).Should(Equal(0))
		})

		It("should return error when value has wrong type (string instead of int)", func() {
			ctx := context.WithValue(context.Background(), userIDKey, "123")

			extractedID, err := GetUserIDFromContext(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("user ID not found in context"))
			Expect(extractedID).Should(Equal(0))
		})

		It("should return error when value has wrong type (float64 instead of int)", func() {
			ctx := context.WithValue(context.Background(), userIDKey, 123.45)

			extractedID, err := GetUserIDFromContext(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("user ID not found in context"))
			Expect(extractedID).Should(Equal(0))
		})

		It("should return error when value has wrong type (bool instead of int)", func() {
			ctx := context.WithValue(context.Background(), userIDKey, true)

			extractedID, err := GetUserIDFromContext(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("user ID not found in context"))
			Expect(extractedID).Should(Equal(0))
		})

		It("should return error when value is nil", func() {
			ctx := context.WithValue(context.Background(), userIDKey, nil)

			extractedID, err := GetUserIDFromContext(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("user ID not found in context"))
			Expect(extractedID).Should(Equal(0))
		})
	})

	When("verifying context key isolation", func() {
		It("should not extract userID from different context key", func() {
			// Use a different key (not userIDKey)
			differentKey := contextKey("differentKey")
			ctx := context.WithValue(context.Background(), differentKey, 123)

			extractedID, err := GetUserIDFromContext(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("user ID not found in context"))
			Expect(extractedID).Should(Equal(0))
		})

		It("should not extract userID from string key with same value", func() {
			// Use plain string "userID" instead of contextKey type
			ctx := context.WithValue(context.Background(), "userID", 123)

			extractedID, err := GetUserIDFromContext(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("user ID not found in context"))
			Expect(extractedID).Should(Equal(0))
		})
	})

	When("integrating with AuthMiddleware workflow", func() {
		It("should successfully extract userID set by AuthMiddleware", func() {
			// Simulate what AuthMiddleware does
			originalCtx := context.Background()
			userID := 456
			modifiedCtx := context.WithValue(originalCtx, userIDKey, userID)

			// Handler should be able to extract the userID
			extractedID, err := GetUserIDFromContext(modifiedCtx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(extractedID).Should(Equal(userID))
		})
	})
})

var _ = Describe("RateLimitMiddleware", func() {
	var (
		rateLimiter *RateLimiter
		handler     http.HandlerFunc
		recorder    *httptest.ResponseRecorder
	)

	// Helper function to create a test handler that always returns 200 OK
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}

	BeforeEach(func() {
		recorder = httptest.NewRecorder()
	})

	When("enforcing rate limits on authentication endpoints", func() {
		BeforeEach(func() {
			// Create rate limiter: 5 requests per minute (5/60 tokens/sec), burst of 5
			rateLimiter = NewRateLimiter(5.0/60.0, 5)
			handler = RateLimitMiddleware(rateLimiter)(testHandler)
		})

		It("should allow requests under the rate limit", func() {
			// Make 5 requests (all should succeed)
			for i := 0; i < 5; i++ {
				req := httptest.NewRequest("POST", "/login", nil)
				req.RemoteAddr = "192.168.1.100:12345"
				recorder = httptest.NewRecorder()
				handler(recorder, req)
				Expect(recorder.Code).Should(Equal(http.StatusOK))
				Expect(recorder.Body.String()).Should(Equal("success"))
			}
		})

		It("should block requests exceeding the rate limit", func() {
			// Make 5 requests (should succeed - burst)
			for i := 0; i < 5; i++ {
				req := httptest.NewRequest("POST", "/login", nil)
				req.RemoteAddr = "192.168.1.100:12345"
				recorder = httptest.NewRecorder()
				handler(recorder, req)
				Expect(recorder.Code).Should(Equal(http.StatusOK))
			}

			// 6th request should be rate limited
			req := httptest.NewRequest("POST", "/login", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			recorder = httptest.NewRecorder()
			handler(recorder, req)
			Expect(recorder.Code).Should(Equal(http.StatusTooManyRequests))
			Expect(recorder.Body.String()).Should(ContainSubstring("rate limit exceeded"))
		})

		It("should return HTTP 429 when rate limit exceeded", func() {
			// Consume all tokens
			for i := 0; i < 5; i++ {
				req := httptest.NewRequest("POST", "/login", nil)
				req.RemoteAddr = "192.168.1.100:12345"
				recorder = httptest.NewRecorder()
				handler(recorder, req)
			}

			// Next request should get 429
			req := httptest.NewRequest("POST", "/login", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			recorder = httptest.NewRecorder()
			handler(recorder, req)
			Expect(recorder.Code).Should(Equal(429))
		})
	})

	When("isolating rate limits per IP address", func() {
		BeforeEach(func() {
			// Create rate limiter: 3 requests per minute, burst of 3 (smaller for easier testing)
			rateLimiter = NewRateLimiter(3.0/60.0, 3)
			handler = RateLimitMiddleware(rateLimiter)(testHandler)
		})

		It("should maintain separate rate limits for different IP addresses", func() {
			// IP 1: Consume all tokens (3 requests)
			for i := 0; i < 3; i++ {
				req := httptest.NewRequest("POST", "/login", nil)
				req.RemoteAddr = "192.168.1.100:12345"
				recorder = httptest.NewRecorder()
				handler(recorder, req)
				Expect(recorder.Code).Should(Equal(http.StatusOK))
			}

			// IP 1: 4th request should be blocked
			req1 := httptest.NewRequest("POST", "/login", nil)
			req1.RemoteAddr = "192.168.1.100:12345"
			recorder1 := httptest.NewRecorder()
			handler(recorder1, req1)
			Expect(recorder1.Code).Should(Equal(http.StatusTooManyRequests))

			// IP 2: Should have independent rate limit (first 3 requests succeed)
			for i := 0; i < 3; i++ {
				req := httptest.NewRequest("POST", "/login", nil)
				req.RemoteAddr = "192.168.1.200:54321"
				recorder = httptest.NewRecorder()
				handler(recorder, req)
				Expect(recorder.Code).Should(Equal(http.StatusOK))
			}

			// IP 2: 4th request should be blocked
			req2 := httptest.NewRequest("POST", "/login", nil)
			req2.RemoteAddr = "192.168.1.200:54321"
			recorder2 := httptest.NewRecorder()
			handler(recorder2, req2)
			Expect(recorder2.Code).Should(Equal(http.StatusTooManyRequests))
		})

		It("should not affect other IPs when one IP is rate limited", func() {
			// Exhaust rate limit for IP1
			for i := 0; i < 4; i++ {
				req := httptest.NewRequest("POST", "/login", nil)
				req.RemoteAddr = "10.0.0.1:8080"
				recorder = httptest.NewRecorder()
				handler(recorder, req)
			}

			// IP2 should still work
			req := httptest.NewRequest("POST", "/login", nil)
			req.RemoteAddr = "10.0.0.2:8080"
			recorder = httptest.NewRecorder()
			handler(recorder, req)
			Expect(recorder.Code).Should(Equal(http.StatusOK))
		})
	})

	When("handling IP address extraction", func() {
		BeforeEach(func() {
			rateLimiter = NewRateLimiter(2.0/60.0, 2)
			handler = RateLimitMiddleware(rateLimiter)(testHandler)
		})

		It("should strip port number from IPv4 address", func() {
			// Both requests from same IP, different ports should share rate limit
			req1 := httptest.NewRequest("POST", "/login", nil)
			req1.RemoteAddr = "192.168.1.100:12345"
			recorder1 := httptest.NewRecorder()
			handler(recorder1, req1)
			Expect(recorder1.Code).Should(Equal(http.StatusOK))

			req2 := httptest.NewRequest("POST", "/login", nil)
			req2.RemoteAddr = "192.168.1.100:54321" // Different port, same IP
			recorder2 := httptest.NewRecorder()
			handler(recorder2, req2)
			Expect(recorder2.Code).Should(Equal(http.StatusOK))

			// 3rd request should be rate limited (same IP)
			req3 := httptest.NewRequest("POST", "/login", nil)
			req3.RemoteAddr = "192.168.1.100:99999"
			recorder3 := httptest.NewRecorder()
			handler(recorder3, req3)
			Expect(recorder3.Code).Should(Equal(http.StatusTooManyRequests))
		})

		It("should handle IPv6 addresses", func() {
			req1 := httptest.NewRequest("POST", "/login", nil)
			req1.RemoteAddr = "[2001:db8::1]:8080"
			recorder1 := httptest.NewRecorder()
			handler(recorder1, req1)
			Expect(recorder1.Code).Should(Equal(http.StatusOK))

			req2 := httptest.NewRequest("POST", "/login", nil)
			req2.RemoteAddr = "[2001:db8::1]:9090" // Same IPv6, different port
			recorder2 := httptest.NewRecorder()
			handler(recorder2, req2)
			Expect(recorder2.Code).Should(Equal(http.StatusOK))

			// 3rd request should be rate limited
			req3 := httptest.NewRequest("POST", "/login", nil)
			req3.RemoteAddr = "[2001:db8::1]:1234"
			recorder3 := httptest.NewRecorder()
			handler(recorder3, req3)
			Expect(recorder3.Code).Should(Equal(http.StatusTooManyRequests))
		})

		It("should handle RemoteAddr without port gracefully", func() {
			// Some configurations might provide just IP without port
			req := httptest.NewRequest("POST", "/login", nil)
			req.RemoteAddr = "192.168.1.100" // No port
			recorder = httptest.NewRecorder()
			handler(recorder, req)
			Expect(recorder.Code).Should(Equal(http.StatusOK))
		})
	})

	When("testing burst behavior", func() {
		It("should allow burst of requests immediately", func() {
			// Create rate limiter with burst of 10
			rateLimiter = NewRateLimiter(10.0/60.0, 10)
			handler = RateLimitMiddleware(rateLimiter)(testHandler)

			// All 10 requests should succeed immediately (burst)
			for i := 0; i < 10; i++ {
				req := httptest.NewRequest("POST", "/register", nil)
				req.RemoteAddr = "203.0.113.50:8080"
				recorder = httptest.NewRecorder()
				handler(recorder, req)
				Expect(recorder.Code).Should(Equal(http.StatusOK), "Request %d should succeed", i+1)
			}

			// 11th request should be blocked
			req := httptest.NewRequest("POST", "/register", nil)
			req.RemoteAddr = "203.0.113.50:8080"
			recorder = httptest.NewRecorder()
			handler(recorder, req)
			Expect(recorder.Code).Should(Equal(http.StatusTooManyRequests))
		})

		It("should enforce long-term rate after burst consumed", func() {
			// Create rate limiter: 2 requests per minute, burst of 3
			rateLimiter = NewRateLimiter(2.0/60.0, 3)
			handler = RateLimitMiddleware(rateLimiter)(testHandler)

			// Burst: First 3 requests succeed immediately
			for i := 0; i < 3; i++ {
				req := httptest.NewRequest("POST", "/login", nil)
				req.RemoteAddr = "10.20.30.40:5555"
				recorder = httptest.NewRecorder()
				handler(recorder, req)
				Expect(recorder.Code).Should(Equal(http.StatusOK))
			}

			// 4th request immediately after should be blocked
			req := httptest.NewRequest("POST", "/login", nil)
			req.RemoteAddr = "10.20.30.40:5555"
			recorder = httptest.NewRecorder()
			handler(recorder, req)
			Expect(recorder.Code).Should(Equal(http.StatusTooManyRequests))
		})
	})

	When("testing token refill over time", func() {
		It("should allow requests after tokens refill", func() {
			// Create rate limiter: 60 requests per minute (1 token/sec), burst of 2
			rateLimiter = NewRateLimiter(60.0/60.0, 2) // 1 token per second
			handler = RateLimitMiddleware(rateLimiter)(testHandler)

			// Consume burst (2 requests)
			for i := 0; i < 2; i++ {
				req := httptest.NewRequest("POST", "/login", nil)
				req.RemoteAddr = "172.16.0.1:8080"
				recorder = httptest.NewRecorder()
				handler(recorder, req)
				Expect(recorder.Code).Should(Equal(http.StatusOK))
			}

			// 3rd request should be blocked
			req1 := httptest.NewRequest("POST", "/login", nil)
			req1.RemoteAddr = "172.16.0.1:8080"
			recorder1 := httptest.NewRecorder()
			handler(recorder1, req1)
			Expect(recorder1.Code).Should(Equal(http.StatusTooManyRequests))

			// Wait for token refill (1 second = 1 token)
			time.Sleep(1100 * time.Millisecond)

			// 4th request should succeed (token refilled)
			req2 := httptest.NewRequest("POST", "/login", nil)
			req2.RemoteAddr = "172.16.0.1:8080"
			recorder2 := httptest.NewRecorder()
			handler(recorder2, req2)
			Expect(recorder2.Code).Should(Equal(http.StatusOK))
		})
	})

	When("preventing brute force attack scenarios", func() {
		BeforeEach(func() {
			// Realistic rate limit: 5 requests per minute
			rateLimiter = NewRateLimiter(5.0/60.0, 5)
			handler = RateLimitMiddleware(rateLimiter)(testHandler)
		})

		It("should prevent password brute force by blocking rapid login attempts", func() {
			attackerIP := "198.51.100.42:6666"

			// Attacker tries 10 passwords rapidly
			successCount := 0
			blockedCount := 0

			for i := 0; i < 10; i++ {
				req := httptest.NewRequest("POST", "/login", nil)
				req.RemoteAddr = attackerIP
				recorder = httptest.NewRecorder()
				handler(recorder, req)

				if recorder.Code == http.StatusOK {
					successCount++
				} else if recorder.Code == http.StatusTooManyRequests {
					blockedCount++
				}
			}

			// First 5 attempts allowed (burst), next 5 blocked
			Expect(successCount).Should(Equal(5))
			Expect(blockedCount).Should(Equal(5))
		})

		It("should prevent user enumeration via registration endpoint", func() {
			// Attacker tries to enumerate users by testing many email addresses
			attackerIP := "203.0.113.100:7777"

			// Try to register 10 different emails
			for i := 0; i < 5; i++ {
				req := httptest.NewRequest("POST", "/register", nil)
				req.RemoteAddr = attackerIP
				recorder = httptest.NewRecorder()
				handler(recorder, req)
				Expect(recorder.Code).Should(Equal(http.StatusOK))
			}

			// 6th attempt should be blocked
			req := httptest.NewRequest("POST", "/register", nil)
			req.RemoteAddr = attackerIP
			recorder = httptest.NewRecorder()
			handler(recorder, req)
			Expect(recorder.Code).Should(Equal(http.StatusTooManyRequests))
		})

		It("should prevent account creation spam", func() {
			// Attacker tries to create many fake accounts
			spammerIP := "192.0.2.50:8888"

			// Rapid account creation attempts
			allowedRequests := 0
			for i := 0; i < 20; i++ {
				req := httptest.NewRequest("POST", "/register", nil)
				req.RemoteAddr = spammerIP
				recorder = httptest.NewRecorder()
				handler(recorder, req)

				if recorder.Code == http.StatusOK {
					allowedRequests++
				}
			}

			// Only first 5 should be allowed (burst), rest blocked
			Expect(allowedRequests).Should(Equal(5))
		})
	})

	When("testing edge cases", func() {
		BeforeEach(func() {
			rateLimiter = NewRateLimiter(5.0/60.0, 5)
			handler = RateLimitMiddleware(rateLimiter)(testHandler)
		})

		It("should handle empty RemoteAddr gracefully", func() {
			req := httptest.NewRequest("POST", "/login", nil)
			req.RemoteAddr = "" // Empty address
			recorder = httptest.NewRecorder()
			handler(recorder, req)
			// Should not panic, should either succeed or fail gracefully
			Expect(recorder.Code).Should(BeNumerically(">=", 200))
		})

		It("should handle malformed RemoteAddr gracefully", func() {
			req := httptest.NewRequest("POST", "/login", nil)
			req.RemoteAddr = "not-a-valid-ip" // Malformed
			recorder = httptest.NewRecorder()
			handler(recorder, req)
			// Should not panic
			Expect(recorder.Code).Should(BeNumerically(">=", 200))
		})
	})
})

var _ = Describe("corsMiddleware", func() {
	var (
		testHandler     http.Handler
		recorder        *httptest.ResponseRecorder
		originalOrigins string
	)

	// Simple test handler that returns 200 OK
	simpleHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	BeforeEach(func() {
		recorder = httptest.NewRecorder()
		// Save original ALLOWED_ORIGINS environment variable
		originalOrigins = os.Getenv("ALLOWED_ORIGINS")
	})

	AfterEach(func() {
		// Restore original ALLOWED_ORIGINS
		if originalOrigins != "" {
			os.Setenv("ALLOWED_ORIGINS", originalOrigins)
		} else {
			os.Unsetenv("ALLOWED_ORIGINS")
		}
	})

	When("allowing valid origins", func() {
		It("should allow requests from whitelisted origin", func() {
			os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000")
			testHandler = corsMiddleware(simpleHandler)

			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", "http://localhost:3000")

			testHandler.ServeHTTP(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusOK))
			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).Should(Equal("http://localhost:3000"))
		})

		It("should allow requests from multiple whitelisted origins", func() {
			os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000,https://mathwizz.com,https://www.mathwizz.com")
			testHandler = corsMiddleware(simpleHandler)

			// Test first origin
			req1 := httptest.NewRequest("GET", "/api/test", nil)
			req1.Header.Set("Origin", "http://localhost:3000")
			recorder1 := httptest.NewRecorder()
			testHandler.ServeHTTP(recorder1, req1)
			Expect(recorder1.Header().Get("Access-Control-Allow-Origin")).Should(Equal("http://localhost:3000"))

			// Test second origin
			req2 := httptest.NewRequest("GET", "/api/test", nil)
			req2.Header.Set("Origin", "https://mathwizz.com")
			recorder2 := httptest.NewRecorder()
			testHandler.ServeHTTP(recorder2, req2)
			Expect(recorder2.Header().Get("Access-Control-Allow-Origin")).Should(Equal("https://mathwizz.com"))

			// Test third origin
			req3 := httptest.NewRequest("GET", "/api/test", nil)
			req3.Header.Set("Origin", "https://www.mathwizz.com")
			recorder3 := httptest.NewRecorder()
			testHandler.ServeHTTP(recorder3, req3)
			Expect(recorder3.Header().Get("Access-Control-Allow-Origin")).Should(Equal("https://www.mathwizz.com"))
		})

		It("should use default origin when ALLOWED_ORIGINS not set", func() {
			os.Unsetenv("ALLOWED_ORIGINS")
			testHandler = corsMiddleware(simpleHandler)

			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", "http://localhost:3000")

			testHandler.ServeHTTP(recorder, req)

			// Default is http://localhost:3000
			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).Should(Equal("http://localhost:3000"))
		})

		It("should handle whitespace in ALLOWED_ORIGINS configuration", func() {
			// User might add spaces after commas
			os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000, https://mathwizz.com,  https://app.mathwizz.com")
			testHandler = corsMiddleware(simpleHandler)

			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", "https://mathwizz.com")

			testHandler.ServeHTTP(recorder, req)

			// Should still match despite spaces in config
			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).Should(Equal("https://mathwizz.com"))
		})
	})

	When("blocking invalid origins", func() {
		BeforeEach(func() {
			os.Setenv("ALLOWED_ORIGINS", "https://mathwizz.com")
			testHandler = corsMiddleware(simpleHandler)
		})

		It("should not set CORS header for non-whitelisted origin", func() {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", "https://evil.com")

			testHandler.ServeHTTP(recorder, req)

			// Access-Control-Allow-Origin should NOT be set
			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).Should(BeEmpty())
			// But request should still succeed (browser will block it)
			Expect(recorder.Code).Should(Equal(http.StatusOK))
		})

		It("should reject attacker origin attempting CSRF", func() {
			req := httptest.NewRequest("POST", "/api/solve", nil)
			req.Header.Set("Origin", "https://attacker.evil.com")

			testHandler.ServeHTTP(recorder, req)

			// No CORS header means browser will block the response
			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).Should(BeEmpty())
		})

		It("should not use wildcard (*) for any origin", func() {
			// Even if no Origin header is provided
			req := httptest.NewRequest("GET", "/api/test", nil)
			// No Origin header set

			testHandler.ServeHTTP(recorder, req)

			// Should never see wildcard
			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).ShouldNot(Equal("*"))
		})
	})

	When("handling preflight OPTIONS requests", func() {
		BeforeEach(func() {
			os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000")
			testHandler = corsMiddleware(simpleHandler)
		})

		It("should respond to OPTIONS preflight request", func() {
			req := httptest.NewRequest("OPTIONS", "/api/solve", nil)
			req.Header.Set("Origin", "http://localhost:3000")

			testHandler.ServeHTTP(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusOK))
			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).Should(Equal("http://localhost:3000"))
			Expect(recorder.Header().Get("Access-Control-Allow-Methods")).Should(ContainSubstring("GET"))
			Expect(recorder.Header().Get("Access-Control-Allow-Methods")).Should(ContainSubstring("POST"))
			Expect(recorder.Header().Get("Access-Control-Allow-Headers")).Should(ContainSubstring("Authorization"))
		})

		It("should include allowed methods in preflight response", func() {
			req := httptest.NewRequest("OPTIONS", "/api/test", nil)
			req.Header.Set("Origin", "http://localhost:3000")

			testHandler.ServeHTTP(recorder, req)

			allowedMethods := recorder.Header().Get("Access-Control-Allow-Methods")
			Expect(allowedMethods).Should(ContainSubstring("GET"))
			Expect(allowedMethods).Should(ContainSubstring("POST"))
			Expect(allowedMethods).Should(ContainSubstring("PUT"))
			Expect(allowedMethods).Should(ContainSubstring("DELETE"))
			Expect(allowedMethods).Should(ContainSubstring("OPTIONS"))
		})

		It("should include allowed headers in preflight response", func() {
			req := httptest.NewRequest("OPTIONS", "/api/test", nil)
			req.Header.Set("Origin", "http://localhost:3000")

			testHandler.ServeHTTP(recorder, req)

			allowedHeaders := recorder.Header().Get("Access-Control-Allow-Headers")
			Expect(allowedHeaders).Should(ContainSubstring("Content-Type"))
			Expect(allowedHeaders).Should(ContainSubstring("Authorization"))
		})
	})

	When("adding security headers", func() {
		BeforeEach(func() {
			os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000")
			testHandler = corsMiddleware(simpleHandler)
		})

		It("should set X-Frame-Options to prevent clickjacking", func() {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", "http://localhost:3000")

			testHandler.ServeHTTP(recorder, req)

			Expect(recorder.Header().Get("X-Frame-Options")).Should(Equal("DENY"))
		})

		It("should set X-Content-Type-Options to prevent MIME sniffing", func() {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", "http://localhost:3000")

			testHandler.ServeHTTP(recorder, req)

			Expect(recorder.Header().Get("X-Content-Type-Options")).Should(Equal("nosniff"))
		})

		It("should set X-XSS-Protection to enable XSS filter", func() {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", "http://localhost:3000")

			testHandler.ServeHTTP(recorder, req)

			Expect(recorder.Header().Get("X-XSS-Protection")).Should(Equal("1; mode=block"))
		})

		It("should add security headers even for blocked origins", func() {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", "https://evil.com")

			testHandler.ServeHTTP(recorder, req)

			// Security headers should always be present
			Expect(recorder.Header().Get("X-Frame-Options")).Should(Equal("DENY"))
			Expect(recorder.Header().Get("X-Content-Type-Options")).Should(Equal("nosniff"))
			Expect(recorder.Header().Get("X-XSS-Protection")).Should(Equal("1; mode=block"))
		})
	})

	When("preventing CSRF attack scenarios", func() {
		BeforeEach(func() {
			os.Setenv("ALLOWED_ORIGINS", "https://mathwizz.com")
			testHandler = corsMiddleware(simpleHandler)
		})

		It("should prevent malicious website from making authenticated requests", func() {
			// Attacker's website (evil.com) tries to make request to MathWizz API
			req := httptest.NewRequest("POST", "/api/solve", nil)
			req.Header.Set("Origin", "https://evil.com")
			req.Header.Set("Authorization", "Bearer stolen-token")

			testHandler.ServeHTTP(recorder, req)

			// Browser will block the response because Origin not allowed
			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).Should(BeEmpty())
		})

		It("should prevent subdomain takeover attacks", func() {
			// Attacker controls a subdomain
			req := httptest.NewRequest("POST", "/api/solve", nil)
			req.Header.Set("Origin", "https://evil.mathwizz.com")

			testHandler.ServeHTTP(recorder, req)

			// Subdomain not explicitly whitelisted, should be blocked
			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).Should(BeEmpty())
		})

		It("should prevent similar domain attacks", func() {
			// Attacker registers similar domain
			req := httptest.NewRequest("POST", "/api/login", nil)
			req.Header.Set("Origin", "https://mathwizz-phishing.com")

			testHandler.ServeHTTP(recorder, req)

			// Similar domain should be blocked (no partial matching)
			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).Should(BeEmpty())
		})
	})

	When("handling edge cases", func() {
		BeforeEach(func() {
			os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000")
			testHandler = corsMiddleware(simpleHandler)
		})

		It("should handle missing Origin header gracefully", func() {
			req := httptest.NewRequest("GET", "/api/test", nil)
			// No Origin header set (same-origin request)

			testHandler.ServeHTTP(recorder, req)

			// Should not panic, should process normally
			Expect(recorder.Code).Should(Equal(http.StatusOK))
			// No CORS header needed for same-origin requests
			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).Should(BeEmpty())
		})

		It("should handle empty Origin header", func() {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", "")

			testHandler.ServeHTTP(recorder, req)

			// Should not panic
			Expect(recorder.Code).Should(Equal(http.StatusOK))
		})

		It("should require exact origin match (no wildcards)", func() {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", "http://localhost:3001") // Different port

			testHandler.ServeHTTP(recorder, req)

			// Should not match (port matters)
			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).Should(BeEmpty())
		})

		It("should be case-sensitive for origin matching", func() {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", "http://LOCALHOST:3000") // Uppercase

			testHandler.ServeHTTP(recorder, req)

			// Should not match (case-sensitive)
			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).Should(BeEmpty())
		})
	})

	When("validating production configurations", func() {
		It("should support HTTPS origins for production", func() {
			os.Setenv("ALLOWED_ORIGINS", "https://mathwizz.com,https://www.mathwizz.com")
			testHandler = corsMiddleware(simpleHandler)

			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", "https://mathwizz.com")

			testHandler.ServeHTTP(recorder, req)

			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).Should(Equal("https://mathwizz.com"))
		})

		It("should support multiple production domains", func() {
			os.Setenv("ALLOWED_ORIGINS", "https://app.mathwizz.com,https://admin.mathwizz.com,https://api.mathwizz.com")
			testHandler = corsMiddleware(simpleHandler)

			// Test each domain independently
			domains := []string{
				"https://app.mathwizz.com",
				"https://admin.mathwizz.com",
				"https://api.mathwizz.com",
			}

			for _, domain := range domains {
				req := httptest.NewRequest("GET", "/api/test", nil)
				req.Header.Set("Origin", domain)
				rec := httptest.NewRecorder()

				testHandler.ServeHTTP(rec, req)

				Expect(rec.Header().Get("Access-Control-Allow-Origin")).Should(Equal(domain))
			}
		})

		It("should not allow HTTP in production whitelist to prevent downgrade attacks", func() {
			os.Setenv("ALLOWED_ORIGINS", "https://mathwizz.com")
			testHandler = corsMiddleware(simpleHandler)

			// Attacker tries HTTP (unencrypted)
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", "http://mathwizz.com") // HTTP not HTTPS

			testHandler.ServeHTTP(recorder, req)

			// Should be blocked (HTTP != HTTPS)
			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).Should(BeEmpty())
		})
	})
})

var _ = Describe("Cookie-Based Authentication", func() {
	var (
		testToken string
		userID    int
		email     string
	)

	BeforeEach(func() {
		// Initialize JWT secret for token generation/validation
		os.Setenv("JWT_SECRET", "test-secret-key-at-least-32-characters-long")
		InitJWTSecret()

		// Generate test user data
		userID = 123
		email = "test@example.com"

		// Generate valid JWT token for testing
		var err error
		testToken, err = GenerateToken(userID, email)
		Expect(err).ShouldNot(HaveOccurred())
	})

	Describe("setAuthCookies", func() {
		var recorder *httptest.ResponseRecorder

		BeforeEach(func() {
			recorder = httptest.NewRecorder()
		})

		When("setting authentication cookies", func() {
			It("should set both authToken and sessionActive cookies", func() {
				setAuthCookies(recorder, testToken)

				cookies := recorder.Result().Cookies()
				Expect(cookies).Should(HaveLen(2))

				// Find each cookie by name
				var authTokenCookie, sessionActiveCookie *http.Cookie
				for _, cookie := range cookies {
					if cookie.Name == "authToken" {
						authTokenCookie = cookie
					} else if cookie.Name == "sessionActive" {
						sessionActiveCookie = cookie
					}
				}

				Expect(authTokenCookie).ShouldNot(BeNil())
				Expect(sessionActiveCookie).ShouldNot(BeNil())
			})

			It("should set authToken cookie with correct attributes for XSS protection", func() {
				setAuthCookies(recorder, testToken)

				cookies := recorder.Result().Cookies()
				var authTokenCookie *http.Cookie
				for _, cookie := range cookies {
					if cookie.Name == "authToken" {
						authTokenCookie = cookie
						break
					}
				}

				Expect(authTokenCookie).ShouldNot(BeNil())
				Expect(authTokenCookie.Value).Should(Equal(testToken))
				Expect(authTokenCookie.HttpOnly).Should(BeTrue())                       // XSS protection
				Expect(authTokenCookie.SameSite).Should(Equal(http.SameSiteStrictMode)) // CSRF protection
				Expect(authTokenCookie.MaxAge).Should(Equal(86400))                     // 24 hours
				Expect(authTokenCookie.Path).Should(Equal("/"))
			})

			It("should set sessionActive cookie readable by JavaScript", func() {
				setAuthCookies(recorder, testToken)

				cookies := recorder.Result().Cookies()
				var sessionActiveCookie *http.Cookie
				for _, cookie := range cookies {
					if cookie.Name == "sessionActive" {
						sessionActiveCookie = cookie
						break
					}
				}

				Expect(sessionActiveCookie).ShouldNot(BeNil())
				Expect(sessionActiveCookie.Value).Should(Equal("true"))
				Expect(sessionActiveCookie.HttpOnly).Should(BeFalse()) // JavaScript can read
				Expect(sessionActiveCookie.SameSite).Should(Equal(http.SameSiteStrictMode))
				Expect(sessionActiveCookie.MaxAge).Should(Equal(86400))
				Expect(sessionActiveCookie.Path).Should(Equal("/"))
			})
		})
	})

	Describe("AuthMiddleware with Cookie-Based Authentication", func() {
		var (
			handler   http.HandlerFunc
			recorder  *httptest.ResponseRecorder
			handlerID int // ID extracted by handler
		)

		BeforeEach(func() {
			recorder = httptest.NewRecorder()
			handlerID = 0 // Reset

			// Test handler that extracts userID from context
			handler = func(w http.ResponseWriter, r *http.Request) {
				id, err := GetUserIDFromContext(r.Context())
				if err == nil {
					handlerID = id
				}
				w.WriteHeader(http.StatusOK)
			}
		})

		When("authenticating with httpOnly cookie", func() {
			It("should accept valid authToken cookie", func() {
				req := httptest.NewRequest("GET", "/api/solve", nil)
				req.AddCookie(&http.Cookie{
					Name:  "authToken",
					Value: testToken,
				})

				wrappedHandler := AuthMiddleware(handler)
				wrappedHandler.ServeHTTP(recorder, req)

				Expect(recorder.Code).Should(Equal(http.StatusOK))
				Expect(handlerID).Should(Equal(userID)) // User ID extracted correctly
			})

			It("should reject request with missing authToken cookie and no Authorization header", func() {
				req := httptest.NewRequest("GET", "/api/solve", nil)
				// No cookie, no header

				wrappedHandler := AuthMiddleware(handler)
				wrappedHandler.ServeHTTP(recorder, req)

				Expect(recorder.Code).Should(Equal(http.StatusUnauthorized))
				Expect(recorder.Body.String()).Should(ContainSubstring("missing authentication credentials"))
			})

			It("should reject request with invalid authToken cookie", func() {
				req := httptest.NewRequest("GET", "/api/solve", nil)
				req.AddCookie(&http.Cookie{
					Name:  "authToken",
					Value: "invalid-token-string",
				})

				wrappedHandler := AuthMiddleware(handler)
				wrappedHandler.ServeHTTP(recorder, req)

				Expect(recorder.Code).Should(Equal(http.StatusUnauthorized))
				Expect(recorder.Body.String()).Should(ContainSubstring("invalid or expired token"))
			})

			It("should reject request with expired authToken cookie", func() {
				// Generate token that's already expired
				claims := Claims{
					UserID: userID,
					Email:  email,
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
						IssuedAt:  jwt.NewNumericDate(time.Now().Add(-25 * time.Hour)),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				expiredToken, _ := token.SignedString(jwtSecret)

				req := httptest.NewRequest("GET", "/api/solve", nil)
				req.AddCookie(&http.Cookie{
					Name:  "authToken",
					Value: expiredToken,
				})

				wrappedHandler := AuthMiddleware(handler)
				wrappedHandler.ServeHTTP(recorder, req)

				Expect(recorder.Code).Should(Equal(http.StatusUnauthorized))
			})
		})

		When("authenticating with Authorization header (backward compatibility)", func() {
			It("should accept valid Bearer token in Authorization header", func() {
				req := httptest.NewRequest("GET", "/api/solve", nil)
				req.Header.Set("Authorization", "Bearer "+testToken)

				wrappedHandler := AuthMiddleware(handler)
				wrappedHandler.ServeHTTP(recorder, req)

				Expect(recorder.Code).Should(Equal(http.StatusOK))
				Expect(handlerID).Should(Equal(userID))
			})

			It("should reject invalid Authorization header format", func() {
				req := httptest.NewRequest("GET", "/api/solve", nil)
				req.Header.Set("Authorization", testToken) // Missing "Bearer " prefix

				wrappedHandler := AuthMiddleware(handler)
				wrappedHandler.ServeHTTP(recorder, req)

				Expect(recorder.Code).Should(Equal(http.StatusUnauthorized))
				Expect(recorder.Body.String()).Should(ContainSubstring("invalid authorization header format"))
			})
		})

		When("cookie takes priority over Authorization header", func() {
			It("should prefer authToken cookie when both cookie and header present", func() {
				// Create two different tokens
				otherUserID := 456
				otherEmail := "other@example.com"
				otherToken, _ := GenerateToken(otherUserID, otherEmail)

				req := httptest.NewRequest("GET", "/api/solve", nil)
				req.AddCookie(&http.Cookie{
					Name:  "authToken",
					Value: testToken, // userID 123
				})
				req.Header.Set("Authorization", "Bearer "+otherToken) // userID 456

				wrappedHandler := AuthMiddleware(handler)
				wrappedHandler.ServeHTTP(recorder, req)

				Expect(recorder.Code).Should(Equal(http.StatusOK))
				Expect(handlerID).Should(Equal(userID)) // Should use cookie (123), not header (456)
			})

			It("should use valid cookie even if Authorization header is malformed", func() {
				req := httptest.NewRequest("GET", "/api/solve", nil)
				req.AddCookie(&http.Cookie{
					Name:  "authToken",
					Value: testToken,
				})
				req.Header.Set("Authorization", "InvalidHeaderFormat")

				wrappedHandler := AuthMiddleware(handler)
				wrappedHandler.ServeHTTP(recorder, req)

				// Should succeed using cookie, ignoring malformed header
				Expect(recorder.Code).Should(Equal(http.StatusOK))
				Expect(handlerID).Should(Equal(userID))
			})
		})

		When("preventing XSS token theft attacks", func() {
			It("should protect token from JavaScript access via httpOnly cookie", func() {
				// This test documents that httpOnly cookies prevent XSS theft
				// In real browser, JavaScript cannot access httpOnly cookies
				// Here we verify the cookie is set with HttpOnly=true

				recorder := httptest.NewRecorder()
				setAuthCookies(recorder, testToken)

				cookies := recorder.Result().Cookies()
				var authTokenCookie *http.Cookie
				for _, cookie := range cookies {
					if cookie.Name == "authToken" {
						authTokenCookie = cookie
						break
					}
				}

				// Verify httpOnly flag prevents JavaScript access
				Expect(authTokenCookie.HttpOnly).Should(BeTrue())
				// In browser, document.cookie will NOT include this cookie
				// JavaScript cannot read the token, preventing XSS token theft
			})

			It("should allow JavaScript to check session status via sessionActive cookie", func() {
				recorder := httptest.NewRecorder()
				setAuthCookies(recorder, testToken)

				cookies := recorder.Result().Cookies()
				var sessionActiveCookie *http.Cookie
				for _, cookie := range cookies {
					if cookie.Name == "sessionActive" {
						sessionActiveCookie = cookie
						break
					}
				}

				// sessionActive is NOT httpOnly, so JavaScript can read it
				Expect(sessionActiveCookie.HttpOnly).Should(BeFalse())
				Expect(sessionActiveCookie.Value).Should(Equal("true"))
				// Frontend can check: document.cookie.includes('sessionActive=true')
				// But cannot access the actual JWT token (which is httpOnly)
			})
		})

		When("preventing CSRF attacks with SameSite cookies", func() {
			It("should set SameSite=Strict to prevent cross-site requests", func() {
				recorder := httptest.NewRecorder()
				setAuthCookies(recorder, testToken)

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					// Both cookies should have SameSite=Strict
					Expect(cookie.SameSite).Should(Equal(http.SameSiteStrictMode))
				}

				// SameSite=Strict means browser won't send cookies on cross-site requests
				// Prevents CSRF attacks from evil.com making requests to MathWizz API
			})
		})
	})
})
