package main

// This file is the entry point for the history-worker service.
// It handles initialization, connects to dependencies (PostgreSQL, NATS), starts the worker,
// and manages graceful shutdown on SIGINT/SIGTERM signals.
//
// The history-worker is a long-running background service that:
//   1. Subscribes to NATS "problem_solved" topic
//   2. Receives ProblemSolvedEvent messages published by web-server
//   3. Validates and persists events to PostgreSQL history table
//   4. Runs continuously until terminated
//
// Lifecycle:
//   - Startup: Connect to database → Connect to NATS → Start worker subscription → Wait for signals
//   - Runtime: Process events asynchronously as they arrive from NATS
//   - Shutdown: Receive SIGINT/SIGTERM → Log shutdown message → Defer cleanup (db.Close, nc.Close)
//
// Graceful Shutdown:
//   This implementation includes signal handling for graceful shutdown (SIGINT, SIGTERM),
//   which is better than web-server/main.go that lacks this feature.

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
)

// main is the entry point for the history-worker service.
// It loads configuration from environment variables and delegates to Run().
//
// Behavior:
//   - Loads configuration from environment variables with defaults
//   - Calls Run() with configuration parameters
//   - Handles errors from Run() by logging and exiting with code 1
//   - Normal termination (signal received) exits with code 0
//
// Environment Variables:
//   - DB_HOST: PostgreSQL host (default: "localhost")
//   - DB_USER: PostgreSQL username (default: "mathwizz")
//   - DB_PASSWORD: PostgreSQL password (default: "mathwizz_password")
//   - DB_NAME: PostgreSQL database name (default: "mathwizz")
//   - DB_SSL_MODE: PostgreSQL SSL mode (default: "require", options: disable, require, verify-ca, verify-full)
//   - NATS_URL: NATS server URL (default: "nats://localhost:4222")
//
// Error Handling:
//   - Run() errors logged with log.Fatalf (exits with code 1)
//   - Normal shutdown (signal) exits with code 0
//
// Testing:
//   - main() is not directly testable due to log.Fatalf
//   - Test Run() function instead (see main_test.go)
//
// Related Files:
//   - main_test.go: Tests for Run() function
//   - worker.go: StartWorker function called from Run()
//   - database.go: ConnectDB function called from Run()
func main() {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := 5432
	dbUser := getEnv("DB_USER", "mathwizz")
	dbPassword := getEnv("DB_PASSWORD", "mathwizz_password")
	dbName := getEnv("DB_NAME", "mathwizz")
	dbSSLMode := getEnv("DB_SSL_MODE", "require")

	natsURL := getEnv("NATS_URL", "nats://localhost:4222")

	if err := Run(dbHost, dbUser, dbPassword, dbName, dbSSLMode, natsURL, dbPort); err != nil {
		log.Fatalf("Fatal error: %v", err)
	}
}

// Run executes the main worker logic and returns errors instead of calling log.Fatalf.
// This function is testable - it can be called from tests with mock dependencies.
//
// Parameters:
//   - dbHost: PostgreSQL server hostname or IP (e.g., "localhost", "postgres-service")
//   - dbUser: Database username (e.g., "mathwizz")
//   - dbPassword: Database password (encrypted in transit when SSL is enabled)
//   - dbName: Database name (e.g., "mathwizz")
//   - dbSSLMode: PostgreSQL SSL mode (disable, require, verify-ca, verify-full)
//   - natsURL: NATS server URL (e.g., "nats://localhost:4222")
//   - dbPort: PostgreSQL port (typically 5432)
//
// Returns:
//   - error: Non-nil if database connection, NATS connection, or worker start fails
//   - nil on graceful shutdown (signal received)
//
// Startup Sequence:
//  1. Validate SSL mode and log warnings for insecure configurations
//  2. Connect to PostgreSQL database
//  3. Connect to NATS message queue with reconnection options
//  4. Start worker subscription to "problem_solved" topic
//  5. Set up signal handlers for graceful shutdown
//  6. Block until SIGINT (Ctrl+C) or SIGTERM (Kubernetes termination)
//  7. Log shutdown and return nil (defer cleanup runs automatically)
//
// Database Connection:
//   - Uses ConnectDB from database.go
//   - Connection pool shared across all event processing goroutines
//   - Cleanup via defer db.Close() (executes even on error returns)
//
// NATS Connection:
//   - Connects to NATS server with automatic reconnection
//   - ReconnectWait: 2 seconds between reconnection attempts
//   - MaxReconnects: -1 (unlimited reconnection attempts)
//   - DisconnectErrHandler: Logs disconnections for debugging
//   - ReconnectHandler: Logs successful reconnections
//   - Resilient to temporary NATS outages
//   - Cleanup via defer nc.Close() (executes even on error returns)
//
// Worker Subscription:
//   - Calls StartWorker to subscribe to "problem_solved" topic
//   - Subscription runs in background goroutines (asynchronous)
//   - Events processed concurrently as they arrive
//   - If StartWorker fails, error is returned (NOT log.Fatalf)
//
// Graceful Shutdown:
//   - Implements signal handling for SIGINT (Ctrl+C) and SIGTERM (Kubernetes)
//   - Blocks on channel receive (<-sigChan) until signal arrives
//   - Logs "Shutting down gracefully..." message
//   - Deferred cleanup runs automatically (db.Close, nc.Close)
//   - Returns nil to indicate clean shutdown
//
// Error Handling:
//   - Database connection failure: Returns error (NOT log.Fatalf)
//   - NATS connection failure: Returns error with cleanup (defer db.Close() executes)
//   - Worker start failure: Returns error with cleanup (defer db.Close() and nc.Close() execute)
//   - Event processing errors: Logged by worker, Run() continues (doesn't return error)
//   - Signal received: Returns nil (clean shutdown)
//
// Resource Cleanup:
//   - Uses defer statements for cleanup (execute even on error returns)
//   - Fixes Bug #32: Database connection closed even if NATS connection fails
//   - Cleanup order: NATS closed first, then database (reverse of creation order)
//
// Kubernetes Deployment:
//   - SIGTERM sent by Kubernetes during pod termination
//   - Default grace period: 30 seconds
//   - Worker should drain in-flight events before exiting
//   - Current implementation: Events being processed may be lost (Bug #33)
//   - Recommendation: Add timeout and event drain logic
//
// Testing Considerations:
//   - Test graceful shutdown with SIGTERM signal
//   - Test database connection failure returns error without leaking resources
//   - Test NATS connection failure returns error and closes database
//   - Test worker start failure returns error and closes database and NATS
//   - Test signal handling interrupts blocking behavior
//
// Example Production Deployment:
//
//	```yaml
//	# Kubernetes Deployment
//	apiVersion: apps/v1
//	kind: Deployment
//	metadata:
//	  name: history-worker
//	spec:
//	  replicas: 3  # Horizontal scaling with queue groups
//	  template:
//	    spec:
//	      containers:
//	      - name: history-worker
//	        image: mathwizz/history-worker:latest
//	        env:
//	        - name: DB_HOST
//	          value: "postgres-service"
//	        - name: DB_USER
//	          valueFrom:
//	            secretKeyRef:
//	              name: mathwizz-secrets
//	              key: db-user
//	        - name: DB_PASSWORD
//	          valueFrom:
//	            secretKeyRef:
//	              name: mathwizz-secrets
//	              key: db-password
//	        - name: NATS_URL
//	          value: "nats://nats-service:4222"
//	        livenessProbe:
//	          httpGet:
//	            path: /health
//	            port: 8081
//	          initialDelaySeconds: 10
//	          periodSeconds: 30
//	        readinessProbe:
//	          httpGet:
//	            path: /health
//	            port: 8081
//	          initialDelaySeconds: 5
//	          periodSeconds: 10
//	```
//
// Related Functions:
//   - StartWorker: Subscribes to NATS and starts event processing (worker.go)
//   - ConnectDB: Establishes PostgreSQL connection (database.go)
//   - getEnv: Loads environment variables with defaults (main.go)
//
// Related Files:
//   - main_test.go: Integration tests for Run() function
//   - worker.go: StartWorker function implementation
//   - database.go: ConnectDB function implementation
func Run(dbHost, dbUser, dbPassword, dbName, dbSSLMode, natsURL string, dbPort int) error {
	// Validate SSL mode and warn about insecure configurations
	log.Printf("Database SSL mode: %s", dbSSLMode)
	if dbSSLMode == "disable" {
		log.Println("WARNING: Database SSL is DISABLED - all data transmitted in plaintext!")
		log.Println("WARNING: This is INSECURE and should only be used for local development")
		log.Println("WARNING: Set DB_SSL_MODE=require for production deployments")
	}

	log.Println("Connecting to database...")
	db, err := ConnectDB(dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()
	log.Println("Database connected successfully")

	log.Println("Connecting to NATS...")
	nc, err := nats.Connect(natsURL,
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				log.Printf("NATS disconnected: %v", err)
			}
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			log.Println("NATS reconnected")
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	defer nc.Close()
	log.Println("NATS connected successfully")

	if err := StartWorker(db, nc); err != nil {
		return fmt.Errorf("failed to start worker: %w", err)
	}

	log.Println("History worker is running. Press Ctrl+C to exit.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
	return nil
}

// getEnv retrieves an environment variable value or returns a default if not set.
// This is a common pattern for configuration management with environment variables.
//
// Parameters:
//   - key: The environment variable name to look up (e.g., "DB_HOST", "NATS_URL")
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
// Example Usage:
//
//	host := getEnv("DB_HOST", "localhost")
//	dbName := getEnv("DB_NAME", "mathwizz")
//
// Note: This function is identical to web-server/main.go:getEnv (same implementation).
// Consider extracting to a shared configuration package if the codebase grows.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
