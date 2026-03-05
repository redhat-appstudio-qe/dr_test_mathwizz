package main

// This file defines the data models used throughout the web-server.
// It contains structs for users, history items, and API request/response payloads.

import "time"

// User represents a registered user in the MathWizz system.
// It contains authentication credentials and metadata for user accounts.
// The password is stored as a bcrypt hash for security.
//
// Fields:
//   - ID: Unique identifier assigned by the database (auto-increment)
//   - Email: User's email address, used as login username (must be unique)
//   - PasswordHash: Bcrypt hash of the user's password (never exposed in API responses via json:"-" tag)
//   - CreatedAt: Timestamp when the user account was created
//
// Security Notes:
//   - PasswordHash uses json:"-" tag to prevent accidental exposure in API responses
//   - Email uniqueness is enforced at the database level via UNIQUE constraint
//
// Example:
//
//	user := User{
//	    Email:        "user@example.com",
//	    PasswordHash: "$2a$10$...",  // bcrypt hash
//	    CreatedAt:    time.Now(),
//	}
type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never send password hash in JSON responses
	CreatedAt    time.Time `json:"created_at"`
}

// HistoryItem represents a record of a solved math problem in the user's history.
// Each item captures what problem was solved, the answer, and when it was solved.
// These records are created asynchronously by the history-worker service via NATS events.
//
// Fields:
//   - ID: Unique identifier for this history record (auto-increment)
//   - UserID: Foreign key referencing the user who solved this problem
//   - ProblemText: The mathematical expression that was solved (e.g., "25+75")
//   - AnswerText: The computed result as a string (e.g., "100")
//   - CreatedAt: Timestamp when this problem was solved
//
// Database Constraints:
//   - ProblemText: Maximum 500 characters (VARCHAR(500))
//   - AnswerText: Maximum 100 characters (VARCHAR(100))
//   - Foreign key constraint with ON DELETE CASCADE (deleting user deletes all their history)
//
// Example:
//
//	item := HistoryItem{
//	    UserID:      1,
//	    ProblemText: "25+75",
//	    AnswerText:  "100",
//	    CreatedAt:   time.Now(),
//	}
type HistoryItem struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	ProblemText string    `json:"problem"`
	AnswerText  string    `json:"answer"`
	CreatedAt   time.Time `json:"created_at"`
}

// RegisterRequest is the JSON payload expected by the POST /register endpoint.
// It contains the credentials needed to create a new user account.
//
// Fields:
//   - Email: The email address for the new account (will be validated for format and uniqueness)
//   - Password: The plaintext password (will be hashed with bcrypt before storage)
//
// Validation Rules:
//   - Email must not be empty and should be valid email format
//   - Email must not exceed 255 characters (database limit)
//   - Email must be unique (not already registered)
//   - Password must be at least 6 characters (current implementation)
//   - Password should not exceed 72 bytes (bcrypt limit, not currently enforced)
//
// Security Notes:
//   - Password is transmitted in plaintext over HTTP (should use HTTPS in production)
//   - Password is immediately hashed with bcrypt (cost factor 10) before database storage
//
// Example JSON:
//
//	{
//	    "email": "user@example.com",
//	    "password": "securepassword123"
//	}
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the JSON payload expected by the POST /login endpoint.
// It contains the credentials for authenticating an existing user.
//
// Fields:
//   - Email: The email address of the account to authenticate
//   - Password: The plaintext password to verify against stored bcrypt hash
//
// Authentication Flow:
//  1. Server looks up user by email in database
//  2. Server compares provided password with stored bcrypt hash using bcrypt.CompareHashAndPassword
//  3. If match, server generates JWT token and returns AuthResponse
//  4. If no match, server returns 401 Unauthorized error
//
// Security Notes:
//   - Password is transmitted in plaintext over HTTP (should use HTTPS in production)
//   - Failed login attempts are not rate-limited (vulnerable to brute force)
//   - No account lockout mechanism after multiple failed attempts
//
// Example JSON:
//
//	{
//	    "email": "user@example.com",
//	    "password": "securepassword123"
//	}
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse is the JSON response returned by successful POST /login and POST /register requests.
// It contains the JWT token needed for authenticated API requests and confirms the user's email.
//
// Fields:
//   - Token: JWT (JSON Web Token) used for authentication on subsequent requests
//   - Email: The authenticated user's email address (for display/confirmation purposes)
//
// Token Details:
//   - Format: JWT signed with HS256 algorithm
//   - Claims: Contains user_id and email
//   - Expiration: 24 hours from issuance
//   - Usage: Include in Authorization header as "Bearer <token>" for authenticated endpoints
//
// Client Responsibilities:
//   - Store token securely (current frontend uses localStorage, should consider httpOnly cookies for production)
//   - Include token in Authorization header: "Authorization: Bearer <token>"
//   - Handle token expiration (401 responses after 24 hours)
//   - Clear token on logout
//
// Example JSON:
//
//	{
//	    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
//	    "email": "user@example.com"
//	}
type AuthResponse struct {
	Token string `json:"token"`
	Email string `json:"email"`
}

// SolveRequest is the JSON payload expected by the POST /solve endpoint.
// It contains a mathematical expression to be evaluated.
//
// Fields:
//   - Problem: A mathematical expression string to evaluate (e.g., "25+75", "10*5-20")
//
// Supported Operations:
//   - Addition: +
//   - Subtraction: -
//   - Multiplication: *
//   - Division: /
//   - Parentheses for grouping: ( )
//
// Validation Rules:
//   - Problem must not be empty
//   - Problem should not exceed 500 characters (database constraint for history storage)
//   - Problem should contain only valid mathematical characters
//
// Limitations:
//   - Result is truncated to integer (7.5 becomes 7)
//   - No validation for expression complexity (potential DoS via very complex expressions)
//   - No validation for very large numbers (potential integer overflow)
//
// Example JSON:
//
//	{
//	    "problem": "25+75"
//	}
type SolveRequest struct {
	Problem string `json:"problem"`
}

// SolveResponse is the JSON response returned by successful POST /solve requests.
// It echoes back the problem and provides the computed answer.
//
// Fields:
//   - Problem: The original mathematical expression that was solved (echoed from request)
//   - Answer: The computed result as a string representation of an integer
//
// Response Flow:
//   - Synchronous: This response is returned immediately after computation
//   - Asynchronous: A ProblemSolvedEvent is also published to NATS for history recording
//   - Eventual Consistency: The answer is available immediately, but history is updated asynchronously
//
// Answer Format:
//   - Always an integer string (float results are truncated)
//   - Examples: "100", "42", "-15"
//   - Division results are truncated: 7/2 returns "3" (not "3.5")
//
// Example JSON:
//
//	{
//	    "problem": "25+75",
//	    "answer": "100"
//	}
type SolveResponse struct {
	Problem string `json:"problem"`
	Answer  string `json:"answer"`
}

// ErrorResponse is the standard JSON error response format used by all endpoints.
// It provides a consistent error structure for client-side error handling.
//
// Fields:
//   - Error: A human-readable error message describing what went wrong
//
// HTTP Status Codes:
//   - 400 Bad Request: Invalid input, validation failure
//   - 401 Unauthorized: Missing or invalid JWT token, incorrect credentials
//   - 500 Internal Server Error: Database errors, server failures
//   - (Note: 409 Conflict should be used for duplicate email but currently returns 500)
//
// Error Message Examples:
//   - "email is required"
//   - "password must be at least 6 characters"
//   - "invalid credentials"
//   - "problem cannot be empty"
//   - "failed to create user"
//
// Security Considerations:
//   - Error messages should be informative but not leak sensitive details
//   - Avoid exposing database errors, stack traces, or internal paths
//   - Use generic messages for authentication failures to prevent user enumeration
//
// Example JSON:
//
//	{
//	    "error": "invalid credentials"
//	}
type ErrorResponse struct {
	Error string `json:"error"`
}

// ProblemSolvedEvent is the event published to the NATS message queue when a user solves a math problem.
// This event is consumed asynchronously by the history-worker service to record the solution in the database.
//
// Fields:
//   - UserID: The ID of the user who solved the problem
//   - ProblemText: The mathematical expression that was solved
//   - AnswerText: The computed result as a string
//
// Event Flow:
//  1. User submits problem via POST /solve
//  2. Web-server computes answer synchronously and returns SolveResponse
//  3. Web-server publishes ProblemSolvedEvent to NATS topic "problem_solved" (fire-and-forget, in goroutine)
//  4. History-worker subscribes to "problem_solved" topic and receives event
//  5. History-worker validates event and inserts HistoryItem into database
//
// NATS Topic:
//   - Topic Name: "problem_solved"
//   - Publisher: web-server (via PublishProblemSolved function)
//   - Subscriber: history-worker (via ProcessEvent function)
//
// Validation (performed by history-worker):
//   - UserID must be > 0
//   - ProblemText must not be empty (should also check length ≤500)
//   - AnswerText must not be empty (should also check length ≤100)
//
// Error Handling:
//   - If publishing fails, error is logged but does not affect the synchronous /solve response
//   - If history-worker fails to process, event may be lost (no retry mechanism)
//
// Example JSON (serialized to NATS message):
//
//	{
//	    "user_id": 1,
//	    "problem": "25+75",
//	    "answer": "100"
//	}
type ProblemSolvedEvent struct {
	UserID      int    `json:"user_id"`
	ProblemText string `json:"problem"`
	AnswerText  string `json:"answer"`
}
