/**
 * Utility functions for client-side validation and data formatting.
 *
 * This module provides pure functions with no side effects for validating user input
 * before sending to the API. Client-side validation improves UX by providing immediate
 * feedback, but should NOT be relied upon for security (server-side validation required).
 *
 * **Design Principles**:
 * - All functions are pure (no side effects, same input → same output)
 * - Return null for valid input, error message string for invalid
 * - Consistent error message formatting for UI display
 * - Basic validation only - comprehensive validation done server-side
 *
 * @module utils
 */

/**
 * Validates email address format using regex pattern.
 *
 * Performs basic email validation checking for presence and simple format (text@text.text).
 * Uses regex pattern /^[^\s@]+@[^\s@]+\.[^\s@]+$/ which matches most common email formats.
 * Note: This is NOT comprehensive email validation (RFC 5322 is complex), but sufficient
 * for basic UX validation. Server-side should perform additional validation.
 *
 * **Validation Rules**:
 * - Must not be empty
 * - Must contain @ symbol
 * - Must contain text before and after @
 * - Must contain . after @
 * - Cannot contain whitespace
 *
 * **Limitations**:
 * - Regex is permissive: accepts technically invalid formats like "a@b.c" or "test@-.com"
 * - Doesn't validate TLD (top-level domain) - allows "user@example.invalidtld"
 *
 * **Not Validated** (acceptable for UX, server should validate):
 * - Unicode characters in email
 * - Special characters in local part (before @)
 * - Multiple @ symbols (regex prevents this)
 * - Exact RFC 5322 compliance
 *
 * @function
 * @param {string} email - Email address to validate
 * @returns {string|null} Error message if invalid, null if valid
 * @returns {string} 'Email is required' if empty
 * @returns {string} 'Invalid email format' if format doesn't match pattern
 * @returns {null} If validation passes
 *
 * @example
 * // Valid emails
 * validateEmail('user@example.com');     // → null (valid)
 * validateEmail('test.user@domain.co');  // → null (valid)
 * validateEmail('a@b.c');                // → null (valid, though unusual)
 *
 * @example
 * // Invalid emails
 * validateEmail('');                // → 'Email is required'
 * validateEmail('notanemail');      // → 'Invalid email format'
 * validateEmail('missing@domain');  // → 'Invalid email format' (no TLD)
 * validateEmail('has space@test.com'); // → 'Invalid email format' (whitespace)
 * validateEmail('  user@test.com'); // → 'Invalid email format' (leading space)
 *
 * @example
 * // Usage in component
 * const handleSubmit = () => {
 *   const error = validateEmail(email);
 *   if (error) {
 *     setErrorMessage(error);
 *     return;
 *   }
 *   // Proceed with API call
 *   await register(email, password);
 * };
 */
export const validateEmail = (email) => {
  if (!email) {
    return 'Email is required';
  }
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!emailRegex.test(email)) {
    return 'Invalid email format';
  }
  return null;
};

/**
 * Validates password meets minimum security requirements.
 *
 * Performs basic password validation checking for presence and minimum length.
 * Enforces a 6-character minimum requirement.
 *
 * **Validation Rules**:
 * - Must not be empty
 * - Must be at least 6 characters long
 *
 * @function
 * @param {string} password - Password to validate
 * @returns {string|null} Error message if invalid, null if valid
 * @returns {string} 'Password is required' if empty
 * @returns {string} 'Password must be at least 6 characters' if too short
 * @returns {null} If validation passes
 *
 * @example
 * // Valid passwords (by function criteria, NOT secure)
 * validatePassword('123456');      // → null (valid but terrible password)
 * validatePassword('password');    // → null (valid but commonly breached)
 * validatePassword('abcdef');      // → null (valid but very weak)
 *
 * @example
 * // Invalid passwords
 * validatePassword('');        // → 'Password is required'
 * validatePassword('12345');   // → 'Password must be at least 6 characters'
 * validatePassword('abc');     // → 'Password must be at least 6 characters'
 *
 * @example
 * // Usage in component
 * const handleRegister = () => {
 *   const emailError = validateEmail(email);
 *   const passwordError = validatePassword(password);
 *
 *   if (emailError || passwordError) {
 *     setErrors({ email: emailError, password: passwordError });
 *     return;
 *   }
 *   // Proceed with registration
 *   await register(email, password);
 * };
 */
export const validatePassword = (password) => {
  if (!password) {
    return 'Password is required';
  }
  if (password.length < 6) {
    return 'Password must be at least 6 characters';
  }
  return null;
};

/**
 * Validates mathematical expression contains only allowed characters.
 *
 * Performs basic validation of math problem input checking for presence and character whitelist.
 * Allows digits, basic operators (+, -, *, /), parentheses, decimal points, and whitespace.
 * This prevents users from submitting expressions with letters or special characters that would
 * fail on the backend (govaluate library).
 *
 * **Validation Rules**:
 * - Must not be empty or only whitespace
 * - Must contain only: digits (0-9), operators (+, -, *, /), parentheses ( ), decimal point (.), whitespace
 * - Uses regex pattern: /^[0-9+\-*\\/().\s]+$/
 *
 * **Allowed Examples**:
 * - Simple: "2+3", "10-5", "4*6", "10/2"
 * - Complex: "(2+3)*4", "((1+2)*3)/4"
 * - Decimals: "2.5+3.5", "10.0/2.5"
 * - Whitespace: "2 + 3", "10 * (4 + 2)"
 *
 * @function
 * @param {string} problem - Mathematical expression to validate
 * @returns {string|null} Error message if invalid, null if valid
 * @returns {string} 'Please enter a math problem' if empty or only whitespace
 * @returns {string} 'Problem contains invalid characters' if non-whitelisted characters present
 * @returns {null} If validation passes
 *
 * @example
 * // Valid problems (whitelisted characters)
 * validateProblem('2+3');          // → null
 * validateProblem('(10-5)*2');     // → null
 * validateProblem('2.5 + 3.5');    // → null
 * validateProblem('  10/2  ');     // → null (whitespace allowed)
 *
 * @example
 * // Invalid problems (non-whitelisted characters)
 * validateProblem('');             // → 'Please enter a math problem'
 * validateProblem('   ');          // → 'Please enter a math problem'
 * validateProblem('2+x');          // → 'Problem contains invalid characters'
 * validateProblem('sqrt(4)');      // → 'Problem contains invalid characters'
 * validateProblem('2^3');          // → 'Problem contains invalid characters' (^ not supported)
 *
 * @example
 * // Usage in component
 * const handleSolve = async () => {
 *   const error = validateProblem(problem);
 *   if (error) {
 *     setErrorMessage(error);
 *     return;
 *   }
 *   // Proceed with API call
 *   const result = await solve(problem);
 *   setAnswer(result.answer);
 * };
 */
export const validateProblem = (problem) => {
  if (!problem || problem.trim() === '') {
    return 'Please enter a math problem';
  }
  const validChars = /^[0-9+\-*/().\s]+$/;
  if (!validChars.test(problem)) {
    return 'Problem contains invalid characters';
  }
  return null;
};

/**
 * Formats ISO 8601 timestamp string to localized human-readable format.
 *
 * Converts backend timestamp (ISO 8601 format like "2024-01-15T10:30:00Z") to locale-specific
 * display format using browser's toLocaleString() method. Output format depends on user's
 * browser locale settings (e.g., "1/15/2024, 10:30:00 AM" for en-US).
 *
 * **Input Format** (from backend):
 * - ISO 8601: "2024-01-15T10:30:00Z"
 * - UTC timezone (Z suffix)
 * - From database TIMESTAMP DEFAULT CURRENT_TIMESTAMP
 *
 * **Output Format** (browser locale-dependent):
 * - en-US: "1/15/2024, 10:30:00 AM"
 * - en-GB: "15/01/2024, 10:30:00"
 * - de-DE: "15.1.2024, 10:30:00"
 * - Converts from UTC to user's local timezone automatically
 *
 * **Use Cases**:
 * - Displaying history item timestamps in HistoryPage
 * - Showing when problems were solved in user's local time
 *
 * **Edge Cases Handled**:
 * - Invalid date strings: Returns "Invalid Date" (Date constructor behavior)
 * - Null/undefined: Returns "Invalid Date"
 * - Non-string types: Coerced to string by Date constructor
 *
 * **Customization Options** (not used, but available):
 * - Could pass options object to toLocaleString for custom formatting:
 *   ```javascript
 *   date.toLocaleString('en-US', {
 *     dateStyle: 'medium',
 *     timeStyle: 'short'
 *   });
 *   // → "Jan 15, 2024, 10:30 AM"
 *   ```
 *
 * @function
 * @param {string} dateString - ISO 8601 timestamp string from backend (e.g., "2024-01-15T10:30:00Z")
 * @returns {string} Localized date/time string in user's browser locale
 *
 * @example
 * // Typical usage with backend timestamp
 * const timestamp = "2024-01-15T10:30:00Z";
 * formatDate(timestamp);  // → "1/15/2024, 10:30:00 AM" (en-US locale)
 *
 * @example
 * // Different locales produce different formats
 * const timestamp = "2024-01-15T14:30:00Z";
 * // Browser set to en-US: "1/15/2024, 2:30:00 PM"
 * // Browser set to en-GB: "15/01/2024, 14:30:00"
 * // Browser set to de-DE: "15.1.2024, 14:30:00"
 *
 * @example
 * // Timezone conversion (UTC to local)
 * const utcTimestamp = "2024-01-15T10:00:00Z";  // 10:00 AM UTC
 * formatDate(utcTimestamp);
 * // User in PST (UTC-8): "1/15/2024, 2:00:00 AM"
 * // User in CET (UTC+1): "1/15/2024, 11:00:00 AM"
 *
 * @example
 * // Edge cases
 * formatDate('');                    // → "Invalid Date"
 * formatDate('not-a-date');          // → "Invalid Date"
 * formatDate(null);                  // → "Invalid Date"
 * formatDate('2024-13-01T00:00:00Z'); // → "Invalid Date" (month 13 doesn't exist)
 *
 * @example
 * // Usage in component (displaying history)
 * const HistoryItem = ({ item }) => (
 *   <div>
 *     <p>{item.problem_text} = {item.answer_text}</p>
 *     <small>{formatDate(item.created_at)}</small>
 *   </div>
 * );
 */
export const formatDate = (dateString) => {
  const date = new Date(dateString);
  return date.toLocaleString();
};
