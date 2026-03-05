package main

// This file implements JWT authentication middleware and rate limiting.
// It provides functions for generating tokens, protecting endpoints, and preventing brute force attacks.

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
)

// jwtSecret is the secret key used to sign and validate JWT tokens.
// MUST be initialized by calling InitJWTSecret() before any JWT operations.
// This variable is set from the JWT_SECRET environment variable at startup.
var jwtSecret []byte

// InitJWTSecret initializes the JWT secret from the JWT_SECRET environment variable.
// This function MUST be called during application startup before any JWT operations.
//
// Environment Variable:
//   - JWT_SECRET: The secret key used to sign and validate JWT tokens (REQUIRED)
//
// Security Requirements:
//   - Must be at least 32 characters long (256 bits) for HS256 security
//   - Should be cryptographically random (not a dictionary word or simple phrase)
//   - Must be different for each deployment environment (dev, staging, production)
//   - Should be rotated periodically (every 90-180 days recommended)
//
// Validation:
//   - Returns error if JWT_SECRET environment variable is not set
//   - Returns error if JWT_SECRET is less than 32 characters
//   - Sets package-level jwtSecret variable on success
//
// Returns:
//   - error: Non-nil if JWT_SECRET is not set or fails validation
//
// Example Usage:
//
//	func main() {
//	    // Initialize JWT secret at startup
//	    if err := InitJWTSecret(); err != nil {
//	        log.Fatalf("Failed to initialize JWT secret: %v", err)
//	    }
//	    // ... rest of application initialization
//	}
//
// Generating Secure Secrets:
//
//	# Generate a secure 64-character random string (bash)
//	openssl rand -base64 48
//
//	# Or use Go
//	go run -e 'package main; import ("crypto/rand"; "encoding/base64"; "fmt"); func main() { b := make([]byte, 48); rand.Read(b); fmt.Println(base64.StdEncoding.EncodeToString(b)) }'
//
// Deployment Examples:
//
//	# Development (with explicit secret)
//	export JWT_SECRET="dev-secret-at-least-32-characters-long-for-security"
//	go run .
//
//	# Production (from secrets file)
//	export JWT_SECRET=$(cat /run/secrets/jwt-secret)
//	./web-server
//
//	# Kubernetes (from Secret)
//	env:
//	  - name: JWT_SECRET
//	    valueFrom:
//	      secretKeyRef:
//	        name: mathwizz-secrets
//	        key: jwt-secret
//
// Production Best Practices:
//  1. Never commit JWT_SECRET to version control
//  2. Use a secrets management system (Vault, AWS Secrets Manager, etc.)
//  3. Rotate secrets periodically
//  4. Use different secrets for each environment
//  5. Consider using asymmetric signing (RS256) for better key management
//  6. Log secret initialization success/failure (but never log the secret itself)
func InitJWTSecret() error {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return fmt.Errorf("JWT_SECRET environment variable is required")
	}

	if len(secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long (got %d)", len(secret))
	}

	jwtSecret = []byte(secret)
	return nil
}

// contextKey is a custom type for context keys to avoid collisions.
// Using a custom type prevents other packages from accidentally using the same string key.
type contextKey string

// userIDKey is the context key used to store the authenticated user's ID in the request context.
// Set by AuthMiddleware after successful JWT validation, retrieved by GetUserIDFromContext.
const userIDKey contextKey = "userID"

// Claims represents the custom claims structure embedded in JWT tokens.
// It extends jwt.RegisteredClaims with application-specific fields (UserID, Email).
//
// Fields:
//   - UserID: The authenticated user's database ID (primary key from users table)
//   - Email: The user's email address (for display/logging purposes)
//   - RegisteredClaims: Standard JWT claims (exp, iat, iss, sub, etc.)
//
// Token Lifecycle:
//  1. User logs in with email/password
//  2. Server validates credentials and calls GenerateToken(userID, email)
//  3. GenerateToken creates Claims struct with 24-hour expiration
//  4. Token signed with jwtSecret and returned to client
//  5. Client stores token (localStorage) and includes in Authorization header
//  6. AuthMiddleware validates token and extracts Claims
//  7. UserID stored in request context for handlers to access
//
// Security Notes:
//   - Claims are NOT encrypted, only signed (readable by anyone with token)
//   - Do not include sensitive data in claims (passwords, secrets, PII)
//   - Email is safe to include as it's already known to the user
//   - UserID is safe as it's just an internal identifier
//
// Example Token Payload (decoded):
//
//	{
//	    "user_id": 123,
//	    "email": "user@example.com",
//	    "exp": 1672531200,  // Unix timestamp (24 hours from iat)
//	    "iat": 1672444800   // Unix timestamp (issue time)
//	}
type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT token for an authenticated user.
// Called after successful login or registration to issue an authentication token to the client.
//
// Parameters:
//   - userID: The user's database ID (primary key from users table)
//   - email: The user's email address (included in token for display/logging)
//
// Returns:
//   - string: The signed JWT token as a base64-encoded string (e.g., "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...")
//   - error: Non-nil if token signing fails (rare, only if jwtSecret is invalid)
//
// Token Properties:
//   - Algorithm: HS256 (HMAC with SHA-256)
//   - Expiration: 24 hours from issuance (exp claim)
//   - Issue time: Current timestamp (iat claim)
//   - Custom claims: user_id, email
//
// Token Structure:
//   - Header: {"alg": "HS256", "typ": "JWT"}
//   - Payload: {"user_id": <userID>, "email": "<email>", "exp": <timestamp>, "iat": <timestamp>}
//   - Signature: HMACSHA256(base64UrlEncode(header) + "." + base64UrlEncode(payload), jwtSecret)
//
// Example Usage:
//
//	// After successful login
//	user, err := GetUserByEmail(db, email)
//	if err != nil {
//	    return c.JSON(401, ErrorResponse{Error: "invalid credentials"})
//	}
//
//	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
//	if err != nil {
//	    return c.JSON(401, ErrorResponse{Error: "invalid credentials"})
//	}
//
//	// Generate token for authenticated user
//	token, err := GenerateToken(user.ID, user.Email)
//	if err != nil {
//	    return c.JSON(500, ErrorResponse{Error: "failed to generate token"})
//	}
//
//	return c.JSON(200, AuthResponse{
//	    Token: token,
//	    Email: user.Email,
//	})
//
// Client Usage:
//
//	// Store token in localStorage
//	localStorage.setItem('authToken', token)
//
//	// Include in subsequent requests
//	fetch('/solve', {
//	    method: 'POST',
//	    headers: {
//	        'Authorization': 'Bearer ' + token,
//	        'Content-Type': 'application/json'
//	    },
//	    body: JSON.stringify({problem: "2+2"})
//	})
//
// Production Recommendations:
//   - Load jwtSecret from environment variable (never hardcode)
//   - Use shorter expiration (e.g., 1-2 hours) with refresh tokens
//   - Add jti (JWT ID) claim for token tracking/revocation
//   - Add iss (issuer) and aud (audience) claims for token scoping
//   - Consider using asymmetric signing (RS256) for better key management
//   - Implement token refresh endpoint to extend sessions without re-authentication
//   - Maintain token blacklist for revocation before expiration
func GenerateToken(userID int, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken parses, validates, and extracts claims from a JWT token string.
// Called by AuthMiddleware to authenticate incoming requests.
//
// Parameters:
//   - tokenString: The JWT token as a base64-encoded string (without "Bearer " prefix)
//
// Returns:
//   - *Claims: Pointer to parsed Claims struct with user_id and email if token is valid
//   - error: Non-nil if token is invalid, expired, malformed, or has wrong signature
//
// Validation Steps:
//  1. Parse token structure (header, payload, signature)
//  2. Verify signature using jwtSecret (ensures token was issued by this server)
//  3. Check algorithm is HS256 (prevents algorithm confusion attacks)
//  4. Check expiration time (exp claim must be in future)
//  5. Extract and return Claims
//
// Error Cases:
//   - Malformed token: Returns "failed to parse token: <details>"
//   - Invalid signature: Returns "failed to parse token: token signature is invalid"
//   - Expired token: Returns "failed to parse token: token is expired"
//   - Wrong algorithm: Returns "unexpected signing method: <algorithm>"
//   - Invalid claims structure: Returns "invalid token"
//   - Empty/missing token: Returns "failed to parse token: <details>"
//
// Example Usage:
//
//	// From AuthMiddleware
//	authHeader := r.Header.Get("Authorization")  // "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
//	parts := strings.Split(authHeader, " ")
//	tokenString := parts[1]  // "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
//
//	claims, err := ValidateToken(tokenString)
//	if err != nil {
//	    http.Error(w, "invalid or expired token", http.StatusUnauthorized)
//	    return
//	}
//
//	// Token valid, claims.UserID and claims.Email are available
//	log.Printf("Authenticated user %d (%s)", claims.UserID, claims.Email)
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// AuthMiddleware is an HTTP middleware function that protects endpoints requiring authentication.
// It extracts, validates, and processes JWT tokens from httpOnly cookies or Authorization header.
// If valid, the middleware adds the user ID to the request context and calls the next handler.
// If invalid, it returns 401 Unauthorized and stops the request chain.
//
// Token Extraction Priority:
//  1. First tries to read from "authToken" httpOnly cookie (secure, XSS-safe)
//  2. Falls back to "Authorization: Bearer <token>" header (backward compatibility)
//
// Parameters:
//   - next: The HTTP handler function to call if authentication succeeds
//
// Returns:
//   - http.HandlerFunc: A wrapped handler that performs authentication before calling next
//
// Usage Pattern:
//
//	// Protect an endpoint with authentication
//	router.HandleFunc("/solve", AuthMiddleware(solveHandler))
//	router.HandleFunc("/history", AuthMiddleware(historyHandler))
//
//	// Unprotected endpoints (no middleware)
//	router.HandleFunc("/login", loginHandler)
//	router.HandleFunc("/register", registerHandler)
//
// Authentication Flow (Cookie-Based):
//  1. Try to read "authToken" cookie from request
//  2. If found, extract token value
//  3. Validate token using ValidateToken function
//  4. If valid, extract userID from claims and add to request context
//  5. Call next handler with modified context
//  6. If any step fails, return 401 Unauthorized and stop
//
// Authentication Flow (Header-Based, Backward Compatibility):
//  1. If no cookie, extract "Authorization" header from request
//  2. Verify header format is "Bearer <token>"
//  3. Extract token string (part after "Bearer ")
//  4. Validate token using ValidateToken function
//  5. If valid, extract userID from claims and add to request context
//  6. Call next handler with modified context
//  7. If any step fails, return 401 Unauthorized and stop
//
// Request Context Modification:
//
//	After successful authentication, the middleware adds:
//	- Key: userIDKey (contextKey type)
//	- Value: claims.UserID (int)
//	- Handlers can retrieve with: GetUserIDFromContext(ctx)
//
// Error Responses:
//   - Missing cookie and header: 401 "missing authentication credentials"
//   - Invalid header format: 401 "invalid authorization header format"
//   - Invalid/expired token: 401 "invalid or expired token"
//
// Valid Cookie Authentication:
//
//	Cookie: authToken=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...; sessionActive=true
//
// Valid Header Authentication (Backward Compatibility):
//
//	Authorization: Bearer *******AwfQ.signature
//
// Example Protected Handler:
//
//	func solveHandler(w http.ResponseWriter, r *http.Request) {
//	    // AuthMiddleware has already validated token and added userID to context
//	    userID, err := GetUserIDFromContext(r.Context())
//	    if err != nil {
//	        // This should never happen if AuthMiddleware succeeded
//	        http.Error(w, "internal server error", 500)
//	        return
//	    }
//
//	    // userID is authenticated user's ID
//	    log.Printf("User %d solving problem", userID)
//
//	    // Process request knowing user is authenticated
//	    // ...
//	}
//
// Middleware Chain Example:
//
//	// Apply multiple middlewares
//	handler := loggingMiddleware(corsMiddleware(AuthMiddleware(solveHandler)))
//	router.HandleFunc("/solve", handler)
//
// Security Benefits of Cookie-Based Authentication:
//   - httpOnly cookies cannot be accessed by JavaScript (XSS protection)
//   - SameSite=Strict prevents CSRF attacks
//   - Secure flag ensures cookies only sent over HTTPS
//   - Tokens not exposed in localStorage (immune to XSS token theft)
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tokenString string

		// Try to get token from httpOnly cookie first (preferred, secure)
		cookie, err := r.Cookie("authToken")
		if err == nil && cookie.Value != "" {
			tokenString = cookie.Value
		} else {
			// Fall back to Authorization header for backward compatibility
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authentication credentials", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}
			tokenString = parts[1]
		}

		claims, err := ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GetUserIDFromContext extracts the authenticated user's ID from the request context.
// This function should be called from handlers that are protected by AuthMiddleware.
//
// Parameters:
//   - ctx: The request context (from http.Request.Context())
//
// Returns:
//   - int: The authenticated user's ID if found in context
//   - error: Non-nil if user ID not found or has wrong type
//
// Context Key:
//   - Looks for value stored under userIDKey (contextKey type)
//   - This value is set by AuthMiddleware after successful authentication
//
// Typical Call Flow:
//  1. Client sends request with Authorization: Bearer <token>
//  2. AuthMiddleware validates token and extracts userID from claims
//  3. AuthMiddleware stores userID in context: context.WithValue(r.Context(), userIDKey, claims.UserID)
//  4. Handler calls GetUserIDFromContext(r.Context()) to retrieve userID
//
// Error Cases:
//   - userIDKey not in context: Returns "user ID not found in context"
//   - This happens if handler is not protected by AuthMiddleware
//   - Or if AuthMiddleware failed but handler was still called (bug)
//   - Value has wrong type (not int): Returns "user ID not found in context"
//   - This should never happen if AuthMiddleware is working correctly
//
// Usage Pattern:
//
//	func protectedHandler(w http.ResponseWriter, r *http.Request) {
//	    // Extract authenticated user ID
//	    userID, err := GetUserIDFromContext(r.Context())
//	    if err != nil {
//	        // This should not happen if AuthMiddleware is in chain
//	        log.Printf("ERROR: %v", err)
//	        http.Error(w, "internal server error", 500)
//	        return
//	    }
//
//	    // userID is guaranteed to be valid (authenticated by AuthMiddleware)
//	    log.Printf("Processing request for user %d", userID)
//
//	    // Use userID for database queries, authorization checks, etc.
//	    history, err := GetHistoryForUser(db, userID, 100, 0)
//	    // ...
//	}
//
// Example Usage in Solve Handler:
//
//	func (s *Server) SolveHandler(w http.ResponseWriter, r *http.Request) {
//	    // Get authenticated user ID
//	    userID, err := GetUserIDFromContext(r.Context())
//	    if err != nil {
//	        respondWithError(w, 500, "internal server error")
//	        return
//	    }
//
//	    // Parse request
//	    var req SolveRequest
//	    json.NewDecoder(r.Body).Decode(&req)
//
//	    // Solve problem
//	    answer, err := SolveMath(req.Problem)
//
//	    // Publish event with authenticated userID
//	    go PublishProblemSolved(s.NATS, userID, req.Problem, strconv.Itoa(answer))
//
//	    // Return response
//	    respondWithJSON(w, 200, SolveResponse{
//	        Problem: req.Problem,
//	        Answer:  strconv.Itoa(answer),
//	    })
//	}
//
// Design Considerations:
//   - Returns error rather than 0 to distinguish "user 0" from "not found"
//   - Type assertion ensures type safety (prevents runtime panics)
//   - Context key uses custom type (contextKey) to avoid collisions with other packages
//
// Alternative Patterns (not used):
//   - Panic if not found (forces caller to handle, but less flexible)
//   - Return 0 with boolean (userID int, ok bool) - Go's zero-value pattern
//   - Store entire Claims struct in context (more data, but single extraction)
//
// Security Notes:
//   - UserID in context is trusted (came from validated JWT)
//   - No additional validation needed in handlers
//   - UserID can be used directly for database queries and authorization
//
// Testing:
//
//	// Test successful extraction
//	ctx := context.WithValue(context.Background(), userIDKey, 123)
//	userID, err := GetUserIDFromContext(ctx)
//	// userID == 123, err == nil
//
//	// Test missing context value
//	ctx := context.Background()
//	userID, err := GetUserIDFromContext(ctx)
//	// userID == 0, err != nil
//
//	// Test wrong type (should not happen in practice)
//	ctx := context.WithValue(context.Background(), userIDKey, "123")
//	userID, err := GetUserIDFromContext(ctx)
//	// userID == 0, err != nil
func GetUserIDFromContext(ctx context.Context) (int, error) {
	userID, ok := ctx.Value(userIDKey).(int)
	if !ok {
		return 0, fmt.Errorf("user ID not found in context")
	}
	return userID, nil
}

// RateLimiter manages per-IP rate limiters to protect endpoints from brute force attacks.
// It maintains a thread-safe map of rate limiters, one per unique IP address.
// This prevents password brute forcing, user enumeration, and account creation spam.
//
// Design Pattern:
//
//	This implements the "rate limiter per client" pattern using golang.org/x/time/rate.
//	Each IP address gets its own token bucket rate limiter with configurable rate and burst.
//
// Fields:
//   - limiters: Map from IP address (string) to rate limiter instance
//   - mu: Read-write mutex for thread-safe concurrent access to limiters map
//   - rate: The refill rate for token buckets (tokens per second)
//   - burst: The maximum burst size (number of tokens bucket can hold)
//
// Thread Safety:
//   - All access to limiters map is protected by RWMutex
//   - getLimiter uses write lock for map mutations
//   - Multiple goroutines can safely access RateLimiter concurrently
//   - rate.Limiter itself is thread-safe (no additional locking needed)
//
// Token Bucket Algorithm:
//   - Each IP has a bucket that holds up to 'burst' tokens
//   - Bucket refills at 'rate' tokens per second
//   - Each request consumes 1 token
//   - If bucket empty, request is rejected (429 Too Many Requests)
//   - Allows bursts of traffic while enforcing long-term rate limit
//
// Memory Considerations:
//   - Each unique IP address creates a new entry in limiters map
//   - Limiters are never removed (potential memory leak for long-running servers)
//   - For production, consider adding periodic cleanup of inactive limiters
//   - Estimate: ~200 bytes per limiter, 10k IPs = ~2MB memory
//
// Example Configuration:
//
//	// 5 requests per minute (strict)
//	rl := NewRateLimiter(5.0/60.0, 5)  // 0.083 tokens/sec, burst 5
//
//	// 10 requests per minute (moderate)
//	rl := NewRateLimiter(10.0/60.0, 10)
//
//	// 100 requests per hour (very strict)
//	rl := NewRateLimiter(100.0/3600.0, 5)
//
// Usage:
//
//	// Create rate limiter (usually in main.go)
//	authRateLimiter := NewRateLimiter(5.0/60.0, 5)
//
//	// Apply to authentication endpoints
//	router.HandleFunc("/login", RateLimitMiddleware(authRateLimiter)(server.LoginHandler))
//	router.HandleFunc("/register", RateLimitMiddleware(authRateLimiter)(server.RegisterHandler))
//
// Security Properties:
//   - Prevents brute force password attacks (limits login attempts)
//   - Prevents user enumeration (limits registration/login probing)
//   - Prevents account creation spam (limits registration rate)
//   - Per-IP isolation (one slow client doesn't affect others)
//   - Fail closed (rejects when limit exceeded, doesn't fail open)
//
// Limitations:
//   - IP-based limiting can be bypassed by distributed attacks (botnets)
//   - Shared IPs (NAT, corporate networks) may hit limit faster
//   - IPv6 addresses could exhaust memory with many /64 prefixes
//   - No IP normalization (127.0.0.1:8080 and 127.0.0.1:8081 treated as different)
//   - Does not account for X-Forwarded-For (intentional for security)
//
// Production Enhancements:
//   - Add periodic cleanup goroutine to remove inactive limiters
//   - Consider Redis-based rate limiting for multi-instance deployments
//   - Add metrics (total requests, rejected requests per IP)
//   - Log rate limit violations for security monitoring
//   - Consider allowlisting trusted IPs (internal services)
//   - Implement progressive delays instead of hard cutoff
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter with the specified rate and burst size.
// The rate limiter uses the token bucket algorithm to enforce request rate limits per IP address.
//
// Parameters:
//   - r: The rate at which tokens are added to the bucket (tokens per second)
//   - b: The maximum number of tokens the bucket can hold (burst size)
//
// Returns:
//   - *RateLimiter: A new rate limiter instance ready to use with RateLimitMiddleware
//
// Rate and Burst Relationship:
//   - rate controls long-term average rate (e.g., 5 requests per minute = 5/60 = 0.083 tokens/sec)
//   - burst controls maximum requests in short time (e.g., burst=5 allows 5 immediate requests)
//   - After burst consumed, requests limited to refill rate
//
// Common Configurations:
//
//	// Strict: 5 requests per minute for authentication endpoints
//	authLimiter := NewRateLimiter(5.0/60.0, 5)
//	// Allows: 5 requests immediately, then 1 every 12 seconds
//
//	// Moderate: 10 requests per minute for API endpoints
//	apiLimiter := NewRateLimiter(10.0/60.0, 10)
//	// Allows: 10 requests immediately, then 1 every 6 seconds
//
//	// Very Strict: 2 requests per minute for admin endpoints
//	adminLimiter := NewRateLimiter(2.0/60.0, 2)
//	// Allows: 2 requests immediately, then 1 every 30 seconds
//
// Example Usage:
//
//	func main() {
//	    // Create rate limiter for authentication endpoints
//	    authRateLimiter := NewRateLimiter(5.0/60.0, 5)
//
//	    // Apply to login and register endpoints
//	    router.HandleFunc("/login", RateLimitMiddleware(authRateLimiter)(loginHandler))
//	    router.HandleFunc("/register", RateLimitMiddleware(authRateLimiter)(registerHandler))
//
//	    // Start server
//	    http.ListenAndServe(":8080", router)
//	}
//
// Rate Calculation Examples:
//
//	// Requests per minute to tokens per second
//	5 requests/min = 5/60 = 0.0833 tokens/sec
//	10 requests/min = 10/60 = 0.1667 tokens/sec
//
//	// Requests per hour to tokens per second
//	100 requests/hour = 100/3600 = 0.0278 tokens/sec
//	1000 requests/hour = 1000/3600 = 0.2778 tokens/sec
//
// Thread Safety:
//   - The returned RateLimiter is safe for concurrent use
//   - Can be shared across multiple goroutines/handlers
//   - Internal map access is protected by mutex
//
// Memory Usage:
//   - Initial allocation: ~100 bytes (empty map + struct)
//   - Per IP: ~200 bytes (limiter instance + map entry)
//   - Example: 10,000 unique IPs ≈ 2MB memory
//
// Performance:
//   - getLimiter: O(1) average case (map lookup)
//   - Allow() check: O(1) (token bucket calculation)
//   - Negligible overhead: <1ms per request
//
// Security Considerations:
//   - Choose rate based on legitimate use cases (not too strict)
//   - Balance security (prevent brute force) vs usability (don't block legitimate users)
//   - Monitor for attackers using distributed IPs to bypass rate limits
//   - Consider complementary defenses (CAPTCHA, account lockout, progressive delays)
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    b,
	}
}

// getLimiter returns the rate limiter for a given IP address.
// Creates a new limiter if one doesn't exist for this IP.
// Thread-safe: uses mutex to protect concurrent map access.
//
// Parameters:
//   - ip: The client IP address (string format, e.g., "192.168.1.100" or "2001:db8::1")
//
// Returns:
//   - *rate.Limiter: The rate limiter instance for this IP address
//
// Behavior:
//   - First request from IP: Creates new limiter with configured rate and burst
//   - Subsequent requests: Returns existing limiter for this IP
//   - Limiter state persists across requests (tokens consumed remain consumed)
//   - Limiters are never removed (potential memory leak, see RateLimiter docs)
//
// Concurrency:
//   - Uses write lock (mu.Lock) to safely modify limiters map
//   - Multiple goroutines can call this concurrently
//   - Lock is held only during map access (minimal contention)
//
// Example Flow:
//
//	// First request from 192.168.1.100
//	limiter1 := rl.getLimiter("192.168.1.100")  // Creates new limiter, starts with full burst
//
//	// Second request from 192.168.1.100
//	limiter2 := rl.getLimiter("192.168.1.100")  // Returns same limiter, tokens consumed from previous request
//
//	// First request from 192.168.1.101
//	limiter3 := rl.getLimiter("192.168.1.101")  // Creates separate limiter, independent bucket
//
// Performance:
//   - O(1) average case (hash map lookup)
//   - Lock contention: Low (lock held for <1μs typically)
//   - Memory allocation: Only on first request from new IP
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[ip] = limiter
	}

	return limiter
}

// RateLimitMiddleware returns an HTTP middleware function that enforces rate limits per IP address.
// Protects endpoints from brute force attacks, user enumeration, and spam by limiting request frequency.
//
// Parameters:
//   - rl: The RateLimiter instance to use for tracking and enforcing limits
//
// Returns:
//   - Middleware function that wraps http.HandlerFunc with rate limiting
//
// Middleware Behavior:
//   - Extracts client IP address from request.RemoteAddr
//   - Strips port number from IP address (e.g., "192.168.1.100:52341" → "192.168.1.100")
//   - Checks if IP has exceeded rate limit using token bucket algorithm
//   - If limit exceeded: Returns 429 Too Many Requests and stops request
//   - If limit not exceeded: Calls next handler in chain
//
// HTTP Status Codes:
//   - 429 Too Many Requests: Client exceeded rate limit (should retry after delay)
//   - Other: Passed through from next handler if rate limit not exceeded
//
// Error Response:
//   - Status: 429
//   - Body: "rate limit exceeded\n"
//   - Headers: Standard text/plain Content-Type
//
// IP Address Extraction:
//   - Uses request.RemoteAddr (the actual TCP connection IP)
//   - Strips port using net.SplitHostPort
//   - Does NOT check X-Forwarded-For header (security by design)
//   - IPv4 and IPv6 addresses supported
//
// Security Considerations:
//   - **Does not trust X-Forwarded-For**: Prevents IP spoofing attacks
//   - In production behind reverse proxy, proxy should set RemoteAddr correctly
//   - Attackers cannot bypass by setting X-Forwarded-For header
//   - Each TCP connection represents one client IP
//
// Reverse Proxy Compatibility:
//   - Works with nginx, HAProxy, Kubernetes Ingress if configured correctly
//   - Proxy must preserve original client IP in RemoteAddr
//   - nginx: proxy_set_header X-Real-IP $remote_addr; (but we use RemoteAddr, not header)
//   - For correct behavior, reverse proxy must connect from unique IP per client
//
// Usage Example:
//
//	func main() {
//	    // Create rate limiter: 5 requests per minute, burst of 5
//	    authLimiter := NewRateLimiter(5.0/60.0, 5)
//
//	    // Apply to authentication endpoints
//	    router.HandleFunc("/login", RateLimitMiddleware(authLimiter)(server.LoginHandler))
//	    router.HandleFunc("/register", RateLimitMiddleware(authLimiter)(server.RegisterHandler))
//
//	    // Protected endpoints can use different or no rate limiting
//	    router.HandleFunc("/solve", AuthMiddleware(server.SolveHandler))  // No rate limit on /solve
//	}
//
// Middleware Chaining:
//
//	// Rate limit first, then authenticate
//	handler := AuthMiddleware(RateLimitMiddleware(authLimiter)(solveHandler))
//
//	// Rate limit only (no auth)
//	handler := RateLimitMiddleware(authLimiter)(loginHandler)
//
// Attack Scenarios Prevented:
//   - **Password Brute Force**: Attacker tries 10,000 passwords on /login
//   - Blocked after 5 attempts, must wait ~12 seconds between attempts
//   - Makes brute force impractically slow
//   - **User Enumeration**: Attacker tests 10,000 emails on /register
//   - Blocked after 5 attempts, prevents bulk enumeration
//   - **Account Spam**: Attacker creates 10,000 fake accounts on /register
//   - Blocked after 5 attempts, prevents database pollution
//
// Example Client Experience:
//
//	// Legitimate user: 1-2 login attempts (typo), no impact
//	POST /login {"email": "...", "password": "wrong"}  → 401 Unauthorized (allowed)
//	POST /login {"email": "...", "password": "correct"} → 200 OK (allowed)
//
//	// Attacker: Many rapid attempts
//	POST /login (attempt 1) → 401 (allowed)
//	POST /login (attempt 2) → 401 (allowed)
//	...
//	POST /login (attempt 5) → 401 (allowed)
//	POST /login (attempt 6) → 429 Rate Limit Exceeded (BLOCKED)
//	... wait 12 seconds ...
//	POST /login (attempt 7) → 401 (allowed, token refilled)
//
// Production Recommendations:
//   - Monitor 429 responses for potential attacks or legitimate user issues
//   - Log rate limit violations with IP and timestamp
//   - Consider CAPTCHA after multiple rate limit hits
//   - Implement account lockout for additional protection
//   - Use progressive delays (exponential backoff) for repeat offenders
//   - Consider allowlisting known good IPs (internal services, trusted partners)
//
// Limitations:
//   - Shared NAT IPs: Multiple legitimate users behind same IP share rate limit
//   - Distributed attacks: Attacker with botnet can bypass per-IP limiting
//   - IPv6 privacy extensions: One user might appear as many IPs
//   - No cross-instance coordination: Each server instance tracks separately
//
// Performance:
//   - Overhead: <1ms per request (map lookup + token bucket check)
//   - Memory: ~200 bytes per unique IP address
//   - CPU: Negligible (simple arithmetic)
//
// Testing:
//
//	// Test rate limiting in integration tests
//	for i := 0; i < 6; i++ {
//	    resp := makeRequest("/login", ...)
//	    if i < 5 {
//	        assert.Equal(t, 401, resp.StatusCode)  // Normal auth failure
//	    } else {
//	        assert.Equal(t, 429, resp.StatusCode)  // Rate limited
//	    }
//	}
func RateLimitMiddleware(rl *RateLimiter) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Extract IP address from RemoteAddr (includes port)
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				// If SplitHostPort fails, use RemoteAddr as-is (might already be just IP)
				ip = r.RemoteAddr
			}

			// Get rate limiter for this IP
			limiter := rl.getLimiter(ip)

			// Check if request is allowed (consumes 1 token if available)
			if !limiter.Allow() {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Rate limit not exceeded, continue to next handler
			next.ServeHTTP(w, r)
		}
	}
}
