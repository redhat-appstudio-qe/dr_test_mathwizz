/**
 * Registration page component for new user account creation.
 *
 * Provides a registration form for users to create accounts with email and password credentials.
 * Uses LoginInputComponent for consistent pixel art retro styling (shared with LoginPage).
 * Performs comprehensive client-side validation before submitting to backend API. On successful
 * registration, backend returns JWT token and user is immediately authenticated (no separate login).
 *
 * **Component Type**: Smart container component (manages state and business logic)
 * **Presentation Component**: LoginInputComponent (dumb component, reused from LoginPage)
 *
 * **Registration Flow**:
 * 1. User enters email and password in form inputs
 * 2. On submit: Client-side validation (validateEmail + validatePassword)
 * 3. If validation passes: Call api.register() to create account on backend
 * 4. Backend creates user in database with bcrypt-hashed password
 * 5. Backend generates JWT token (valid 24 hours) and returns with user data
 * 6. api.register() saves token to localStorage
 * 7. RegisterPage calls onRegisterSuccess() callback
 * 8. App.jsx updates authenticated=true and navigates to solver page
 * 9. User immediately logged in, no separate login step required
 *
 * **Validation**:
 * - Email: validateEmail() - checks format, not empty
 * - Password: validatePassword() - checks >=6 chars
 * - Server-side: Backend also validates (checks non-empty, >=6 chars, creates bcrypt hash)
 *
 * **Error Handling**:
 * - Validation errors: Displayed immediately without API call (better UX)
 * - API errors: Displayed from err.message (e.g., "failed to create user")
 *
 * **Comparison with LoginPage**:
 * - **Same**: Component structure, state management, JSX return, error handling
 * - **Different**: RegisterPage calls validatePassword, LoginPage doesn't
 * - **Different**: buttonText="CREATE ACCOUNT" vs "LOGIN"
 * - **Different**: API call (register vs login)
 *
 * @component
 * @param {Object} props - Component props
 * @param {Function} props.onRegisterSuccess - Callback invoked after successful registration (no arguments)
 * @returns {JSX.Element} Registration form within page container
 *
 * @example
 * // Usage in App.jsx
 * <RegisterPage onRegisterSuccess={handleRegisterSuccess} />
 *
 * @example
 * // handleRegisterSuccess callback (in App.jsx)
 * const handleRegisterSuccess = () => {
 *   setAuthenticated(true);
 *   setCurrentPage('solve');
 * };
 */
import React, { useState } from 'react';
import { register } from '../api';
import { validateEmail, validatePassword } from '../utils';
import LoginInputComponent from './LoginInputComponent';

/**
 * RegisterPage functional component with form state management.
 *
 * Manages three pieces of local state: email, password, and error message.
 * Delegates presentation to LoginInputComponent (shared with LoginPage) while
 * handling registration-specific business logic (full password validation, register API call).
 *
 * **Props**:
 * @param {Object} props
 * @param {Function} props.onRegisterSuccess - Success callback from parent (App.jsx)
 *
 * @function
 */
const RegisterPage = ({ onRegisterSuccess }) => {
  /**
   * Email input value (controlled component).
   *
   * Bound to email input in LoginInputComponent. Updated via setEmail on every keystroke.
   * Validated with validateEmail() before API call. Passed directly to api.register().
   *
   * **Validation**: Must match email regex pattern, not empty
   *
   * @type {string}
   * @default ''
   */
  const [email, setEmail] = useState('');

  /**
   * Password input value (controlled component).
   *
   * Bound to password input in LoginInputComponent. Updated via setPassword on every keystroke.
   * Masked in UI (input type="password"). Validated with validatePassword() before API call.
   * Passed directly to api.register().
   *
   * **Security Note**: Stored in React state (plain text in memory) until form submission.
   * Not exposed via localStorage or other persistent storage. Cleared on component unmount.
   * Server hashes with bcrypt before database storage.
   *
   * **Validation**: Must be >=6 characters
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
   * - 'Password is required': validatePassword returned error (empty password)
   * - 'Password must be at least 6 characters': validatePassword returned error (too short)
   * - 'failed to create user': Backend returned error
   * - Other API error messages: Network errors, server errors
   *
   * @type {string}
   * @default ''
   */
  const [error, setError] = useState('');

  /**
   * Form submit handler with comprehensive validation and account creation.
   *
   * Triggered when user clicks CREATE ACCOUNT button or presses Enter in form. Prevents default
   * form submission (page reload), validates both email AND password client-side, calls register
   * API, and handles success/failure.
   *
   * **Flow**:
   * 1. Prevent default form submission (e.preventDefault())
   * 2. Clear any previous error message
   * 3. Validate email with validateEmail() - checks format and not empty
   * 4. Validate password with validatePassword() - checks >=6 chars
   * 5. If either validation fails: Set error message, return early (no API call)
   * 6. If validation passes: Call await register(email, password)
   * 7. register() creates account, saves token to localStorage, returns user data
   * 8. Call onRegisterSuccess() to notify parent (App.jsx)
   * 9. App.jsx updates auth state and navigates to solver page
   * 10. User immediately authenticated, no separate login required
   * 11. If API fails: Catch error, set err.message to error state
   *
   * **Error Messages**:
   * - Client-side validation: Immediate feedback without network call (good UX)
   * - Server errors: Generic "failed to create user" message
   *
   * @async
   * @function
   * @param {Event} e - Form submit event
   * @returns {Promise<void>}
   *
   * @example
   * // Successful registration flow
   * // 1. User enters: email="newuser@example.com", password="password123"
   * // 2. Validation passes (email format valid, password >=6 chars)
   * // 3. API returns: { token: "eyJ...", email: "newuser@example.com", user_id: 456 }
   * // 4. Token saved to localStorage
   * // 5. onRegisterSuccess() called
   * // 6. User immediately authenticated and navigated to solver page
   *
   * @example
   * // Validation error (password too short)
   * // 1. User enters: email="user@example.com", password="12345" (5 chars)
   * // 2. validatePassword returns "Password must be at least 6 characters"
   * // 3. Error displayed, no API call made
   *
   * @example
   * // Registration error (duplicate email)
   * // 1. User enters: email="existing@example.com", password="password123"
   * // 2. Validation passes
   * // 3. API returns 400 with { error: "failed to create user" }
   * // 4. Error displayed: "failed to create user"
   */
  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');

    const emailError = validateEmail(email);
    if (emailError) {
      setError(emailError);
      return;
    }

    const passwordError = validatePassword(password);
    if (passwordError) {
      setError(passwordError);
      return;
    }

    try {
      await register(email, password);
      onRegisterSuccess();
    } catch (err) {
      setError(err.message);
    }
  };

  /**
   * JSX Return: Renders page container with LoginInputComponent.
   *
   * **Structure**:
   * - Outer div with .page-container class (centers content, defined in global.css)
   * - LoginInputComponent (reusable presentational component, shared with LoginPage)
   *
   * **Props Passed to LoginInputComponent**:
   * - email: Current email state value (controlled input)
   * - password: Current password state value (controlled input)
   * - onEmailChange: setEmail state setter (updates on each keystroke)
   * - onPasswordChange: setPassword state setter (updates on each keystroke)
   * - onSubmit: handleSubmit function (validates email + password, calls register API)
   * - buttonText: "CREATE ACCOUNT" (displayed on submit button, different from LoginPage's "LOGIN")
   * - error: Current error message or empty string (displayed in red if non-empty)
   *
   * **Separation of Concerns**:
   * - RegisterPage (this component): Smart container with registration business logic
   * - LoginInputComponent: Dumb presentation component with no business logic
   * - Same presentation component reused by both LoginPage and RegisterPage (good design)
   *
   * @returns {JSX.Element} Registration form page
   */
  return (
    <div className="page-container">
      <LoginInputComponent
        email={email}
        password={password}
        onEmailChange={setEmail}
        onPasswordChange={setPassword}
        onSubmit={handleSubmit}
        buttonText="CREATE ACCOUNT"
        error={error}
      />
    </div>
  );
};

export default RegisterPage;
