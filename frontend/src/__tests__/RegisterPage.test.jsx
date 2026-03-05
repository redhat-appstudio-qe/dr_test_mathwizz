// Comprehensive unit tests for RegisterPage.jsx
// Tests registration form submission, validation (email + password), error handling, and callbacks
// Exposes Bug #42 (no loading state) and Bug #43 (no input trimming)

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import RegisterPage from '../components/RegisterPage';
import { register } from '../api';
import { validateEmail, validatePassword } from '../utils';

// Mock the API module
jest.mock('../api', () => ({
  register: jest.fn(),
}));

// Mock the utils module
jest.mock('../utils', () => ({
  validateEmail: jest.fn(),
  validatePassword: jest.fn(),
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

describe('RegisterPage Component', () => {
  let mockOnRegisterSuccess;

  beforeEach(() => {
    // Reset all mocks before each test
    jest.clearAllMocks();

    // Create fresh mock function for each test
    mockOnRegisterSuccess = jest.fn();

    // Default: validation functions return null (valid)
    validateEmail.mockReturnValue(null);
    validatePassword.mockReturnValue(null);
  });

  describe('when testing component rendering', () => {
    test('should render LoginInputComponent with correct initial state', () => {
      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      // Verify LoginInputComponent is rendered (reused from LoginPage)
      expect(screen.getByTestId('login-input-component')).toBeInTheDocument();

      // Verify button text is "CREATE ACCOUNT" (different from LoginPage)
      expect(screen.getByTestId('submit-button')).toHaveTextContent('CREATE ACCOUNT');

      // Verify inputs are empty initially
      expect(screen.getByTestId('email-input')).toHaveValue('');
      expect(screen.getByTestId('password-input')).toHaveValue('');

      // Verify no error message initially
      expect(screen.queryByTestId('error-message')).not.toBeInTheDocument();
    });

    test('should render page container with correct className', () => {
      const { container } = render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      // Verify page-container div exists
      const pageContainer = container.querySelector('.page-container');
      expect(pageContainer).toBeInTheDocument();
    });
  });

  describe('when testing successful registration flow', () => {
    test('should call register API and onRegisterSuccess callback with valid credentials', async () => {
      // Mock successful API response
      register.mockResolvedValueOnce({ email: 'newuser@example.com', user_id: 2 });

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      // Fill in email and password
      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      const submitButton = screen.getByTestId('submit-button');

      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      // Submit form
      fireEvent.click(submitButton);

      // Wait for async operations
      await waitFor(() => {
        // Verify validateEmail was called with email
        expect(validateEmail).toHaveBeenCalledWith('newuser@example.com');

        // Verify validatePassword was called with password
        expect(validatePassword).toHaveBeenCalledWith('password123');

        // Verify register API was called with email and password
        expect(register).toHaveBeenCalledWith('newuser@example.com', 'password123');

        // Verify onRegisterSuccess callback was called
        expect(mockOnRegisterSuccess).toHaveBeenCalledTimes(1);
      });

      // Verify no error message displayed
      expect(screen.queryByTestId('error-message')).not.toBeInTheDocument();
    });

    test('should handle Enter key submission in form', async () => {
      register.mockResolvedValueOnce({ email: 'newuser@example.com', user_id: 2 });

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');

      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      // Submit form via Enter key (submit event on form)
      const form = screen.getByTestId('login-input-component').querySelector('form');
      fireEvent.submit(form);

      await waitFor(() => {
        expect(register).toHaveBeenCalledWith('newuser@example.com', 'password123');
        expect(mockOnRegisterSuccess).toHaveBeenCalledTimes(1);
      });
    });

    test('should clear error message from previous attempt on successful registration', async () => {
      // First attempt: password validation error
      validatePassword.mockReturnValueOnce('Password must be at least 6 characters');

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // Verify error is shown
      expect(screen.getByTestId('error-message')).toHaveTextContent('Password must be at least 6 characters');

      // Second attempt: successful registration
      validatePassword.mockReturnValue(null);
      register.mockResolvedValueOnce({ email: 'newuser@example.com', user_id: 2 });

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');

      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(mockOnRegisterSuccess).toHaveBeenCalled();
      });

      // Verify error is cleared after successful registration
      expect(screen.queryByTestId('error-message')).not.toBeInTheDocument();
    });

    test('should handle minimum valid password length (6 characters)', async () => {
      register.mockResolvedValueOnce({ email: 'newuser@example.com', user_id: 2 });

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');

      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });
      fireEvent.change(passwordInput, { target: { value: '123456' } }); // Exactly 6 chars

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(register).toHaveBeenCalledWith('newuser@example.com', '123456');
        expect(mockOnRegisterSuccess).toHaveBeenCalled();
      });
    });
  });

  describe('when testing email validation errors', () => {
    test('should display error when email is empty', () => {
      validateEmail.mockReturnValue('Email is required');

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // Verify error message is displayed
      expect(screen.getByTestId('error-message')).toHaveTextContent('Email is required');

      // Verify register API was NOT called
      expect(register).not.toHaveBeenCalled();

      // Verify onRegisterSuccess was NOT called
      expect(mockOnRegisterSuccess).not.toHaveBeenCalled();
    });

    test('should display error when email format is invalid', () => {
      validateEmail.mockReturnValue('Invalid email format');

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      fireEvent.change(emailInput, { target: { value: 'notanemail' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // Verify error message is displayed
      expect(screen.getByTestId('error-message')).toHaveTextContent('Invalid email format');

      // Verify register API was NOT called
      expect(register).not.toHaveBeenCalled();
    });

    test('should validate email before password (short-circuit)', () => {
      validateEmail.mockReturnValue('Invalid email format');
      // validatePassword should not be called if email validation fails

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // Email validation error shown first
      expect(screen.getByTestId('error-message')).toHaveTextContent('Invalid email format');

      // Verify validatePassword was NOT called (short-circuit validation)
      expect(validatePassword).not.toHaveBeenCalled();
    });
  });

  describe('when testing password validation errors', () => {
    test('should display error when password is empty', () => {
      validatePassword.mockReturnValue('Password is required');

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // Verify error message is displayed
      expect(screen.getByTestId('error-message')).toHaveTextContent('Password is required');

      // Verify register API was NOT called
      expect(register).not.toHaveBeenCalled();
    });

    test('should display error when password is too short (< 6 characters)', () => {
      validatePassword.mockReturnValue('Password must be at least 6 characters');

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });
      fireEvent.change(passwordInput, { target: { value: '12345' } }); // 5 chars

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // Verify password validation error
      expect(screen.getByTestId('error-message')).toHaveTextContent('Password must be at least 6 characters');

      // Verify register API was NOT called
      expect(register).not.toHaveBeenCalled();
    });

    test('should display error when both email and password are empty', () => {
      validateEmail.mockReturnValue('Email is required');
      // validatePassword should not be called due to short-circuit

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // Verify email validation error is displayed first (short-circuit)
      expect(screen.getByTestId('error-message')).toHaveTextContent('Email is required');

      // Verify register API was NOT called
      expect(register).not.toHaveBeenCalled();
    });

    test('should clear previous error when submitting again', () => {
      validatePassword
        .mockReturnValueOnce('Password must be at least 6 characters')
        .mockReturnValueOnce('Password is required');

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });

      const submitButton = screen.getByTestId('submit-button');

      // First submit: password too short
      fireEvent.click(submitButton);
      expect(screen.getByTestId('error-message')).toHaveTextContent('Password must be at least 6 characters');

      // Second submit: password is required
      fireEvent.click(submitButton);
      expect(screen.getByTestId('error-message')).toHaveTextContent('Password is required');

      // Verify error was updated (not concatenated)
      expect(screen.queryByText('Password must be at least 6 characters')).not.toBeInTheDocument();
    });
  });

  describe('when testing API error handling', () => {
    test('should display error message when register API fails', async () => {
      // Mock API rejection (e.g., duplicate email)
      register.mockRejectedValueOnce(new Error('failed to create user'));

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      fireEvent.change(emailInput, { target: { value: 'existing@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // Wait for async error handling
      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toHaveTextContent('failed to create user');
      });

      // Verify onRegisterSuccess was NOT called on failure
      expect(mockOnRegisterSuccess).not.toHaveBeenCalled();
    });

    test('should display network error message', async () => {
      register.mockRejectedValueOnce(new Error('Network error'));

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toHaveTextContent('Network error');
      });
    });

    test('should not clear input fields when API fails', async () => {
      register.mockRejectedValueOnce(new Error('failed to create user'));

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      // Verify inputs still have values (user can retry)
      expect(emailInput).toHaveValue('newuser@example.com');
      expect(passwordInput).toHaveValue('password123');
    });
  });

  describe('when testing input trimming (Bug #43)', () => {
    test('should handle email with whitespace (HTML5 email input auto-trims)', async () => {
      /**
       * NOTE: HTML5 email input type automatically trims leading/trailing whitespace.
       *
       * CURRENT BEHAVIOR: Email input with type="email" auto-trims spaces before onChange fires.
       * So "  newuser@example.com  " becomes "newuser@example.com" in the input value.
       *
       * This is actually GOOD behavior from the browser. Bug #43 is more about
       * password inputs (which don't auto-trim) and explicit validation before API calls.
       */
      register.mockResolvedValueOnce({ email: 'newuser@example.com', user_id: 2 });

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');

      // Email with leading/trailing spaces (browser will auto-trim)
      fireEvent.change(emailInput, { target: { value: '  newuser@example.com  ' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      await waitFor(() => {
        // HTML5 email input auto-trims, so register receives trimmed email
        expect(register).toHaveBeenCalledWith('newuser@example.com', 'password123');
      });
    });

    test('should NOT trim leading/trailing whitespace from password - exposes Bug #43', async () => {
      /**
       * BUG #43: RegisterPage does NOT trim whitespace from password input.
       *
       * CURRENT BEHAVIOR: Password "  mypassword  " is sent to API with spaces.
       * EXPECTED BEHAVIOR: Password should be trimmed before API call.
       */
      register.mockResolvedValueOnce({ email: 'newuser@example.com', user_id: 2 });

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');

      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });
      // Password with leading/trailing spaces
      fireEvent.change(passwordInput, { target: { value: '  password123  ' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      await waitFor(() => {
        // BUG #43: Password is NOT trimmed
        expect(register).toHaveBeenCalledWith('newuser@example.com', '  password123  ');
      });
    });

    test('should preserve internal whitespace in email (not trim middle spaces)', async () => {
      register.mockResolvedValueOnce({ email: 'new user@example.com', user_id: 2 });

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');

      // Email with space in middle (invalid email, but tests trimming behavior)
      fireEvent.change(emailInput, { target: { value: 'new user@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      await waitFor(() => {
        // Internal spaces preserved (not trimmed)
        expect(register).toHaveBeenCalledWith('new user@example.com', 'password123');
      });
    });
  });

  describe('when testing loading state (Bug #42)', () => {
    test('should NOT show loading state during API call - exposes Bug #42', async () => {
      /**
       * BUG #42: RegisterPage has NO loading state during registration.
       *
       * CURRENT BEHAVIOR: No visual feedback during API call (button stays enabled, no loading text).
       * EXPECTED BEHAVIOR: Should show loading indicator and disable button during API call.
       *
       * IMPACT:
       * - Users can double-click submit button, causing duplicate API calls (multiple accounts)
       * - No visual feedback that registration is in progress
       * - Poor UX, especially on slow networks
       *
       * This test verifies current behavior (no loading state). When Bug #42 is fixed,
       * this test should be updated to verify loading state is shown.
       */
      // Use Promise to control timing of API resolution
      let resolveRegister;
      const registerPromise = new Promise((resolve) => {
        resolveRegister = resolve;
      });
      register.mockReturnValue(registerPromise);

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);

      // BUG #42: No loading indicator shown (this assertion should fail when bug is fixed)
      expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
      expect(submitButton).not.toBeDisabled(); // Button remains enabled during API call

      // Resolve the promise
      resolveRegister({ email: 'newuser@example.com', user_id: 2 });

      await waitFor(() => {
        expect(mockOnRegisterSuccess).toHaveBeenCalled();
      });
    });

    test('should allow multiple rapid clicks (no debouncing) - exposes Bug #42', async () => {
      /**
       * BUG #42: Without loading state/disabled button, users can click submit multiple times.
       *
       * This causes multiple API calls for same registration, potentially creating duplicate accounts.
       */
      register.mockResolvedValue({ email: 'newuser@example.com', user_id: 2 });

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      const submitButton = screen.getByTestId('submit-button');

      // Rapid clicks (should be prevented with loading state/disabled button)
      fireEvent.click(submitButton);
      fireEvent.click(submitButton);
      fireEvent.click(submitButton);

      await waitFor(() => {
        // BUG #42: Register API called multiple times (no debouncing)
        // When fixed, register should only be called once
        expect(register).toHaveBeenCalledTimes(3);
      });
    });
  });

  describe('when testing state updates and re-renders', () => {
    test('should update email state on input change', () => {
      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');

      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });

      // Verify input value updated (controlled component)
      expect(emailInput).toHaveValue('newuser@example.com');
    });

    test('should update password state on input change', () => {
      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const passwordInput = screen.getByTestId('password-input');

      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      // Verify input value updated
      expect(passwordInput).toHaveValue('password123');
    });

    test('should handle multiple input changes correctly', () => {
      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');

      // Multiple changes to email
      fireEvent.change(emailInput, { target: { value: 'n' } });
      fireEvent.change(emailInput, { target: { value: 'ne' } });
      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });

      // Multiple changes to password
      fireEvent.change(passwordInput, { target: { value: 'p' } });
      fireEvent.change(passwordInput, { target: { value: 'pa' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });

      // Verify final values
      expect(emailInput).toHaveValue('newuser@example.com');
      expect(passwordInput).toHaveValue('password123');
    });

    test('should persist error until new submission', async () => {
      register.mockRejectedValueOnce(new Error('failed to create user'));

      render(<RegisterPage onRegisterSuccess={mockOnRegisterSuccess} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      const submitButton = screen.getByTestId('submit-button');

      fireEvent.change(emailInput, { target: { value: 'existing@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toHaveTextContent('failed to create user');
      });

      // Change inputs (error should persist)
      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'newpassword' } });

      // Error still visible (only cleared on new submit)
      expect(screen.getByTestId('error-message')).toHaveTextContent('failed to create user');
    });
  });
});
