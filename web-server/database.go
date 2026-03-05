package main

// This file handles all database operations for the web-server.
// It provides functions for connecting to PostgreSQL and managing users and history.

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// DBConnection is an interface abstracting database operations for testability.
// It defines the minimal set of database methods needed by this application,
// allowing functions to accept either *sql.DB or mock implementations in tests.
//
// This interface enables dependency injection and makes unit testing possible
// without requiring a real database connection.
//
// Methods:
//   - QueryRow: Executes a query expected to return at most one row (e.g., SELECT by primary key)
//   - Query: Executes a query expected to return multiple rows (e.g., SELECT with multiple results)
//   - Exec: Executes a command that doesn't return rows (INSERT, UPDATE, DELETE)
//
// Usage Pattern:
//   - Production: Pass *sql.DB which implements this interface
//   - Testing: Create mock struct implementing QueryRow, Query, and Exec methods
//
// Security Note:
//   - All methods accept variadic args for SQL parameters, enabling parameterized queries
//   - Always use $1, $2, etc. placeholders in queries to prevent SQL injection
//
// Example:
//
//	// Production usage
//	var db *sql.DB = getProductionDB()
//	user, err := GetUserByEmail(db, "user@example.com")
//
//	// Test usage
//	mockDB := &MockDB{...}
//	user, err := GetUserByEmail(mockDB, "test@example.com")
type DBConnection interface {
	QueryRow(query string, args ...interface{}) *sql.Row
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// ConnectDB establishes and verifies a connection to a PostgreSQL database.
// It creates a connection pool and pings the database to ensure connectivity before returning.
//
// Parameters:
//   - host: Database server hostname or IP address (e.g., "localhost", "database-service")
//   - user: PostgreSQL username for authentication (e.g., "mathwizz_user")
//   - password: Password for the database user (encrypted in transit when SSL is enabled)
//   - dbname: Name of the database to connect to (e.g., "mathwizz_db")
//   - port: PostgreSQL port number (typically 5432)
//   - sslMode: PostgreSQL SSL mode - controls encryption of database traffic
//
// SSL Mode Values:
//   - "disable": No SSL (INSECURE - all data transmitted in plaintext)
//   - "require": Requires SSL, but doesn't verify server certificate (protects against passive eavesdropping)
//   - "verify-ca": Requires SSL and verifies server certificate against CA (recommended for production)
//   - "verify-full": Requires SSL, verifies certificate and hostname match (most secure)
//
// Returns:
//   - *sql.DB: A connection pool handle for executing queries. Callers should defer db.Close()
//   - error: Non-nil if connection fails, with wrapped error details
//
// Connection Behavior:
//   - Creates a connection pool (not a single connection)
//   - Pool size is controlled by Go's default settings (can be configured with SetMaxOpenConns, SetMaxIdleConns)
//   - Pings database to verify connectivity before returning (fails fast if database is unreachable)
//   - Connection is lazy - actual network connections are established on first query
//
// Security Considerations:
//   - SSL mode is configurable via parameter for flexibility across environments
//   - Production deployments should use "require", "verify-ca", or "verify-full"
//   - Development environments may use "disable" for local testing (if explicitly configured)
//   - Password is passed in connection string and visible in process listings (use secret management)
//   - No connection timeout configured - may hang indefinitely if database is unreachable
//   - No retry logic - single failure causes application startup to fail
//   - No validation of input parameters (empty strings, invalid ports)
//   - No connection pool configuration (uses Go defaults which may not be optimal)
//
// Error Cases:
//   - Invalid connection string format: Returns "failed to open database: <details>"
//   - Database unreachable or authentication failure: Returns "failed to ping database: <details>"
//   - Network timeout (no explicit timeout, may hang indefinitely)
//   - SSL handshake failure (if sslmode=require but server doesn't support SSL): Returns "failed to ping database: SSL not supported"
//   - Certificate verification failure (if verify-ca/verify-full and cert invalid): Returns "failed to ping database: certificate verify failed"
//
// Example Usage:
//
//	// Production - require SSL
//	db, err := ConnectDB("database-service", "mathwizz_user", "password123", "mathwizz_db", 5432, "require")
//	if err != nil {
//	    log.Fatalf("Database connection failed: %v", err)
//	}
//	defer db.Close()
//
//	// Development - local unencrypted connection
//	db, err := ConnectDB("localhost", "mathwizz_user", "password123", "mathwizz_db", 5432, "disable")
//
// Production Recommendations:
//   - Use sslMode="require" (minimum) or "verify-full" (best)
//   - Add connection timeout: connStr += " connect_timeout=10"
//   - Configure connection pool: db.SetMaxOpenConns(25), db.SetMaxIdleConns(5), db.SetConnMaxLifetime(5*time.Minute)
//   - Validate input parameters before constructing connection string
//   - Load credentials from environment variables or secret management system
//   - For verify-ca/verify-full: mount CA certificate and configure sslrootcert parameter
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

// GetUserByEmail retrieves a user record from the database by their email address.
// This function is used during login to look up users by their email (username).
//
// Parameters:
//   - db: Database connection or mock implementing DBConnection interface
//   - email: The email address to search for (case-sensitive, not validated or sanitized)
//
// Returns:
//   - *User: Pointer to User struct with all fields populated (ID, Email, PasswordHash, CreatedAt)
//   - error: Non-nil if user not found or database error occurs
//
// Query Details:
//   - Uses parameterized query with $1 placeholder (prevents SQL injection)
//   - Selects: id, email, password_hash, created_at
//   - WHERE clause: email = $1 (exact match, case-sensitive)
//   - PasswordHash is included in result (for bcrypt comparison during login)
//
// Error Handling:
//   - sql.ErrNoRows: Returns error "user not found" (distinct error for user not existing)
//   - Other database errors: Returns "database error: <wrapped error>"
//   - Distinguishes between "user doesn't exist" vs "database failure"
//
// Security Considerations:
//   - Email parameter is NOT validated (no format check, length check, or sanitization)
//   - Accepts empty strings, malformed emails, very long strings (potential DoS)
//   - Returns password hash to caller (appropriate for login flow, but must not be exposed to clients)
//   - No rate limiting (vulnerable to user enumeration via timing attacks)
//   - Error message "user not found" enables user enumeration (attacker can determine if email exists)
//
// Example Usage:
//
//	// Login flow
//	user, err := GetUserByEmail(db, "user@example.com")
//	if err != nil {
//	    // User not found or database error
//	    return c.JSON(401, ErrorResponse{Error: "invalid credentials"})
//	}
//
//	// Compare provided password with stored hash
//	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(providedPassword))
//	if err != nil {
//	    return c.JSON(401, ErrorResponse{Error: "invalid credentials"})
//	}
//
//	// User authenticated successfully
//	token := generateJWT(user.ID, user.Email)
//
// Edge Cases:
//   - Empty email string: Queries database with empty string (unlikely to match, but wastes resources)
//   - Email with SQL-like characters: Safe due to parameterized query ($1)
//   - Very long email (>255 chars): Queries database unnecessarily (should fail validation first)
//   - Email with Unicode: Depends on database collation settings
func GetUserByEmail(db DBConnection, email string) (*User, error) {
	user := &User{}
	query := "SELECT id, email, password_hash, created_at FROM users WHERE email = $1"

	err := db.QueryRow(query, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	switch {
	case err == sql.ErrNoRows:
		return nil, fmt.Errorf("user not found")
	case err != nil:
		return nil, fmt.Errorf("database error: %w", err)
	default:
		return user, nil
	}
}

// CreateUser inserts a new user record into the database and returns the created user with database-assigned ID.
// This function is called during registration after the password has been hashed with bcrypt.
//
// Parameters:
//   - db: Database connection or mock implementing DBConnection interface
//   - email: The email address for the new user (not validated, should be unique)
//   - passwordHash: Bcrypt hash of the user's password (should be from bcrypt.GenerateFromPassword)
//
// Returns:
//   - *User: Pointer to the newly created User struct with ID populated from database
//   - error: Non-nil if insert fails (e.g., duplicate email, database error)
//
// Behavior:
//   - Creates User struct with provided email and passwordHash
//   - Sets CreatedAt to current timestamp (time.Now())
//   - Executes INSERT with RETURNING id clause to get database-assigned ID
//   - Returns complete User struct including the new ID
//
// Database Operation:
//   - INSERT INTO users (email, password_hash, created_at) VALUES ($1, $2, $3) RETURNING id
//   - Uses parameterized query (prevents SQL injection)
//   - Database assigns auto-increment ID (SERIAL primary key)
//   - Database enforces UNIQUE constraint on email (duplicate email causes error)
//
// Error Handling:
//   - Duplicate email: Database returns unique constraint violation (error wrapping hides this detail)
//   - Other database errors: Returns "failed to create user: <wrapped error>"
//   - Error message is generic and doesn't distinguish between constraint violations and other errors
//
// Security Considerations:
//   - **NO INPUT VALIDATION**: Accepts any string for email and passwordHash
//   - Empty email: Would insert empty string (should fail validation first)
//   - Empty passwordHash: Would insert empty hash (authentication would always fail)
//   - Email not validated for format, length (database VARCHAR(255) limit)
//   - PasswordHash not validated for format, length (should be bcrypt hash ~60 chars)
//   - Malformed inputs cause database errors instead of clear validation errors
//
// Example Usage:
//
//	// Registration flow
//	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), 10)
//	if err != nil {
//	    return fmt.Errorf("failed to hash password: %w", err)
//	}
//
//	user, err := CreateUser(db, "newuser@example.com", string(hashedPassword))
//	if err != nil {
//	    // Could be duplicate email or database error - can't distinguish
//	    return c.JSON(500, ErrorResponse{Error: "failed to create user"})
//	}
//
//	// User created successfully, user.ID is now populated
//	token := generateJWT(user.ID, user.Email)
//	return c.JSON(200, AuthResponse{Token: token, Email: user.Email})
//
// Recommended Improvements:
//   - Validate email: `if !isValidEmail(email) { return nil, fmt.Errorf("invalid email format") }`
//   - Validate hash: `if !strings.HasPrefix(passwordHash, "$2") { return nil, fmt.Errorf("invalid password hash") }`
//   - Check length: `if len(email) > 255 { return nil, fmt.Errorf("email too long") }`
//   - Parse error to distinguish duplicate email: `if isDuplicateKeyError(err) { return nil, ErrDuplicateEmail }`
func CreateUser(db DBConnection, email, passwordHash string) (*User, error) {
	user := &User{
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}

	query := "INSERT INTO users (email, password_hash, created_at) VALUES ($1, $2, $3) RETURNING id"
	err := db.QueryRow(query, user.Email, user.PasswordHash, user.CreatedAt).Scan(&user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

const (
	// MaxHistoryLimit is the maximum number of history items that can be retrieved in a single request
	// This prevents memory exhaustion and DoS attacks from users with thousands of history items
	MaxHistoryLimit = 200

	// DefaultHistoryLimit is the default number of history items returned when no limit is specified
	DefaultHistoryLimit = 100
)

// GetHistoryForUser retrieves problem-solving history records for a specific user from the database
// with pagination support. Results are ordered by creation timestamp in descending order (most recent first).
//
// Parameters:
//   - db: Database connection or mock implementing DBConnection interface
//   - userID: The ID of the user whose history to retrieve (not validated, assumes valid user ID)
//   - limit: Maximum number of items to return (clamped to MaxHistoryLimit=200 to prevent DoS)
//   - offset: Number of items to skip for pagination (e.g., offset=0 for first page, offset=100 for second page)
//
// Returns:
//   - []HistoryItem: Slice of history items (empty slice if user has no history or offset exceeds total)
//   - error: Non-nil if database query fails, row scanning fails, or iteration encounters error
//
// Pagination Behavior:
//   - limit <= 0: Uses DefaultHistoryLimit (100)
//   - limit > MaxHistoryLimit: Clamped to MaxHistoryLimit (200) to prevent abuse
//   - offset < 0: Treated as 0
//   - offset >= total items: Returns empty slice (not an error)
//
// Query Details:
//   - SELECT id, user_id, problem_text, answer_text, created_at FROM history WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
//   - Uses parameterized query with $1, $2, $3 placeholders (prevents SQL injection)
//   - ORDER BY created_at DESC ensures newest items appear first
//   - LIMIT clause prevents unbounded result sets (DoS protection)
//   - OFFSET clause enables pagination
//
// Behavior:
//   - Validates and clamps limit to prevent abuse
//   - Validates offset to prevent negative values
//   - Queries database with LIMIT and OFFSET
//   - Iterates through rows using rows.Next()
//   - Scans each row into HistoryItem struct and appends to slice
//   - Returns empty slice (not nil) if user has no history or offset exceeds total
//   - Properly closes rows with defer rows.Close()
//   - Checks rows.Err() after iteration to catch any errors during iteration
//
// Error Handling:
//   - Query execution failure: Returns "failed to query history: <wrapped error>"
//   - Row scanning failure: Returns "failed to scan history item: <wrapped error>"
//   - Iteration error: Returns "error iterating history rows: <wrapped error>"
//   - No error if user has no history (returns empty slice)
//
// Security Improvements:
//   - LIMIT clause prevents loading all history items at once (fixes DoS vulnerability)
//   - Maximum limit (200) prevents abuse even with malicious limit parameter
//   - OFFSET enables pagination without loading entire dataset
//   - Still no validation of userID (accepts 0, negative, non-existent user IDs)
//
// Performance Improvements:
//   - Bounded memory usage (max 200 items per request)
//   - Database can use LIMIT optimization
//   - Faster response times for users with large history
//   - Better UX with paginated results
//
// Example Usage:
//
//	// Get first page (most recent 100 items)
//	history, err := GetHistoryForUser(db, 123, 100, 0)
//	if err != nil {
//	    return c.JSON(500, ErrorResponse{Error: "failed to get history"})
//	}
//
//	// Get second page (next 100 items)
//	history, err := GetHistoryForUser(db, 123, 100, 100)
//
//	// Use default limit (100 items)
//	history, err := GetHistoryForUser(db, 123, 0, 0)
//
//	// Request 500 items (will be clamped to 200)
//	history, err := GetHistoryForUser(db, 123, 500, 0)
//
//	// Return with pagination metadata
//	return c.JSON(200, map[string]interface{}{
//	    "items": history,
//	    "limit": limit,
//	    "offset": offset,
//	    "count": len(history),
//	    "hasMore": len(history) == limit,
//	})
//
// Database Index:
//   - Should have index on (user_id, created_at) for efficient ORDER BY with LIMIT/OFFSET
func GetHistoryForUser(db DBConnection, userID, limit, offset int) ([]HistoryItem, error) {
	// Validate and clamp limit to prevent DoS
	if limit <= 0 {
		limit = DefaultHistoryLimit
	}
	if limit > MaxHistoryLimit {
		limit = MaxHistoryLimit
	}

	// Validate offset
	if offset < 0 {
		offset = 0
	}

	query := "SELECT id, user_id, problem_text, answer_text, created_at FROM history WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3"
	rows, err := db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var history []HistoryItem
	for rows.Next() {
		var item HistoryItem
		if err := rows.Scan(&item.ID, &item.UserID, &item.ProblemText, &item.AnswerText, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan history item: %w", err)
		}
		history = append(history, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating history rows: %w", err)
	}

	return history, nil
}

// CreateHistoryItem inserts a new problem-solving history record into the database.
// This function is primarily called by the history-worker service when processing ProblemSolvedEvent
// messages from NATS, but is defined in this shared package for reusability.
//
// Parameters:
//   - db: Database connection or mock implementing DBConnection interface
//   - userID: The ID of the user who solved the problem (not validated)
//   - problem: The mathematical expression that was solved (e.g., "25+75")
//   - answer: The computed result as a string (e.g., "100")
//
// Returns:
//   - error: Non-nil if insert fails, nil if successful
//
// Database Operation:
//   - INSERT INTO history (user_id, problem_text, answer_text, created_at) VALUES ($1, $2, $3, $4)
//   - Uses parameterized query (prevents SQL injection)
//   - Sets created_at to time.Now() at moment of insertion
//   - Does NOT use RETURNING clause (ID is not returned to caller)
//
// Behavior:
//   - Executes INSERT statement with provided parameters
//   - Returns error if database operation fails
//   - **WARNING**: Ignores sql.Result returned by Exec (doesn't verify row was inserted)
//
// Security Considerations:
//   - **NO INPUT VALIDATION**: Accepts any values for userID, problem, answer
//   - userID not validated (could be 0, negative, or non-existent user)
//   - problem not validated for length (database limit: VARCHAR(500))
//   - answer not validated for length (database limit: VARCHAR(100))
//   - Very long strings cause database errors instead of validation errors
//   - No check for empty strings (though database allows NOT NULL empty strings)
//
// Silent Failure Risk:
//   - If INSERT silently fails (e.g., database trigger, constraint, replication issue),
//     function returns nil (success) without detecting failure
//   - History event appears processed but is actually lost
//   - No logging or alerting of silent failures
//
// Database Constraints:
//   - Foreign key constraint: user_id must reference existing user (violation causes error)
//   - Length constraints: problem_text VARCHAR(500), answer_text VARCHAR(100)
//   - Exceeding lengths causes database error (not caught by validation)
//
// Error Cases:
//   - Foreign key violation (userID doesn't exist): Returns "failed to create history item: <details>"
//   - Length constraint violation: Returns "failed to create history item: <details>"
//   - Database connection failure: Returns "failed to create history item: <details>"
//   - Silent failure (0 rows affected): Returns nil (SUCCESS) - **BUG**
//
// Example Usage (history-worker):
//
//	// Process NATS event
//	var event ProblemSolvedEvent
//	json.Unmarshal(msg.Data, &event)
//
//	// Validate event (history-worker does basic validation)
//	if err := validateEvent(event); err != nil {
//	    log.Printf("Invalid event: %v", err)
//	    return
//	}
//
//	// Insert into database
//	err := CreateHistoryItem(db, event.UserID, event.ProblemText, event.AnswerText)
//	if err != nil {
//	    log.Printf("Failed to create history: %v", err)
//	    // Event is lost - no retry mechanism
//	}
//
// Recommended Improvements:
//
//	```go
//	func CreateHistoryItem(db DBConnection, userID int, problem, answer string) error {
//	    // Validate inputs
//	    if userID <= 0 {
//	        return fmt.Errorf("invalid userID: %d", userID)
//	    }
//	    if len(problem) == 0 || len(problem) > 500 {
//	        return fmt.Errorf("invalid problem length: %d", len(problem))
//	    }
//	    if len(answer) == 0 || len(answer) > 100 {
//	        return fmt.Errorf("invalid answer length: %d", len(answer))
//	    }
//
//	    // Execute INSERT
//	    query := "INSERT INTO history (user_id, problem_text, answer_text, created_at) VALUES ($1, $2, $3, $4)"
//	    result, err := db.Exec(query, userID, problem, answer, time.Now())
//	    if err != nil {
//	        return fmt.Errorf("failed to create history item: %w", err)
//	    }
//
//	    // Verify exactly 1 row was inserted
//	    rows, err := result.RowsAffected()
//	    if err != nil {
//	        return fmt.Errorf("failed to get rows affected: %w", err)
//	    }
//	    if rows != 1 {
//	        return fmt.Errorf("expected 1 row inserted, got %d", rows)
//	    }
//
//	    return nil
//	}
//	```
func CreateHistoryItem(db DBConnection, userID int, problem, answer string) error {
	query := "INSERT INTO history (user_id, problem_text, answer_text, created_at) VALUES ($1, $2, $3, $4)"
	_, err := db.Exec(query, userID, problem, answer, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create history item: %w", err)
	}
	return nil
}
