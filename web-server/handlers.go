package main

// This file implements all HTTP handlers for the web-server API.
// It handles registration, login, solving math problems, and retrieving history.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/nats-io/nats.go"
	"golang.org/x/crypto/bcrypt"
)

// Server holds dependencies required by all HTTP handlers.
// It implements the dependency injection pattern, allowing handlers to access
// shared resources (database, message queue) without using global variables.
//
// Fields:
//   - DB: PostgreSQL database connection pool for user and history operations
//   - NATS: Message queue connection for publishing ProblemSolvedEvents asynchronously
//
// Design Pattern:
//
//	This struct follows the "dependency injection via struct" pattern common in Go.
//	Handlers are methods on Server, automatically receiving access to dependencies.
//
// Initialization:
//
//	server := &Server{
//	    DB:   dbConn,   // from ConnectDB()
//	    NATS: natsConn, // from ConnectNATS()
//	}
//	router.HandleFunc("/register", server.RegisterHandler)
//	router.HandleFunc("/login", server.LoginHandler)
//	router.HandleFunc("/solve", AuthMiddleware(server.SolveHandler))
//
// Benefits:
//   - Testable: Can inject mock database and NATS connections
//   - No global state: Each Server instance has its own dependencies
//   - Thread-safe: sql.DB and nats.Conn are safe for concurrent use
//
// Usage Example:
//
//	See main.go for complete server initialization
type Server struct {
	DB   *sql.DB
	NATS *nats.Conn
}

// RegisterHandler handles user registration (account creation) requests at POST /register.
// Validates input, hashes password with bcrypt, creates user in database, and returns JWT token.
//
// HTTP Method: POST
// Endpoint: /register
// Authentication: None required (public endpoint)
// Request Body: RegisterRequest JSON
// Response: AuthResponse JSON with token and email
//
// Request Flow:
//  1. Parse RegisterRequest JSON from request body
//  2. Validate email and password presence and format
//  3. Hash password with bcrypt (cost factor 10)
//  4. Create user in database via CreateUser
//  5. Generate JWT token for new user
//  6. Return token and email to client
//
// Input Validation:
//   - Email must not be empty (but format not validated)
//   - Password must not be empty
//   - Password must be at least 6 characters
//   - NO maximum password length check (bcrypt limit: 72 bytes)
//   - NO email format validation (accepts any non-empty string)
//   - NO email length validation (database limit: 255 chars)
//
// HTTP Status Codes:
//   - 201 Created: User successfully registered, token returned
//   - 400 Bad Request: Invalid JSON, missing email/password, password too short
//   - 500 Internal Server Error: Password hashing failed, database error, token generation failed
//
// Error Responses:
//   - "invalid request body": Malformed JSON
//   - "email is required": Empty email
//   - "password is required": Empty password
//   - "password must be at least 6 characters": Password too short
//   - "failed to hash password": Bcrypt error (rare)
//   - "failed to create user": Database error (including duplicate email)
//   - "failed to generate token": JWT signing error (rare)
//
// Example Request:
//
//	POST /register
//	Content-Type: application/json
//
//	{
//	    "email": "newuser@example.com",
//	    "password": "securepassword123"
//	}
//
// Example Success Response (201):
//
//	{
//	    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
//	    "email": "newuser@example.com"
//	}
//
// Example Error Response (400):
//
//	{
//	    "error": "password must be at least 6 characters"
//	}
//
// Production Recommendations:
//   - Require stronger passwords (12+ chars, complexity requirements)
//   - Validate email format with regex
//   - Return 409 Conflict for duplicate emails
//   - Implement rate limiting (5 registrations/hour per IP)
//   - Require email verification before account is usable
//   - Add CAPTCHA to prevent automated registrations
//   - Check password against known breach databases (e.g., HaveIBeenPwned API)
//   - Enforce maximum password length (72 bytes for bcrypt)
func (s *Server) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	switch {
	case req.Email == "":
		respondError(w, "email is required", http.StatusBadRequest)
		return
	case req.Password == "":
		respondError(w, "password is required", http.StatusBadRequest)
		return
	case len(req.Password) < 6:
		respondError(w, "password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	user, err := CreateUser(s.DB, req.Email, string(hash))
	if err != nil {
		respondError(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	token, err := GenerateToken(user.ID, user.Email)
	if err != nil {
		respondError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	setAuthCookies(w, token)
	respondJSON(w, AuthResponse{Token: token, Email: user.Email}, http.StatusCreated)
}

// LoginHandler handles user authentication (login) requests at POST /login.
// Validates credentials using bcrypt and returns JWT token on success.
//
// HTTP Method: POST | Endpoint: /login | Authentication: None (public)
//
// Example Request:
//
//	POST /login
//	{"email": "user@example.com", "password": "password123"}
//
// Status Codes: 200 OK (success), 400 (missing fields), 401 (invalid credentials), 500 (token gen error)
func (s *Server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		respondError(w, "email and password are required", http.StatusBadRequest)
		return
	}

	user, err := GetUserByEmail(s.DB, req.Email)
	if err != nil {
		respondError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := GenerateToken(user.ID, user.Email)
	if err != nil {
		respondError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	setAuthCookies(w, token)
	respondJSON(w, AuthResponse{Token: token, Email: user.Email}, http.StatusOK)
}

// HistoryHandler retrieves authenticated user's problem-solving history at GET /history with pagination.
// Protected by AuthMiddleware - requires valid JWT token in Authorization header.
//
// HTTP Method: GET | Endpoint: /history | Authentication: Required (JWT)
//
// Query Parameters:
//   - limit (optional): Number of items to return (default: 100, max: 200)
//   - offset (optional): Number of items to skip for pagination (default: 0)
//
// Examples:
//   - GET /history                    → Returns first 100 items (default)
//   - GET /history?limit=50           → Returns first 50 items
//   - GET /history?limit=50&offset=50 → Returns items 51-100 (second page)
//   - GET /history?limit=300          → Returns first 200 items (clamped to max)
//
// Returns: Array of HistoryItem (empty array if no history)
// Status Codes:
//   - 200 OK: Successfully retrieved history (may be empty array)
//   - 400 Bad Request: Invalid limit or offset parameter (not a valid integer)
//   - 401 Unauthorized: Missing or invalid JWT token
//   - 500 Internal Server Error: Database error
func (s *Server) HistoryHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse optional pagination parameters from query string
	limit := 0 // 0 means use default (100)
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			respondError(w, "invalid limit parameter", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil {
			respondError(w, "invalid offset parameter", http.StatusBadRequest)
			return
		}
		offset = parsedOffset
	}

	history, err := GetHistoryForUser(s.DB, userID, limit, offset)
	if err != nil {
		respondError(w, "failed to retrieve history", http.StatusInternalServerError)
		return
	}

	if history == nil {
		history = []HistoryItem{}
	}

	respondJSON(w, history, http.StatusOK)
}

// SolveHandler evaluates mathematical expressions at POST /solve.
// Solves synchronously, returns answer, and publishes event to NATS asynchronously for history.
//
// HTTP Method: POST | Endpoint: /solve | Authentication: Required (JWT)
//
// Request Flow:
//  1. AuthMiddleware validates JWT and extracts userID
//  2. Parse SolveRequest JSON
//  3. Call SolveMath to evaluate expression
//  4. Launch goroutine to publish ProblemSolvedEvent to NATS (fire-and-forget)
//  5. Return answer immediately to client
//
// Resilience:
//   - The NATS publishing goroutine has panic recovery to prevent server crashes
//   - If publishing fails or panics, the error is logged but the HTTP response succeeds
//   - The synchronous solve operation is unaffected by NATS failures
//
// Example Request:
//
//	POST /solve
//	Authorization: Bearer <token>
//	{"problem": "25+75"}
//
// Example Response: {"problem": "25+75", "answer": "100"}
// Status Codes: 200 OK, 400 (invalid input), 401 (unauthorized), 500 (internal error)
func (s *Server) SolveHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req SolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Problem == "" {
		respondError(w, "problem is required", http.StatusBadRequest)
		return
	}

	answer, err := SolveMath(req.Problem)
	if err != nil {
		respondError(w, fmt.Sprintf("failed to solve problem: %v", err), http.StatusBadRequest)
		return
	}

	answerStr := fmt.Sprintf("%d", answer)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in PublishProblemSolved goroutine: %v", r)
			}
		}()

		if err := PublishProblemSolved(s.NATS, userID, req.Problem, answerStr); err != nil {
			log.Printf("failed to publish problem solved event: %v", err)
		}
	}()

	respondJSON(w, SolveResponse{Problem: req.Problem, Answer: answerStr}, http.StatusOK)
}

// HealthHandler provides a simple health check endpoint at GET /health.
// Returns {"status": "healthy"} to indicate server is running. No authentication required.
// Used by Kubernetes liveness/readiness probes.
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, map[string]string{"status": "healthy"}, http.StatusOK)
}

// respondJSON is a helper function to send JSON responses with proper Content-Type header.
// Sets Content-Type to application/json, writes status code, and encodes data as JSON.
//
// Parameters:
//   - w: ResponseWriter to write response to
//   - data: Any struct/map/slice to encode as JSON
//   - status: HTTP status code (e.g., http.StatusOK, http.StatusCreated)
//
// Note: Does not check json.Encoder error (silent failure if encoding fails)
func respondJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError is a helper function to send error responses in ErrorResponse JSON format.
// Wraps error message in ErrorResponse struct and calls respondJSON.
//
// Parameters:
//   - w: ResponseWriter
//   - message: Error message string (should be user-friendly, not internal details)
//   - status: HTTP status code (typically 400, 401, 500)
func respondError(w http.ResponseWriter, message string, status int) {
	respondJSON(w, ErrorResponse{Error: message}, status)
}

// setAuthCookies sets secure authentication cookies for the user session.
// Sets two cookies: authToken (httpOnly, contains JWT) and sessionActive (readable by JS).
//
// Security Properties:
//   - authToken: httpOnly prevents XSS attacks from stealing token
//   - Secure flag: Only sent over HTTPS in production
//   - SameSite=Strict: Prevents CSRF attacks
//   - sessionActive: Allows frontend to check authentication status
//
// Cookies Set:
//
//  1. authToken (httpOnly=true):
//     - Contains the actual JWT token for authentication
//     - Cannot be accessed by JavaScript (XSS protection)
//     - MaxAge: 86400 seconds (24 hours, matches JWT expiration)
//     - Used by AuthMiddleware to authenticate requests
//
//  2. sessionActive (httpOnly=false):
//     - Contains "true" to indicate active session
//     - Can be read by JavaScript for UI state (isAuthenticated)
//     - Does NOT contain sensitive data (just a boolean flag)
//     - MaxAge: 86400 seconds (same as authToken)
//
// Parameters:
//   - w: ResponseWriter to set cookies on
//   - token: The JWT token string to store in authToken cookie
//
// Example Usage:
//
//	// After successful login/registration
//	token, _ := GenerateToken(user.ID, user.Email)
//	setAuthCookies(w, token)
//	respondJSON(w, AuthResponse{Email: user.Email}, http.StatusOK)
//
// Frontend Usage:
//
//	// Check if authenticated (reads sessionActive cookie)
//	function isAuthenticated() {
//	    return document.cookie.includes('sessionActive=true');
//	}
//
//	// Make authenticated requests (authToken sent automatically)
//	fetch('/solve', {
//	    method: 'POST',
//	    credentials: 'include',  // Sends cookies
//	    body: JSON.stringify({problem: '2+2'})
//	});
//
// Production Considerations:
//   - Secure flag should be true in production (HTTPS only)
//   - Consider SameSite=Lax for cross-origin subdomains
//   - Implement logout endpoint that clears both cookies
//   - Consider shorter MaxAge (1-2 hours) with refresh token pattern
func setAuthCookies(w http.ResponseWriter, token string) {
	// Set httpOnly cookie with JWT token (XSS-safe)
	http.SetCookie(w, &http.Cookie{
		Name:     "authToken",
		Value:    token,
		HttpOnly: true,                    // JavaScript cannot access (XSS protection)
		Secure:   false,                   // false for local development (HTTP), should be true in production (HTTPS)
		SameSite: http.SameSiteStrictMode, // CSRF protection
		MaxAge:   86400,                   // 24 hours (matches JWT expiration)
		Path:     "/",                     // Available for all endpoints
	})

	// Set non-httpOnly cookie for session status indicator (readable by JS)
	http.SetCookie(w, &http.Cookie{
		Name:     "sessionActive",
		Value:    "true",
		HttpOnly: false,                   // JavaScript can read for isAuthenticated()
		Secure:   false,                   // false for local development (HTTP), should be true in production (HTTPS)
		SameSite: http.SameSiteStrictMode, // CSRF protection
		MaxAge:   86400,                   // 24 hours
		Path:     "/",
	})
}
