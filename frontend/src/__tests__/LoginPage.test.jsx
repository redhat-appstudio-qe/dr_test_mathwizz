// Comprehensive unit tests for LoginPage.jsx
// Tests authentication form submission, validation, error handling, and callbacks
// Exposes Bug #42 (no loading state) and Bug #43 (no input trimming)

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import LoginPage from '../components/LoginPage';
import { login } from '../api';
import { validateEmail } from '../utils';

// Mock the API module
jest.mock('../api', () => ({
  login: jest.fn(),
}));

// Mock the utils module
jest.mock('../utils', () => ({
  validateEmail: jest.fn(),
}));

// Mock LoginInputComponent to simplify testing
jest.mock('../components/LoginInputComponent', () => {
  return function MockLoginInputComponent({
    email,
    password,
    onEmailChange,
    onPasswordChange,
    onSubmit,
    buttonText,
    error
  }) {
    return (
      <div data-testid="login-input-component">
        <form onSubmit={onSubmit}>
          <input
            data-testid="email-input"
            type="email"
            value={email}
            onChange={(e) => onEmailChange(e.target.value)}
            placeholder="Email"
          />
          <input
            data-testid="password-input"
            type="password"
            value={password}
            onChange={(e) => onPasswordChange(e.target.value)}
            placeholder="Password"
          />
          {error && <div data-testid="error-message">{error}</div>}
          <button data-testid="submit-button" type="submit">
            {buttonText}
          </button>
        </form>
      </div>
    );
  };
});

describe('LoginPage Component', () => {
  let mockOnLoginSuccess;

  beforeEach(() => {
    // Reset all mocks before each test
    jest.clearAllMocks();

    // Create fresh mock function for each test
    mockOnLoginSuccess = jest.fn();

    // Default: validateEmail returns null (valid)
    validateEmail.mockReturnValue(null);
  });

  describe('when testing component rendering', () => {
    test('should render LoginInputComponent with correct initial state', () => {
      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      // Verify LoginInputComponent is rendered
      expect(screen.getByTestId('login-input-component')).toBeInTheDocument();

      // Verify button text is "LOGIN"
      expect(screen.getByTestId('submit-button')).toHaveTextContent('LOGIN');

      // Verify inputs are empty initially
      expect(screen.getByTestId('email-input')).toHaveValue('');
      expect(screen.getByTestId('password-input')).toHaveValue('');

      // Verify no error message initially
      expect(screen.queryByTestId('error-message')).not.toBeInTheDocument();
    });

    test('should render page container with correct className', () => {
      const { container } = render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      // Verify page-container div exists
      const pageContainer = container.querySelector('.page-container');
      expect(pageContainer).toBeInTheDocument();
    });
  });

  describe('when testing successful login flow', () => {
    test('should call login API and onLoginSuccess callback with valid credentials', async () => {
      // Mock successful API response
      login.mockResolvedValueOnce({ email: 'test@example.com', user_id: 1 });

      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      // Fill in email and password
      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      const submitButton = screen.getByTestId('submit-button');

      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      // Submit form
      fireEvent.click(submitButton);

      // Wait for async operations
      await waitFor(() => {
        // Verify validateEmail was called with email
        expect(validateEmail).toHaveBeenCalledWith('test@example.com');

        // Verify login API was called with email and password
        expect(login).toHaveBeenCalledWith('test@example.com', 'password123');

        // Verify onLoginSuccess callback was called
        expect(mockOnLoginSuccess).toHaveBeenCalledTimes(1);
      });

      // Verify no error message displayed
      expect(screen.queryByTestId('error-message')).not.toBeInTheDocument();
    });

    test('should handle Enter key submission in form', async () => {
      login.mockResolvedValueOnce({ email: 'test@example.com', user_id: 1 });

      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');

      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      // Submit form via Enter key (submit event on form)
      const form = screen.getByTestId('login-input-component').querySelector('form');
      fireEvent.submit(form);

      await waitFor(() => {
        expect(login).toHaveBeenCalledWith('test@example.com', 'password123');
        expect(mockOnLoginSuccess).toHaveBeenCalledTimes(1);
      });
    });

    test('should clear error message from previous attempt on successful login', async () => {
      // First attempt: validation error
      validateEmail.mockReturnValueOnce('Invalid email format');

      const { rerender } = render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // Verify error is shown
      expect(screen.getByTestId('error-message')).toHaveTextContent('Invalid email format');

      // Second attempt: successful login
      validateEmail.mockReturnValue(null);
      login.mockResolvedValueOnce({ email: 'test@example.com', user_id: 1 });

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');

      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });
      fireEvent.click(submitButton);

      // Error should be cleared before API call
      await waitFor(() => {
        expect(mockOnLoginSuccess).toHaveBeenCalled();
      });

      // Verify error is cleared after successful login
      expect(screen.queryByTestId('error-message')).not.toBeInTheDocument();
    });
  });

  describe('when testing validation errors', () => {
    test('should display error when email is invalid (empty)', () => {
      validateEmail.mockReturnValue('Email is required');

      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // Verify error message is displayed
      expect(screen.getByTestId('error-message')).toHaveTextContent('Email is required');

      // Verify login API was NOT called
      expect(login).not.toHaveBeenCalled();

      // Verify onLoginSuccess was NOT called
      expect(mockOnLoginSuccess).not.toHaveBeenCalled();
    });

    test('should display error when email format is invalid', () => {
      validateEmail.mockReturnValue('Invalid email format');

      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      fireEvent.change(emailInput, { target: { value: 'notanemail' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // Verify error message is displayed
      expect(screen.getByTestId('error-message')).toHaveTextContent('Invalid email format');

      // Verify login API was NOT called
      expect(login).not.toHaveBeenCalled();

      // Verify onLoginSuccess was NOT called
      expect(mockOnLoginSuccess).not.toHaveBeenCalled();
    });

    test('should display error when password is empty', () => {
      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // Verify error message is displayed
      expect(screen.getByTestId('error-message')).toHaveTextContent('Password is required');

      // Verify login API was NOT called (password validation happens before API call)
      expect(login).not.toHaveBeenCalled();

      // Verify onLoginSuccess was NOT called
      expect(mockOnLoginSuccess).not.toHaveBeenCalled();
    });

    test('should display error when both email and password are empty', () => {
      validateEmail.mockReturnValue('Email is required');

      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // Verify email validation error is displayed first (short-circuit)
      expect(screen.getByTestId('error-message')).toHaveTextContent('Email is required');

      // Verify login API was NOT called
      expect(login).not.toHaveBeenCalled();
    });

    test('should clear previous error when submitting again', () => {
      validateEmail
        .mockReturnValueOnce('Invalid email format')
        .mockReturnValueOnce('Email is required');

      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      // First submit: invalid email format
      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);
      expect(screen.getByTestId('error-message')).toHaveTextContent('Invalid email format');

      // Second submit: email is required
      fireEvent.click(submitButton);
      expect(screen.getByTestId('error-message')).toHaveTextContent('Email is required');

      // Verify error was updated (not concatenated)
      expect(screen.queryByText('Invalid email format')).not.toBeInTheDocument();
    });
  });

  describe('when testing API error handling', () => {
    test('should display error message when login API fails', async () => {
      // Mock API rejection
      login.mockRejectedValueOnce(new Error('invalid credentials'));

      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'wrongpassword' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // Wait for async error handling
      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toHaveTextContent('invalid credentials');
      });

      // Verify onLoginSuccess was NOT called on failure
      expect(mockOnLoginSuccess).not.toHaveBeenCalled();
    });

    test('should display network error message', async () => {
      login.mockRejectedValueOnce(new Error('Network error'));

      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toHaveTextContent('Network error');
      });
    });

    test('should not clear input fields when API fails', async () => {
      login.mockRejectedValueOnce(new Error('invalid credentials'));

      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'wrongpassword' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      // Verify inputs still have values (user can retry without re-entering)
      expect(emailInput).toHaveValue('test@example.com');
      expect(passwordInput).toHaveValue('wrongpassword');
    });
  });

  describe('when testing input trimming (Bug #43)', () => {
    test('should handle email with whitespace (HTML5 email input auto-trims)', async () => {
      /**
       * NOTE: HTML5 email input type automatically trims leading/trailing whitespace.
       *
       * CURRENT BEHAVIOR: Email input with type="email" auto-trims spaces before onChange fires.
       * So "  test@example.com  " becomes "test@example.com" in the input value.
       *
       * This is actually GOOD behavior from the browser. Bug #43 is more about
       * password inputs (which don't auto-trim) and explicit validation before API calls.
       */
      login.mockResolvedValueOnce({ email: 'test@example.com', user_id: 1 });

      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');

      // Email with leading/trailing spaces (browser will auto-trim)
      fireEvent.change(emailInput, { target: { value: '  test@example.com  ' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      await waitFor(() => {
        // HTML5 email input auto-trims, so login receives trimmed email
        expect(login).toHaveBeenCalledWith('test@example.com', 'password123');
      });
    });

    test('should NOT trim leading/trailing whitespace from password - exposes Bug #43', async () => {
      /**
       * BUG #43: LoginPage does NOT trim whitespace from password input.
       *
       * CURRENT BEHAVIOR: Password "  mypassword  " is sent to API with spaces.
       * EXPECTED BEHAVIOR: Password should be trimmed before API call.
       *
       * Note: Some argue passwords should preserve whitespace (intentional spaces),
       * but leading/trailing whitespace is usually unintentional (copy-paste errors).
       */
      login.mockResolvedValueOnce({ email: 'test@example.com', user_id: 1 });

      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');

      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
      // Password with leading/trailing spaces
      fireEvent.change(passwordInput, { target: { value: '  password123  ' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      await waitFor(() => {
        // BUG #43: Password is NOT trimmed
        expect(login).toHaveBeenCalledWith('test@example.com', '  password123  ');
      });
    });

    test('should preserve internal whitespace in email (not trim middle spaces)', async () => {
      login.mockResolvedValueOnce({ email: 'test user@example.com', user_id: 1 });

      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');

      // Email with space in middle (invalid email, but tests trimming behavior)
      fireEvent.change(emailInput, { target: { value: 'test user@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      await waitFor(() => {
        // Internal spaces preserved (not trimmed)
        expect(login).toHaveBeenCalledWith('test user@example.com', 'password123');
      });
    });
  });

  describe('when testing loading state (Bug #42)', () => {
    test('should NOT show loading state during API call - exposes Bug #42', async () => {
      /**
       * BUG #42: LoginPage has NO loading state during authentication.
       *
       * CURRENT BEHAVIOR: No visual feedback during API call (button stays enabled, no loading text).
       * EXPECTED BEHAVIOR: Should show loading indicator and disable button during API call.
       *
       * IMPACT:
       * - Users can double-click submit button, causing duplicate API calls
       * - No visual feedback that login is in progress
       * - Poor UX, especially on slow networks
       *
       * This test verifies current behavior (no loading state). When Bug #42 is fixed,
       * this test should be updated to verify loading state is shown.
       */
      // Use Promise to control timing of API resolution
      let resolveLogin;
      const loginPromise = new Promise((resolve) => {
        resolveLogin = resolve;
      });
      login.mockReturnValue(loginPromise);

      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // BUG #42: No loading indicator shown (this assertion should fail when bug is fixed)
      // When fixed, should see loading text or disabled button
      expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
      expect(submitButton).not.toBeDisabled(); // Button remains enabled during API call

      // Resolve the promise
      resolveLogin({ email: 'test@example.com', user_id: 1 });

      await waitFor(() => {
        expect(mockOnLoginSuccess).toHaveBeenCalled();
      });
    });

    test('should allow multiple rapid clicks (no debouncing) - exposes Bug #42', async () => {
      /**
       * BUG #42: Without loading state/disabled button, users can click submit multiple times.
       *
       * This causes multiple API calls for same login attempt, wasting backend resources.
       */
      login.mockResolvedValue({ email: 'test@example.com', user_id: 1 });

      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      const submitButton = screen.getByTestId('submit-button');

      // Rapid clicks (should be prevented with loading state/disabled button)
      fireEvent.click(submitButton);
      fireEvent.click(submitButton);
      fireEvent.click(submitButton);

      await waitFor(() => {
        // BUG #42: Login API called multiple times (no debouncing)
        // When fixed, login should only be called once
        expect(login).toHaveBeenCalledTimes(3);
      });
    });
  });

  describe('when testing state updates and re-renders', () => {
    test('should update email state on input change', () => {
      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const emailInput = screen.getByTestId('email-input');

      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });

      // Verify input value updated (controlled component)
      expect(emailInput).toHaveValue('test@example.com');
    });

    test('should update password state on input change', () => {
      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const passwordInput = screen.getByTestId('password-input');

      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      // Verify input value updated
      expect(passwordInput).toHaveValue('password123');
    });

    test('should handle multiple input changes correctly', () => {
      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');

      // Multiple changes to email
      fireEvent.change(emailInput, { target: { value: 't' } });
      fireEvent.change(emailInput, { target: { value: 'te' } });
      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });

      // Multiple changes to password
      fireEvent.change(passwordInput, { target: { value: 'p' } });
      fireEvent.change(passwordInput, { target: { value: 'pa' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      // Verify final values
      expect(emailInput).toHaveValue('test@example.com');
      expect(passwordInput).toHaveValue('password123');
    });

    test('should persist error until new submission', async () => {
      login.mockRejectedValueOnce(new Error('invalid credentials'));

      render(<LoginPage onLoginSuccess={mockOnLoginSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      const submitButton = screen.getByTestId('submit-button');

      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'wrongpassword' } });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toHaveTextContent('invalid credentials');
      });

      // Change inputs (error should persist)
      fireEvent.change(emailInput, { target: { value: 'new@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'newpassword' } });

      // Error still visible (only cleared on new submit)
      expect(screen.getByTestId('error-message')).toHaveTextContent('invalid credentials');
    });
  });
});
