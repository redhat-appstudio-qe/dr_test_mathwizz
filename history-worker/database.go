package main

// This file handles all database operations for the history-worker service.
// It provides functions for connecting to PostgreSQL and persisting problem-solving history.
//
// The history-worker writes to the same PostgreSQL database as the web-server, but only
// performs INSERT operations on the history table. The web-server handles reads (GET /history).

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// DBConnection is an interface abstraction over database operations for testability.
// Allows dependency injection of mock database connections in unit tests.
//
// Methods:
//   - Exec: Execute a parameterized SQL statement (INSERT, UPDATE, DELETE)
//
// Usage Pattern:
//
//	This interface enables testing without a real PostgreSQL instance:
//	- Production: Pass *sql.DB (implements DBConnection via Exec method)
//	- Testing: Pass mock struct that implements DBConnection
//
// Example Mock for Testing:
//
//	type MockDB struct {
//	    ExecFunc func(query string, args ...interface{}) (sql.Result, error)
//	}
//
//	func (m *MockDB) Exec(query string, args ...interface{}) (sql.Result, error) {
//	    return m.ExecFunc(query, args...)
//	}
//
//	// In test
//	mockDB := &MockDB{
//	    ExecFunc: func(query string, args ...interface{}) (sql.Result, error) {
//	        // Verify query, return mock result
//	        return mockResult, nil
//	    },
//	}
//	err := CreateHistoryItem(mockDB, 123, "2+2", "4")
//
// Design Rationale:
//   - Minimal interface (only Exec needed for history-worker)
//   - sql.DB implements this interface naturally (has Exec method)
//   - Enables testing without database infrastructure
//   - Standard Go pattern for database abstraction
type DBConnection interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// ConnectDB establishes a connection to the PostgreSQL database.
// Opens a connection pool, verifies connectivity, and returns a database handle.
//
// Parameters:
//   - host: PostgreSQL server hostname or IP (e.g., "localhost", "postgres-service")
//   - user: Database username (e.g., "mathwizz")
//   - password: Database password (encrypted in transit when SSL is enabled)
//   - dbname: Database name (e.g., "mathwizz")
//   - port: PostgreSQL port (typically 5432)
//   - sslMode: PostgreSQL SSL mode - controls encryption of database traffic
//
// SSL Mode Values:
//   - "disable": No SSL (INSECURE - all data transmitted in plaintext)
//   - "require": Requires SSL, but doesn't verify server certificate (protects against passive eavesdropping)
//   - "verify-ca": Requires SSL and verifies server certificate against CA (recommended for production)
//   - "verify-full": Requires SSL, verifies certificate and hostname match (most secure)
//
// Returns:
//   - *sql.DB: Database connection pool (safe for concurrent use)
//   - error: Non-nil if connection fails or ping times out
//
// Connection Pool Behavior:
//   - sql.Open creates a connection pool, but doesn't establish connection immediately
//   - db.Ping() forces a connection attempt to verify database is reachable
//   - Connection pool is safe for concurrent use (multiple goroutines can share it)
//   - Default pool settings: unlimited max open connections, 2 max idle connections
//
// Security Considerations:
//   - SSL mode is configurable via parameter for flexibility across environments
//   - Production deployments should use "require", "verify-ca", or "verify-full"
//   - Development environments may use "disable" for local testing (if explicitly configured)
//   - Password is passed in connection string (use secret management in production)
//
// Production Recommendations:
//   - Use sslMode="require" (minimum) or "verify-full" (best)
//   - Configure PostgreSQL to require SSL with certificates
//   - Add connection timeout: `connect_timeout=10`
//   - Configure connection pool limits:
//   - db.SetMaxOpenConns(25)   // Limit concurrent connections
//   - db.SetMaxIdleConns(5)    // Limit idle connections
//   - db.SetConnMaxLifetime(5 * time.Minute)  // Recycle connections
//   - Load password from secret management system, not plaintext
//   - For verify-ca/verify-full: mount CA certificate and configure sslrootcert parameter
//
// Error Cases:
//   - Invalid host/port: Returns "failed to open database: <details>"
//   - Database not running: Returns "failed to ping database: connection refused"
//   - Wrong credentials: Returns "failed to ping database: authentication failed"
//   - Network timeout: Returns "failed to ping database: timeout"
//   - SSL handshake failure (if sslmode=require but server doesn't support SSL): Returns "failed to ping database: SSL not supported"
//   - Certificate verification failure (if verify-ca/verify-full and cert invalid): Returns "failed to ping database: certificate verify failed"
//
// Example Usage:
//
//	// Development - local unencrypted connection
//	db, err := ConnectDB("localhost", "mathwizz", "password", "mathwizz", 5432, "disable")
//	if err != nil {
//	    log.Fatalf("Database connection failed: %v", err)
//	}
//	defer db.Close()
//
//	// Production - require SSL
//	db, err := ConnectDB(
//	    os.Getenv("DB_HOST"),
//	    os.Getenv("DB_USER"),
//	    os.Getenv("DB_PASSWORD"),
//	    os.Getenv("DB_NAME"),
//	    5432,
//	    "require",
//	)
//
// Kubernetes Deployment:
//   - DB_HOST should point to PostgreSQL service name (e.g., "postgres-service")
//   - DB_PASSWORD should be loaded from Kubernetes secret
//   - DB_SSL_MODE should be set to "require" or higher
//   - Consider using SSL certificates mounted as secrets
//
// Related Files:
//   - web-server/database.go: Uses identical connection pattern
//   - main.go: Calls this function during startup
func ConnectDB(host, user, password, dbname string, port int, sslMode string) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// CreateHistoryItem inserts a new problem-solving history record into the database.
// Called by ProcessEvent when consuming ProblemSolvedEvent messages from NATS.
//
// Parameters:
//   - db: Database connection (implements DBConnection interface for testability)
//   - userID: The authenticated user's ID (foreign key references users.id)
//   - problem: The mathematical expression that was solved (e.g., "25+75")
//   - answer: The computed answer as a string (e.g., "100")
//
// Returns:
//   - error: Non-nil if INSERT fails, nil on success
//
// Database Operation:
//   - Table: history
//   - Query: INSERT INTO history (user_id, problem_text, answer_text, created_at) VALUES ($1, $2, $3, $4)
//   - Uses parameterized query ($1, $2, $3, $4) to prevent SQL injection
//   - Sets created_at to current timestamp (time.Now())
//   - Auto-generates ID using PostgreSQL SERIAL type
//
// Event Processing Flow:
//  1. Web-server publishes ProblemSolvedEvent to NATS
//  2. History-worker receives event via subscription
//  3. Worker calls ParseEvent to validate event structure
//  4. Worker calls CreateHistoryItem to persist to database
//  5. On success, event processing complete (logged)
//  6. On failure, error logged but event is lost (no retry)
//
// SQL Injection Safety:
//   - Uses parameterized queries with $1, $2, $3, $4 placeholders
//   - PostgreSQL driver (lib/pq) handles proper escaping
//   - No string concatenation or formatting of SQL
//
// Error Handling:
//   - Wraps errors with context using fmt.Errorf and %w
//   - Database errors propagate to ProcessEvent caller
//   - ProcessEvent logs error but does not retry (event lost)
//   - Error examples:
//   - Foreign key violation: "failed to create history item: user_id 999 does not exist"
//   - String too long: "failed to create history item: value too long for type character varying(255)"
//   - Database down: "failed to create history item: connection refused"
//
// Example Usage:
//
//	// After receiving and validating event
//	event := &ProblemSolvedEvent{
//	    UserID:      123,
//	    ProblemText: "25+75",
//	    AnswerText:  "100",
//	}
//
//	err := CreateHistoryItem(db, event.UserID, event.ProblemText, event.AnswerText)
//	if err != nil {
//	    log.Printf("Failed to save history: %v", err)
//	}
//
// Testing Considerations:
//   - Use DBConnection interface to mock database
//   - Test foreign key violations (userID doesn't exist)
//   - Test length constraint violations
//   - Test concurrent inserts (connection pool safety)
//
// Related Functions:
//   - ParseEvent: Validates event before calling this function
//   - ProcessEvent: Orchestrates parsing and database persistence
//   - web-server/database.go:CreateHistoryItem: Similar function (but web-server doesn't use it)
func CreateHistoryItem(db DBConnection, userID int, problem, answer string) error {
	query := "INSERT INTO history (user_id, problem_text, answer_text, created_at) VALUES ($1, $2, $3, $4)"
	_, err := db.Exec(query, userID, problem, answer, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create history item: %w", err)
	}
	return nil
}
