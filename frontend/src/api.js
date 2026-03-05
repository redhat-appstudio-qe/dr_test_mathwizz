/**
 * API client for backend communication with the MathWizz web-server.
 *
 * This module provides a centralized interface for all HTTP requests to the backend API.
 * It handles authentication token management (localStorage-based), request formatting,
 * error handling, and response parsing.
 *
 * @module api
 */

/**
 * Base URL for API requests to the web-server.
 *
 * Determined by environment variable REACT_APP_API_URL (set at build time) or defaults to
 * http://localhost:8080 for development. Note that React environment variables are baked
 * into the bundle at build time and cannot be changed at runtime.
 *
 * **Configuration**:
 * - Development: Uses default http://localhost:8080
 * - Production: Set REACT_APP_API_URL at Docker build time via --build-arg
 *
 * @constant {string}
 * @default 'http://localhost:8080'
 */
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

/**
 * Saves the authentication token (no-op for cookie-based auth).
 *
 * NOTE: This function is a no-op. The JWT token is now automatically stored in an httpOnly
 * cookie by the backend server (RegisterHandler/LoginHandler). The frontend does not need
 * to manually save tokens. This function is kept for backward compatibility.
 *
 * @function
 * @param {string} token - The JWT token string (ignored - token is set via httpOnly cookie by backend)
 * @returns {void}
 *
 * @example
 * // After successful login
 * const data = await login(email, password);
 * saveToken(data.token); // No-op - backend already set httpOnly cookie
 */
const saveToken = (token) => {
  // No-op: Token is automatically set as httpOnly cookie by backend
  // Cookie is sent automatically with credentials: 'include' on all requests
};

/**
 * Clears the session indicator cookie to log out the user (client-side only).
 *
 * NOTE: This only clears the sessionActive cookie (which JavaScript can access).
 * The httpOnly authToken cookie cannot be cleared by JavaScript. For complete logout,
 * call a backend /logout endpoint that clears both cookies server-side.
 *
 * Current Implementation:
 *   - Clears sessionActive cookie (makes isAuthenticated() return false)
 *   - Cannot clear httpOnly authToken cookie (requires backend /logout endpoint)
 *   - Token remains valid on server until expiration (24 hours)
 *
 * @function
 * @returns {void}
 *
 * @example
 * // User clicks logout button
 * clearToken();  // Clears sessionActive cookie
 * // Redirect to login page
 * // For complete logout, call: await fetch('/logout', {credentials: 'include'})
 */
export const clearToken = () => {
  // Clear sessionActive cookie by setting it to expired
  document.cookie = 'sessionActive=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT; SameSite=Strict';

  // NOTE: Cannot clear httpOnly authToken cookie from JavaScript
  // For complete logout, backend should provide /logout endpoint that clears both cookies
};

/**
 * Safely parses error responses from the server, handling both JSON and non-JSON responses.
 *
 * This function attempts to parse error responses as JSON first. If the response is not valid
 * JSON (e.g., HTML error pages from Nginx, plain text errors, network errors), it falls back
 * to extracting the text content. This prevents SyntaxError exceptions from masking the actual
 * error and provides better error messages to users and developers.
 *
 * **Error Response Handling**:
 * 1. First attempts to parse response.json() to extract {"error": "message"} format
 * 2. If JSON parsing throws SyntaxError (non-JSON response), falls back to response.text()
 * 3. Returns error.error field if present (backend convention), otherwise returns raw text
 *
 * **Use Cases**:
 * - Backend returns JSON error: {"error": "invalid credentials"} → "invalid credentials"
 * - Nginx returns HTML 502: "<html><body>Bad Gateway</body></html>" → Full HTML text
 * - Network error returns plain text: "Service Unavailable" → "Service Unavailable"
 * - Empty response: "" → fallbackMessage
 *
 * @async
 * @function
 * @param {Response} response - Fetch API Response object (with !response.ok status)
 * @param {string} fallbackMessage - Default error message if parsing fails or response is empty
 * @returns {Promise<string>} Error message extracted from response or fallback message
 *
 * @example
 * // JSON error response (normal case)
 * const response = await fetch('/api/login', {...});
 * if (!response.ok) {
 *   const errorMessage = await parseErrorResponse(response, 'Login failed');
 *   // Returns: "invalid credentials" (from {"error": "invalid credentials"})
 *   throw new Error(errorMessage);
 * }
 *
 * @example
 * // Non-JSON error response (Nginx 502 Bad Gateway)
 * const response = await fetch('/api/solve', {...});
 * if (!response.ok) {
 *   const errorMessage = await parseErrorResponse(response, 'Solve failed');
 *   // Returns: "<html><body>502 Bad Gateway</body></html>" (full HTML text)
 *   // or "Solve failed" if text is empty
 * }
 */
const parseErrorResponse = async (response, fallbackMessage) => {
  try {
    const error = await response.json();
    return error.error || fallbackMessage;
  } catch (jsonError) {
    try {
      const text = await response.text();
      return text || fallbackMessage;
    } catch (textError) {
      return fallbackMessage;
    }
  }
};

/**
 * Checks if the user is currently authenticated by verifying sessionActive cookie presence.
 *
 * Returns true if the sessionActive cookie exists, false otherwise. This cookie is set by
 * the backend during login/registration alongside the httpOnly authToken cookie. The
 * sessionActive cookie is readable by JavaScript (not httpOnly) and serves as a session
 * indicator for client-side UI state.
 *
 * Security Note:
 *   - This only checks for session indicator cookie (sessionActive)
 *   - The actual JWT token is in httpOnly authToken cookie (inaccessible to JavaScript)
 *   - sessionActive cookie contains "true" (not sensitive data)
 *   - Actual authentication validation happens server-side in AuthMiddleware
 *
 * Limitations:
 *   - Does not verify token is valid or unexpired (server validates)
 *   - If sessionActive cookie is manually deleted, this returns false even if authToken is valid
 *   - If token expires (>24 hours), sessionActive may still exist until next API call returns 401
 *
 * Usage:
 *   - Used by App.jsx on mount to determine initial authentication state
 *   - Used for client-side routing decisions (show login page vs protected pages)
 *   - Does NOT prevent API calls with invalid tokens (server validates)
 *
 * @function
 * @returns {boolean} true if sessionActive cookie exists, false otherwise
 *
 * @example
 * if (isAuthenticated()) {
 *   // Show authenticated UI
 *   return <SolverPage />;
 * } else {
 *   // Show login page
 *   return <LoginPage />;
 * }
 *
 * @example
 * // Check authentication before making API call
 * if (!isAuthenticated()) {
 *   throw new Error('Not authenticated');
 * }
 * await solve(problem);
 */
export const isAuthenticated = () => {
  // Check if sessionActive cookie exists (set by backend during login/registration)
  return document.cookie.split('; ').some(cookie => cookie.startsWith('sessionActive='));
};

/**
 * Registers a new user account and authenticates them.
 *
 * Sends POST request to /register endpoint with email and password. On success, the server
 * creates a new user account, hashes the password with bcrypt, and returns a JWT token.
 * The token is automatically saved to localStorage for subsequent authenticated requests.
 *
 * **Request Flow**:
 * 1. Client sends credentials to POST /register
 * 2. Server validates inputs (checks non-empty, minimum password length 6)
 * 3. Server hashes password with bcrypt
 * 4. Server creates user in database
 * 5. Server generates JWT token (valid for 24 hours)
 * 6. Client saves token to localStorage
 *
 * **Validation**:
 * - Client-side validation should be performed BEFORE calling this (see utils.js:
 *   validateEmail, validatePassword)
 * - Server-side validation checks for non-empty fields and minimum password length
 *
 * @async
 * @function
 * @param {string} email - User's email address (should be validated client-side first)
 * @param {string} password - User's password (should be validated client-side first, min 6 chars)
 * @returns {Promise<Object>} Registration response containing token and user info
 * @returns {string} return.token - JWT authentication token (saved to localStorage automatically)
 * @returns {string} return.email - User's email address
 * @returns {number} return.user_id - Newly created user's database ID
 *
 * @throws {Error} If network request fails or server returns error (JSON or non-JSON response)
 * @throws {Error} If email already registered
 *
 * @example
 * // With client-side validation
 * import { register } from './api';
 * import { validateEmail, validatePassword } from './utils';
 *
 * const emailError = validateEmail(email);
 * const passwordError = validatePassword(password);
 *
 * if (!emailError && !passwordError) {
 *   try {
 *     const data = await register(email, password);
 *     console.log('Registered with user ID:', data.user_id);
 *     // Token automatically saved, user is now authenticated
 *   } catch (error) {
 *     console.error('Registration failed:', error.message);
 *   }
 * }
 *
 * @example
 * // Response structure on success
 * {
 *   "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
 *   "email": "user@example.com",
 *   "user_id": 123
 * }
 *
 * @example
 * // Error response structure
 * {
 *   "error": "failed to create user"
 * }
 */
export const register = async (email, password) => {
  const response = await fetch(`${API_BASE_URL}/register`, {
    method: 'POST',
    credentials: 'include',  // Send and receive cookies (authToken, sessionActive)
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });

  if (!response.ok) {
    const errorMessage = await parseErrorResponse(response, 'Registration failed');
    throw new Error(errorMessage);
  }

  const data = await response.json();
  saveToken(data.token);  // No-op: Token automatically set as httpOnly cookie by backend
  return data;
};

/**
 * Authenticates an existing user and retrieves a session token.
 *
 * Sends POST request to /login endpoint with email and password. Server verifies credentials
 * against database (compares bcrypt hash), and on success returns a JWT token valid for 24 hours.
 * The token is automatically saved to localStorage for subsequent authenticated requests.
 *
 * **Authentication Flow**:
 * 1. Client sends credentials to POST /login
 * 2. Server looks up user by email (GetUserByEmail)
 * 3. Server compares password with bcrypt hash (constant-time comparison)
 * 4. Server returns generic "invalid credentials" for both "user not found" and "wrong password"
 *    to prevent user enumeration
 * 5. Server generates JWT token with user_id and email claims
 * 6. Client saves token to localStorage
 *
 * **Error Handling**:
 * - Server returns 401 Unauthorized with `{"error": "invalid credentials"}` for both wrong email
 *   and wrong password (prevents user enumeration)
 * - Client throws Error with error message from server or generic "Login failed"
 *
 * @async
 * @function
 * @param {string} email - User's email address
 * @param {string} password - User's password (plaintext - hashed server-side with bcrypt)
 * @returns {Promise<Object>} Login response containing token and user info
 * @returns {string} return.token - JWT authentication token (saved to localStorage automatically)
 * @returns {string} return.email - User's email address
 * @returns {number} return.user_id - User's database ID
 *
 * @throws {Error} If credentials are invalid or network request fails (handles both JSON and non-JSON error responses)
 *
 * @example
 * // Successful login
 * try {
 *   const data = await login('user@example.com', 'password123');
 *   console.log('Logged in as:', data.email);
 *   // Token automatically saved to localStorage
 *   // User can now call authenticated endpoints (solve, getHistory)
 * } catch (error) {
 *   console.error('Login failed:', error.message);
 *   // Displays "invalid credentials" for both wrong email and wrong password
 * }
 *
 * @example
 * // Success response structure
 * {
 *   "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
 *   "email": "user@example.com",
 *   "user_id": 123
 * }
 *
 * @example
 * // Error response structure (both wrong email and wrong password)
 * {
 *   "error": "invalid credentials"
 * }
 */
export const login = async (email, password) => {
  const response = await fetch(`${API_BASE_URL}/login`, {
    method: 'POST',
    credentials: 'include',  // Send and receive cookies (authToken, sessionActive)
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });

  if (!response.ok) {
    const errorMessage = await parseErrorResponse(response, 'Login failed');
    throw new Error(errorMessage);
  }

  const data = await response.json();
  saveToken(data.token);  // No-op: Token automatically set as httpOnly cookie by backend
  return data;
};

/**
 * Solves a mathematical expression and returns the result (authenticated endpoint).
 *
 * Sends POST request to /solve endpoint with a math expression string. Server synchronously
 * evaluates the expression using govaluate library and returns the integer result. Server also
 * asynchronously publishes a ProblemSolvedEvent to NATS for history recording by history-worker
 * (eventual consistency - history appears after a delay, typically <100ms).
 *
 * **Requires Authentication**: Must have valid JWT token in localStorage. If not authenticated
 * or token is expired/invalid, throws "Not authenticated" error (client-side) or receives 401
 * Unauthorized (server-side).
 *
 * **Problem Solving Flow**:
 * 1. Client validates problem client-side (validateProblem should be called first)
 * 2. Client sends problem to POST /solve with Authorization: Bearer <token> header
 * 3. Server validates JWT token (AuthMiddleware)
 * 4. Server synchronously solves problem (SolveMath function)
 * 5. Server asynchronously publishes event to NATS (fire-and-forget, no delivery guarantee)
 * 6. History-worker eventually processes event and writes to database (eventual consistency)
 * 7. Client receives answer immediately (does NOT wait for history recording)
 *
 * **Supported Operations**:
 * - Addition: `2+3` → 5
 * - Subtraction: `10-4` → 6
 * - Multiplication: `5*6` → 30
 * - Division: `10/2` → 5 (Note: Result truncated to int, 7/2 → 3)
 * - Parentheses: `(2+3)*4` → 20
 * - Decimal numbers: `2.5+3.5` → 6 (truncated to int)
 *
 * **Validation**:
 * - Client-side: validateProblem() should be called BEFORE this function
 * - Server-side: Minimal validation (only checks non-empty)
 *
 * @async
 * @function
 * @param {string} problem - Mathematical expression to evaluate (e.g., "2+3*4")
 * @returns {Promise<Object>} Solution response containing the answer
 * @returns {number} return.answer - The calculated result as an integer
 *
 * @throws {Error} If not authenticated (no token in localStorage)
 * @throws {Error} If token is invalid or expired (401 Unauthorized from server)
 * @throws {Error} If problem is invalid (empty, invalid characters, parse error)
 * @throws {Error} If network request fails (handles both JSON and non-JSON error responses)
 *
 * @example
 * // Successful problem solving
 * import { solve } from './api';
 * import { validateProblem } from './utils';
 *
 * const problem = "2+3*4";
 * const error = validateProblem(problem);
 *
 * if (!error) {
 *   try {
 *     const result = await solve(problem);
 *     console.log('Answer:', result.answer); // 14
 *     // History will appear in getHistory() after ~100ms delay
 *   } catch (error) {
 *     console.error('Solve failed:', error.message);
 *   }
 * }
 *
 * @example
 * // Success response structure
 * {
 *   "answer": 14
 * }
 *
 * @example
 * // Error responses
 * // Not authenticated (client-side check)
 * Error: 'Not authenticated'
 *
 * // Invalid token (server 401)
 * {
 *   "error": "invalid or expired token"
 * }
 *
 * // Invalid expression (server 400)
 * {
 *   "error": "failed to solve problem: Invalid token: '+' at position 3"
 * }
 */
export const solve = async (problem) => {
  if (!isAuthenticated()) {
    throw new Error('Not authenticated');
  }

  const response = await fetch(`${API_BASE_URL}/solve`, {
    method: 'POST',
    credentials: 'include',  // Send authToken cookie automatically
    headers: {
      'Content-Type': 'application/json',
      // No Authorization header needed - token sent via httpOnly cookie
    },
    body: JSON.stringify({ problem }),
  });

  if (!response.ok) {
    const errorMessage = await parseErrorResponse(response, 'Failed to solve problem');
    throw new Error(errorMessage);
  }

  return response.json();
};

/**
 * Retrieves the authenticated user's problem-solving history (authenticated endpoint).
 *
 * Sends GET request to /history endpoint to fetch all math problems the user has solved.
 * Returns array of history items ordered by created_at timestamp (most recent first based
 * on typical query pattern, though backend doesn't specify ORDER BY - potential issue).
 * History is populated asynchronously by history-worker processing NATS events, so recently
 * solved problems may not appear immediately (eventual consistency, typically <100ms delay).
 *
 * **Requires Authentication**: Must have valid JWT token in localStorage. Server validates
 * token and extracts user_id from JWT claims to fetch only that user's history.
 *
 * **History Retrieval Flow**:
 * 1. Client calls getHistory() with token from localStorage
 * 2. Server validates JWT token (AuthMiddleware)
 * 3. Server extracts user_id from token claims
 * 4. Server queries database: SELECT * FROM history WHERE user_id = $1
 * 5. Server returns history items for user
 * 6. Client receives array of history objects
 *
 * **Response Structure**:
 * - Each history item contains: id, user_id, problem_text, answer_text, created_at
 * - created_at is ISO 8601 timestamp string (e.g., "2024-01-15T10:30:00Z")
 * - Empty array [] returned if user has no history
 *
 * @async
 * @function
 * @returns {Promise<Array<Object>>} Array of history items for the authenticated user
 * @returns {number} return[].id - History item database ID
 * @returns {number} return[].user_id - User's database ID (always matches authenticated user)
 * @returns {string} return[].problem_text - The mathematical expression that was solved
 * @returns {string} return[].answer_text - The calculated answer as a string
 * @returns {string} return[].created_at - ISO 8601 timestamp when problem was solved
 *
 * @throws {Error} If not authenticated (no token in localStorage)
 * @throws {Error} If token is invalid or expired (401 Unauthorized from server)
 * @throws {Error} If network request fails (handles both JSON and non-JSON error responses)
 *
 * @example
 * // Fetch user's history
 * try {
 *   const history = await getHistory();
 *   console.log(`User has solved ${history.length} problems`);
 *   history.forEach(item => {
 *     console.log(`${item.problem_text} = ${item.answer_text}`);
 *   });
 * } catch (error) {
 *   console.error('Failed to fetch history:', error.message);
 * }
 *
 * @example
 * // Success response structure
 * [
 *   {
 *     "id": 1,
 *     "user_id": 123,
 *     "problem_text": "2+3",
 *     "answer_text": "5",
 *     "created_at": "2024-01-15T10:30:00Z"
 *   },
 *   {
 *     "id": 2,
 *     "user_id": 123,
 *     "problem_text": "10*5",
 *     "answer_text": "50",
 *     "created_at": "2024-01-15T10:31:00Z"
 *   }
 * ]
 *
 * @example
 * // Empty history (new user or no problems solved yet)
 * []
 *
 * @example
 * // Error responses
 * // Not authenticated (client-side check)
 * Error: 'Not authenticated'
 *
 * // Invalid/expired token (server 401)
 * {
 *   "error": "invalid or expired token"
 * }
 */
export const getHistory = async () => {
  if (!isAuthenticated()) {
    throw new Error('Not authenticated');
  }

  const response = await fetch(`${API_BASE_URL}/history`, {
    method: 'GET',
    credentials: 'include',  // Send authToken cookie automatically
    headers: {
      // No Authorization header needed - token sent via httpOnly cookie
    },
  });

  if (!response.ok) {
    const errorMessage = await parseErrorResponse(response, 'Failed to fetch history');
    throw new Error(errorMessage);
  }

  return response.json();
};
