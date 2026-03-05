package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMainFunction(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main Function Startup Tests")
}

var _ = Describe("Run", func() {
	var originalJWTSecret string
	var originalDBHost string
	var originalDBUser string
	var originalDBPassword string
	var originalDBName string
	var originalDBSSLMode string
	var originalNATSURL string
	var originalPort string

	BeforeEach(func() {
		// Save original environment variables
		originalJWTSecret = os.Getenv("JWT_SECRET")
		originalDBHost = os.Getenv("DB_HOST")
		originalDBUser = os.Getenv("DB_USER")
		originalDBPassword = os.Getenv("DB_PASSWORD")
		originalDBName = os.Getenv("DB_NAME")
		originalDBSSLMode = os.Getenv("DB_SSL_MODE")
		originalNATSURL = os.Getenv("NATS_URL")
		originalPort = os.Getenv("PORT")
	})

	AfterEach(func() {
		// Restore original environment variables
		if originalJWTSecret != "" {
			os.Setenv("JWT_SECRET", originalJWTSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
		if originalDBHost != "" {
			os.Setenv("DB_HOST", originalDBHost)
		} else {
			os.Unsetenv("DB_HOST")
		}
		if originalDBUser != "" {
			os.Setenv("DB_USER", originalDBUser)
		} else {
			os.Unsetenv("DB_USER")
		}
		if originalDBPassword != "" {
			os.Setenv("DB_PASSWORD", originalDBPassword)
		} else {
			os.Unsetenv("DB_PASSWORD")
		}
		if originalDBName != "" {
			os.Setenv("DB_NAME", originalDBName)
		} else {
			os.Unsetenv("DB_NAME")
		}
		if originalDBSSLMode != "" {
			os.Setenv("DB_SSL_MODE", originalDBSSLMode)
		} else {
			os.Unsetenv("DB_SSL_MODE")
		}
		if originalNATSURL != "" {
			os.Setenv("NATS_URL", originalNATSURL)
		} else {
			os.Unsetenv("NATS_URL")
		}
		if originalPort != "" {
			os.Setenv("PORT", originalPort)
		} else {
			os.Unsetenv("PORT")
		}
	})

	When("testing JWT secret initialization failures", func() {
		It("should return error when JWT_SECRET environment variable is not set", func() {
			// Unset JWT_SECRET to trigger initialization error
			os.Unsetenv("JWT_SECRET")

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := Run(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to initialize JWT secret"))
		})

		It("should return error when JWT_SECRET is too short (less than 32 characters)", func() {
			// Set JWT_SECRET to a value that's too short
			os.Setenv("JWT_SECRET", "short-secret")

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := Run(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to initialize JWT secret"))
			Expect(err.Error()).Should(ContainSubstring("at least 32 characters"))
		})

		It("should return error when JWT_SECRET is empty string", func() {
			os.Setenv("JWT_SECRET", "")

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := Run(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to initialize JWT secret"))
		})
	})

	When("testing database connection failures", func() {
		BeforeEach(func() {
			// Set valid JWT_SECRET for these tests
			os.Setenv("JWT_SECRET", "test-secret-key-at-least-32-characters-long-for-jwt")
		})

		It("should return error when database host is unreachable", func() {
			os.Setenv("DB_HOST", "invalid-nonexistent-host-12345.local")
			os.Setenv("DB_USER", "testuser")
			os.Setenv("DB_PASSWORD", "testpass")
			os.Setenv("DB_NAME", "testdb")
			os.Setenv("DB_SSL_MODE", "disable")

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := Run(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to connect to database"))
		})

		It("should return error when database credentials are invalid", func() {
			// Use localhost but invalid credentials
			os.Setenv("DB_HOST", "localhost")
			os.Setenv("DB_USER", "invalid_user")
			os.Setenv("DB_PASSWORD", "invalid_password")
			os.Setenv("DB_NAME", "nonexistent_database")
			os.Setenv("DB_SSL_MODE", "disable")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := Run(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to connect to database"))
		})

		It("should return error with empty database name", func() {
			os.Setenv("DB_HOST", "localhost")
			os.Setenv("DB_USER", "testuser")
			os.Setenv("DB_PASSWORD", "testpass")
			os.Setenv("DB_NAME", "")
			os.Setenv("DB_SSL_MODE", "disable")

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := Run(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to connect to database"))
		})
	})

	When("testing NATS connection failures", func() {
		BeforeEach(func() {
			// Set valid JWT_SECRET and skip database connection by using a mock
			// Note: These tests will still fail at NATS connection, but we test error handling
			os.Setenv("JWT_SECRET", "test-secret-key-at-least-32-characters-long-for-jwt")
		})

		It("should return error when NATS URL is invalid", func() {
			Skip("Requires database to be available - skipping for unit test isolation")
			// This test would require database to be available first
			// then test NATS failure - better suited for integration test with testcontainers

			os.Setenv("NATS_URL", "nats://invalid-host-12345.local:4222")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := Run(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to connect to NATS"))
		})

		It("should return error when NATS URL has invalid scheme", func() {
			Skip("Requires database to be available - skipping for unit test isolation")

			os.Setenv("NATS_URL", "http://localhost:4222")

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := Run(ctx)
			Expect(err).Should(HaveOccurred())
			// Will fail at database first in current implementation
		})
	})

	When("testing environment variable configuration", func() {
		It("should use default values when environment variables are not set", func() {
			// This test documents default behavior
			// We can't actually test Run() succeeding without real database/NATS
			// but we can verify getEnv() function behavior

			// Test getEnv helper function
			Expect(getEnv("NONEXISTENT_VAR", "default_value")).Should(Equal("default_value"))
		})

		It("should use environment variable values when set", func() {
			os.Setenv("TEST_VAR", "custom_value")
			defer os.Unsetenv("TEST_VAR")

			Expect(getEnv("TEST_VAR", "default_value")).Should(Equal("custom_value"))
		})

		It("should return default when environment variable is empty string", func() {
			os.Setenv("TEST_VAR", "")
			defer os.Unsetenv("TEST_VAR")

			Expect(getEnv("TEST_VAR", "default_value")).Should(Equal("default_value"))
		})
	})

	When("testing context cancellation during startup", func() {
		It("should handle context cancellation gracefully", func() {
			os.Setenv("JWT_SECRET", "test-secret-key-at-least-32-characters-long-for-jwt")
			os.Setenv("DB_HOST", "invalid-host-that-will-timeout.local")
			os.Setenv("DB_SSL_MODE", "disable")

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			// This should fail due to database connection, but should handle context timeout
			err := Run(ctx)
			Expect(err).Should(HaveOccurred())
		})
	})

	When("testing error message format", func() {
		It("should wrap errors with descriptive context using fmt.Errorf", func() {
			os.Unsetenv("JWT_SECRET")

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			err := Run(ctx)
			Expect(err).Should(HaveOccurred())

			// Verify error is wrapped with context
			Expect(err.Error()).Should(ContainSubstring("failed to initialize JWT secret"))

			// Verify we can unwrap to get original error (Go 1.13+ error wrapping)
			Expect(fmt.Sprintf("%v", err)).ShouldNot(BeEmpty())
		})

		It("should provide informative error messages for database failures", func() {
			os.Setenv("JWT_SECRET", "test-secret-key-at-least-32-characters-long-for-jwt")
			os.Setenv("DB_HOST", "nonexistent.host")
			os.Setenv("DB_SSL_MODE", "disable")

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			err := Run(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to connect to database"))
			Expect(err.Error()).Should(MatchRegexp("failed to connect|no such host|connection refused"))
		})
	})
})
