/**
 * Login page component for existing user authentication.
 *
 * Provides a login form for users to authenticate with email and password credentials.
 * Uses LoginInputComponent for consistent pixel art retro styling. Performs client-side
 * validation before submitting credentials to the backend API. On successful login,
 * invokes parent callback to update application authentication state.
 *
 * **Component Type**: Smart container component (manages state and business logic)
 * **Presentation Component**: LoginInputComponent (dumb component, no business logic)
 *
 * **Authentication Flow**:
 * 1. User enters email and password in form inputs
 * 2. On submit: Client-side validation (validateEmail, check password not empty)
 * 3. If validation passes: Call api.login() to authenticate with backend
 * 4. Backend verifies credentials with bcrypt comparison (constant-time)
 * 5. Backend returns JWT token (valid 24 hours) and user data
 * 6. api.login() saves token to localStorage
 * 7. LoginPage calls onLoginSuccess() callback
 * 8. App.jsx updates authenticated=true and navigates to solver page
 *
 * **Validation**:
 * - Email: validateEmail() - checks format, not empty
 * - Password: Only checks not empty
 * - Server-side: Backend also validates and returns generic "invalid credentials" for security
 *
 * **Error Handling**:
 * - Validation errors: Displayed immediately without API call
 * - API errors: Displayed from err.message (e.g., "invalid credentials", network errors)
 *
 * @component
 * @param {Object} props - Component props
 * @param {Function} props.onLoginSuccess - Callback invoked after successful login (no arguments)
 * @returns {JSX.Element} Login form within page container
 *
 * @example
 * // Usage in App.jsx
 * <LoginPage onLoginSuccess={handleLoginSuccess} />
 *
 * @example
 * // handleLoginSuccess callback (in App.jsx)
 * const handleLoginSuccess = () => {
 *   setAuthenticated(true);
 *   setCurrentPage('solve');
 * };
 */
import React, { useState } from 'react';
import { login } from '../api';
import { validateEmail } from '../utils';
import LoginInputComponent from './LoginInputComponent';

/**
 * LoginPage functional component with form state management.
 *
 * Manages three pieces of local state: email, password, and error message.
 * Delegates presentation to LoginInputComponent while handling business logic
 * (validation, API calls, error handling).
 *
 * **Props**:
 * @param {Object} props
 * @param {Function} props.onLoginSuccess - Success callback from parent (App.jsx)
 *
 * @function
 */
const LoginPage = ({ onLoginSuccess }) => {
  /**
   * Email input value (controlled component).
   *
   * Bound to email input in LoginInputComponent. Updated via setEmail on every keystroke.
   * Passed directly to api.login().
   *
   * @type {string}
   * @default ''
   */
  const [email, setEmail] = useState('');

  /**
   * Password input value (controlled component).
   *
   * Bound to password input in LoginInputComponent. Updated via setPassword on every keystroke.
   * Masked in UI (input type="password"). Passed directly to api.login().
   *
   * **Security Note**: Stored in React state (plain text in memory) until form submission.
   * Not exposed via localStorage or other persistent storage. Cleared on component unmount.
   *
   * @type {string}
   * @default ''
   */
  const [password, setPassword] = useState('');

  /**
   * Error message state for validation and API errors.
   *
   * Displayed in red text below form inputs when non-empty. Cleared on each submit attempt.
   *
   * **Possible Values**:
   * - '' (empty): No error, form is valid
   * - 'Email is required': validateEmail returned error (empty email)
   * - 'Invalid email format': validateEmail returned error (bad format)
   * - 'Password is required': Password field empty (line 25)
   * - 'invalid credentials': Backend returned error (wrong email or password)
   * - Other API error messages: Network errors, server errors
   *
   * @type {string}
   * @default ''
   */
  const [error, setError] = useState('');

  /**
   * Form submit handler with validation and authentication.
   *
   * Triggered when user clicks LOGIN button or presses Enter in form. Prevents default
   * form submission (page reload), validates inputs client-side, calls login API, and
   * handles success/failure.
   *
   * **Flow**:
   * 1. Prevent default form submission (e.preventDefault())
   * 2. Clear any previous error message
   * 3. Validate email with validateEmail() - checks format and not empty
   * 4. Validate password - only checks not empty
   * 5. If validation fails: Set error message, return early (no API call)
   * 6. If validation passes: Call await login(email, password)
   * 7. login() saves token to localStorage and returns user data
   * 8. Call onLoginSuccess() to notify parent (App.jsx)
   * 9. App.jsx updates auth state and navigates to solver page
   * 10. If API fails: Catch error, set err.message to error state
   *
   * **Error Messages**:
   * - Client-side validation: Immediate feedback without network call
   * - Server errors: Generic "invalid credentials" for both wrong email and wrong password
   *   (good security - prevents user enumeration)
   *
   * @async
   * @function
   * @param {Event} e - Form submit event
   * @returns {Promise<void>}
   *
   * @example
   * // Successful login flow
   * // 1. User enters: email="user@example.com", password="password123"
   * // 2. Validation passes
   * // 3. API returns: { token: "eyJ...", email: "user@example.com", user_id: 123 }
   * // 4. Token saved to localStorage
   * // 5. onLoginSuccess() called
   * // 6. User navigated to solver page
   *
   * @example
   * // Validation error (bad email format)
   * // 1. User enters: email="notanemail", password="password123"
   * // 2. validateEmail returns "Invalid email format"
   * // 3. Error displayed, no API call made
   *
   * @example
   * // Authentication error (wrong credentials)
   * // 1. User enters: email="user@example.com", password="wrongpassword"
   * // 2. Validation passes
   * // 3. API returns 401 with { error: "invalid credentials" }
   * // 4. Error displayed: "invalid credentials"
   */
  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');

    const emailError = validateEmail(email);
    if (emailError) {
      setError(emailError);
      return;
    }

    if (!password) {
      setError('Password is required');
      return;
    }

    try {
      await login(email, password);
      onLoginSuccess();
    } catch (err) {
      setError(err.message);
    }
  };

  /**
   * JSX Return: Renders page container with LoginInputComponent.
   *
   * **Structure**:
   * - Outer div with .page-container class (centers content, defined in global.css)
   * - LoginInputComponent (reusable presentational component)
   *
   * **Props Passed to LoginInputComponent**:
   * - email: Current email state value (controlled input)
   * - password: Current password state value (controlled input)
   * - onEmailChange: setEmail state setter (updates on each keystroke)
   * - onPasswordChange: setPassword state setter (updates on each keystroke)
   * - onSubmit: handleSubmit function (validates and calls API)
   * - buttonText: "LOGIN" (displayed on submit button)
   * - error: Current error message or empty string (displayed in red if non-empty)
   *
   * **Separation of Concerns**:
   * - LoginPage (this component): Smart container with business logic
   * - LoginInputComponent: Dumb presentation component with no business logic
   * - This pattern enables reusability (RegisterPage uses same component with different buttonText)
   *
   * @returns {JSX.Element} Login form page
   */
  return (
    <div className="page-container">
      <LoginInputComponent
        email={email}
        password={password}
        onEmailChange={setEmail}
        onPasswordChange={setPassword}
        onSubmit={handleSubmit}
        buttonText="LOGIN"
        error={error}
      />
    </div>
  );
};

export default LoginPage;
