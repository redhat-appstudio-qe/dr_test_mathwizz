package main

// This file is the entry point for the web-server service.
// It initializes database and NATS connections, sets up routes, and starts the HTTP server.

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

// main is the entry point for the web-server service.
// Initializes database and NATS connections, configures routes with middleware,
// and starts the HTTP server with appropriate timeouts.
//
// Startup Sequence:
//  1. Load configuration from environment variables (with defaults)
//  2. Connect to PostgreSQL database
//  3. Connect to NATS message queue
//  4. Initialize Server struct with dependencies
//  5. Configure routes with gorilla/mux router
//  6. Apply CORS middleware globally
//  7. Start HTTP server with timeouts
//
// Environment Variables:
//   - JWT_SECRET: Secret key for JWT signing (REQUIRED, min 32 chars)
//   - DB_HOST: PostgreSQL host (default: "localhost")
//   - DB_USER: PostgreSQL username (default: "mathwizz")
//   - DB_PASSWORD: PostgreSQL password (default: "mathwizz_password")
//   - DB_NAME: PostgreSQL database name (default: "mathwizz")
//   - DB_SSL_MODE: PostgreSQL SSL mode (default: "require", options: disable, require, verify-ca, verify-full)
//   - NATS_URL: NATS server URL (default: "nats://localhost:4222")
//   - PORT: HTTP server port (default: "8080")
//   - ALLOWED_ORIGINS: Comma-separated CORS origins (default: "http://localhost:3000")
//
// Routes Configured:
//   - GET    /health   - Health check endpoint (no auth)
//   - POST   /register - User registration (no auth)
//   - POST   /login    - User login (no auth)
//   - GET    /history  - Get user history (requires JWT via AuthMiddleware)
//   - POST   /solve    - Solve math problem (requires JWT via AuthMiddleware)
//   - OPTIONS (all)    - CORS preflight (handled by corsMiddleware)
//
// HTTP Server Configuration:
//   - ReadTimeout: 15 seconds (prevents slow client DoS)
//   - WriteTimeout: 15 seconds (prevents slow response DoS)
//   - IdleTimeout: 60 seconds (keeps connections alive briefly)
//
// Graceful Shutdown:
//   - Listens for SIGINT (Ctrl+C) and SIGTERM (Kubernetes pod termination)
//   - Stops accepting new connections when shutdown signal received
//   - Allows in-flight requests to complete (up to 30 second timeout)
//   - Closes database connections cleanly via deferred db.Close()
//   - Closes NATS connections cleanly via deferred nc.Close()
//   - Prevents data loss in async NATS publish goroutines
//   - Ensures proper cleanup in Kubernetes environments
//
// Security Considerations:
//   - **Good**: Uses proper HTTP timeouts to prevent resource exhaustion
//   - **Good**: Uses dependency injection (Server struct)
//   - **Good**: Defers cleanup (db.Close(), nc.Close())
//   - **Good**: Implements graceful shutdown (SIGTERM/SIGINT handling)
//   - **ISSUE**: No TLS/HTTPS configuration (should use TLS in production)
//   - **ISSUE**: No startup health checks (database schema validation)
//   - **ISSUE**: Logs contain configuration details (could leak sensitive info)
//
// Production Improvements Needed:
//  1. Require critical environment variables (fail if not set)
//  2. Validate database schema on startup
//  3. Add TLS support with Let's Encrypt or cert files
//  4. Implement structured logging (JSON format)
//  5. Add metrics and tracing (Prometheus, OpenTelemetry)
//  6. Implement health check that validates DB/NATS connectivity
//
// Example Deployment:
//
//	# Development
//	export DB_HOST=localhost
//	export DB_PASSWORD=secure_password_123
//	go run .
//
//	# Production (with all variables set)
//	export DB_HOST=database-service
//	export DB_PASSWORD=$(cat /run/secrets/db-password)
//	export JWT_SECRET=$(cat /run/secrets/jwt-secret)
//	export PORT=8080
//	./web-server
//
// Run initializes and starts the web server with the given configuration.
// Returns error instead of calling log.Fatal to enable testing of startup failures.
//
// This function extracts the initialization logic from main() to make it testable.
// It performs the same startup sequence but returns errors instead of exiting the process.
//
// Parameters:
//   - ctx: Context for cancellation and shutdown signaling
//
// Returns:
//   - error: Any error that occurs during initialization or server operation
//
// Testability:
//   - Can be tested with invalid database configurations
//   - Can be tested with invalid NATS URLs
//   - Can be tested with missing JWT secrets
//   - Can be tested for proper resource cleanup
//   - Enables integration tests for startup logic
//
// Example Usage:
//
//	// In main()
//	ctx := context.Background()
//	if err := Run(ctx); err != nil {
//	    log.Fatalf("Server failed: %v", err)
//	}
//
//	// In tests
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//	err := Run(ctx)
//	// Assert on err
func Run(ctx context.Context) error {
	// Initialize JWT secret from environment variable (REQUIRED)
	log.Println("Initializing JWT secret...")
	if err := InitJWTSecret(); err != nil {
		return fmt.Errorf("failed to initialize JWT secret: %w", err)
	}
	log.Println("JWT secret initialized successfully")

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := 5432
	dbUser := getEnv("DB_USER", "mathwizz")
	dbPassword := getEnv("DB_PASSWORD", "mathwizz_password")
	dbName := getEnv("DB_NAME", "mathwizz")
	dbSSLMode := getEnv("DB_SSL_MODE", "require")

	natsURL := getEnv("NATS_URL", "nats://localhost:4222")

	port := getEnv("PORT", "8080")

	// Validate SSL mode and warn about insecure configurations
	log.Printf("Database SSL mode: %s", dbSSLMode)
	if dbSSLMode == "disable" {
		log.Println("WARNING: Database SSL is DISABLED - all data transmitted in plaintext!")
		log.Println("WARNING: This is INSECURE and should only be used for local development")
		log.Println("WARNING: Set DB_SSL_MODE=require for production deployments")
	}

	// Log CORS configuration
	allowedOrigins := getEnv("ALLOWED_ORIGINS", "http://localhost:3000")
	log.Printf("CORS allowed origins: %s", allowedOrigins)
	if allowedOrigins == "http://localhost:3000" {
		log.Println("INFO: Using default CORS origin for development (http://localhost:3000)")
		log.Println("INFO: Set ALLOWED_ORIGINS environment variable for production deployments")
	}

	log.Println("Connecting to database...")
	db, err := ConnectDB(dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()
	log.Println("Database connected successfully")

	log.Println("Connecting to NATS...")
	nc, err := ConnectNATS(natsURL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	defer nc.Close()
	log.Println("NATS connected successfully")

	server := &Server{
		DB:   db,
		NATS: nc,
	}

	// Create rate limiter for authentication endpoints (5 requests per minute, burst of 5)
	// This prevents brute force attacks, user enumeration, and account creation spam
	authRateLimiter := NewRateLimiter(5.0/60.0, 5)

	router := mux.NewRouter()

	router.HandleFunc("/health", server.HealthHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/register", RateLimitMiddleware(authRateLimiter)(server.RegisterHandler)).Methods("POST", "OPTIONS")
	router.HandleFunc("/login", RateLimitMiddleware(authRateLimiter)(server.LoginHandler)).Methods("POST", "OPTIONS")
	router.HandleFunc("/history", AuthMiddleware(server.HistoryHandler)).Methods("GET", "OPTIONS")
	router.HandleFunc("/solve", AuthMiddleware(server.SolveHandler)).Methods("POST", "OPTIONS")

	router.Use(corsMiddleware)

	addr := fmt.Sprintf(":%s", port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Create a channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- fmt.Errorf("server failed: %w", err)
		}
	}()

	log.Printf("Server is ready to handle requests at %s", addr)

	// Wait for interrupt signal or context cancellation or server error
	select {
	case <-quit:
		log.Println("Shutdown signal received, shutting down gracefully...")
	case <-ctx.Done():
		log.Println("Context cancelled, shutting down gracefully...")
	case err := <-serverErr:
		return err
	}

	// Create context with timeout for shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server exited")
	return nil
}

func main() {
	ctx := context.Background()
	if err := Run(ctx); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// corsMiddleware adds Cross-Origin Resource Sharing (CORS) headers to all responses.
// Restricts cross-origin requests to only allowed frontend origins for security.
//
// Configuration:
//   - Allowed origins loaded from ALLOWED_ORIGINS environment variable (comma-separated)
//   - Default: "http://localhost:3000" (development)
//   - Production example: "https://mathwizz.com,https://www.mathwizz.com"
//   - Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
//   - Access-Control-Allow-Headers: Content-Type, Authorization
//
// Security Features:
//   - **Origin Validation**: Only allows requests from explicitly whitelisted origins
//   - **No Wildcard**: Never uses "*" (prevents CSRF attacks from malicious websites)
//   - **Security Headers**: Adds X-Frame-Options, X-Content-Type-Options, X-XSS-Protection
//   - **Defense in Depth**: Multiple headers provide layered protection
//
// Origin Matching:
//   - Extracts Origin header from incoming request
//   - Compares against allowed origins list (exact string match)
//   - Only sets Access-Control-Allow-Origin if origin is allowed
//   - No header set if origin not in whitelist (browser blocks request)
//
// Preflight Handling:
//   - OPTIONS requests return 200 OK immediately for CORS preflight
//   - Browser verifies CORS permissions before actual request
//   - All routes include OPTIONS method for preflight support
//
// Additional Security Headers:
//   - X-Frame-Options: DENY - Prevents clickjacking attacks (page cannot be embedded in iframe)
//   - X-Content-Type-Options: nosniff - Prevents MIME type sniffing
//   - X-XSS-Protection: 1; mode=block - Enables browser XSS filter
//
// Attack Scenarios Prevented:
//  1. **CSRF via Malicious Website**:
//     - Attacker hosts evil.com
//     - Victim visits evil.com while logged into MathWizz
//     - Evil.com JavaScript tries to make requests to MathWizz API
//     - Browser blocks request (origin not in whitelist)
//  2. **Clickjacking**:
//     - Attacker embeds MathWizz in hidden iframe
//     - X-Frame-Options: DENY blocks iframe embedding
//  3. **MIME Sniffing**:
//     - Attacker tricks browser into executing JSON as JavaScript
//     - X-Content-Type-Options: nosniff prevents type confusion
//
// Environment Variable Format:
//
//	# Development (single origin)
//	export ALLOWED_ORIGINS="http://localhost:3000"
//
//	# Production (multiple origins, comma-separated, no spaces)
//	export ALLOWED_ORIGINS="https://mathwizz.com,https://www.mathwizz.com,https://app.mathwizz.com"
//
// Middleware Application:
//   - Applied globally to all routes via router.Use(corsMiddleware)
//   - Runs before route handlers
//   - Runs before AuthMiddleware (CORS happens first)
//
// Security Best Practices:
//   - Always use HTTPS origins in production (https://)
//   - Never use wildcard ("*") for allowed origins
//   - Keep the allowed origins list as small as possible (principle of least privilege)
//   - Include all required subdomains (www, app, api) explicitly
//   - Log rejected origins for security monitoring
func corsMiddleware(next http.Handler) http.Handler {
	// Load allowed origins from environment variable
	// Default to localhost:3000 for development if not set
	allowedOriginsEnv := getEnv("ALLOWED_ORIGINS", "http://localhost:3000")
	allowedOrigins := strings.Split(allowedOriginsEnv, ",")

	// Trim whitespace from each origin (in case user added spaces)
	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if the origin is in the allowed list
		originAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				// Only set Access-Control-Allow-Origin if origin is whitelisted
				w.Header().Set("Access-Control-Allow-Origin", origin)
				// Allow credentials (cookies) for authenticated requests
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				originAllowed = true
				break
			}
		}

		// Set CORS headers for allowed methods and headers
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Add security headers (applied to all requests regardless of origin)
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Log rejected origins for security monitoring (but allow request to proceed)
		// Note: Browser will block the request if origin not allowed
		if origin != "" && !originAllowed {
			log.Printf("CORS: Rejected origin: %s (allowed origins: %v)", origin, allowedOrigins)
		}

		next.ServeHTTP(w, r)
	})
}

// getEnv retrieves an environment variable value or returns a default if not set.
// This is a common pattern for configuration management with environment variables.
//
// Parameters:
//   - key: The environment variable name to look up (e.g., "DB_HOST", "PORT")
//   - defaultValue: The value to return if the environment variable is not set or empty
//
// Returns:
//   - string: The environment variable value if set and non-empty, otherwise defaultValue
//
// Behavior:
//   - Uses os.Getenv(key) to retrieve the value
//   - Checks if value is empty string ("")
//   - Returns defaultValue if empty, otherwise returns the actual value
//
// Security Considerations:
//   - **CRITICAL**: Should NOT be used for sensitive values like passwords or secrets
//   - Default values are visible in public code (open-source exposure)
//   - Empty string ("") is treated as "not set" (might not be desired behavior)
//
// Appropriate Uses (OK to have defaults):
//   - DB_HOST: "localhost" (reasonable default)
//   - PORT: "8080" (standard HTTP alt port)
//   - DB_NAME: "mathwizz" (application-specific)
//   - LOG_LEVEL: "info" (sensible default)
//
// Inappropriate Uses (should require explicit setting):
//   - DB_PASSWORD: Should have NO default (fail if not set)
//   - JWT_SECRET: Should have NO default (fail if not set)
//   - API_KEYS: Should have NO default (fail if not set)
//   - PRIVATE_KEYS: Should have NO default (fail if not set)
//
// Better Pattern for Secrets:
//
//	```go
//	dbPassword := os.Getenv("DB_PASSWORD")
//	if dbPassword == "" {
//	    log.Fatal("DB_PASSWORD environment variable is required")
//	}
//	if len(dbPassword) < 16 {
//	    log.Fatal("DB_PASSWORD must be at least 16 characters")
//	}
//	```
//
// Example Usage:
//
//	// Good: non-sensitive with reasonable default
//	port := getEnv("PORT", "8080")
//
//	// Bad: sensitive value with weak default
//	password := getEnv("DB_PASSWORD", "weak_default")  // Don't do this!
//
//	// Better: no default for sensitive values
//	password := os.Getenv("DB_PASSWORD")
//	if password == "" {
//	    log.Fatal("DB_PASSWORD required")
//	}
//
// Alternative Patterns:
//   - Use a configuration struct with validation
//   - Use a configuration library (viper, envconfig)
//   - Use a secrets manager (Vault, AWS Secrets Manager)
//   - Distinguish between required and optional config
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
