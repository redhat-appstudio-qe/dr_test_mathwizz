// Comprehensive unit tests for LoginInputComponent.jsx
// Tests presentational component rendering, form submission, input handling, and prop forwarding
// Pure presentation component - all data and handlers come from props

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import LoginInputComponent from '../components/LoginInputComponent';

describe('LoginInputComponent - Presentational Component', () => {
  // Mock props for all tests
  let mockProps;

  beforeEach(() => {
    // Create fresh mock functions for each test
    mockProps = {
      email: '',
      password: '',
      onEmailChange: jest.fn(),
      onPasswordChange: jest.fn(),
      onSubmit: jest.fn((e) => e.preventDefault()), // Prevent default to avoid console warnings
      buttonText: 'SUBMIT',
      error: '',
    };
  });

  describe('when testing component rendering with default props', () => {
    test('should render form container with correct styling', () => {
      const { container } = render(<LoginInputComponent {...mockProps} />);

      // Verify form container exists (first div child)
      const formContainer = container.firstChild;
      expect(formContainer).toBeInTheDocument();

      // Verify inline styles are applied (pixel art retro theme)
      expect(formContainer).toHaveStyle({
        background: 'rgba(22, 33, 62, 0.8)',
        border: '4px solid #0f3460',
      });
    });

    test('should render email input with correct attributes', () => {
      render(<LoginInputComponent {...mockProps} />);

      const emailInput = screen.getByTestId('email-input');

      // Verify input exists and has correct attributes
      expect(emailInput).toBeInTheDocument();
      expect(emailInput).toHaveAttribute('type', 'email');
      expect(emailInput).toHaveAttribute('placeholder', 'Email');
      expect(emailInput).toHaveAttribute('required');
      expect(emailInput).toHaveValue('');
    });

    test('should render password input with correct attributes', () => {
      render(<LoginInputComponent {...mockProps} />);

      const passwordInput = screen.getByTestId('password-input');

      // Verify input exists and has correct attributes
      expect(passwordInput).toBeInTheDocument();
      expect(passwordInput).toHaveAttribute('type', 'password');
      expect(passwordInput).toHaveAttribute('placeholder', 'Password');
      expect(passwordInput).toHaveAttribute('required');
      expect(passwordInput).toHaveValue('');
    });

    test('should render submit button with button text from props', () => {
      mockProps.buttonText = 'LOGIN';

      render(<LoginInputComponent {...mockProps} />);

      const submitButton = screen.getByTestId('submit-button');

      // Verify button exists and displays correct text
      expect(submitButton).toBeInTheDocument();
      expect(submitButton).toHaveAttribute('type', 'submit');
      expect(submitButton).toHaveTextContent('LOGIN');
    });

    test('should NOT render error message when error prop is empty', () => {
      mockProps.error = '';

      const { container } = render(<LoginInputComponent {...mockProps} />);

      // Verify no error message element exists
      const errorElement = container.querySelector('.error-message');
      expect(errorElement).not.toBeInTheDocument();
    });

    test('should have form element wrapping inputs and button', () => {
      const { container } = render(<LoginInputComponent {...mockProps} />);

      const form = container.querySelector('form');

      // Verify form exists
      expect(form).toBeInTheDocument();

      // Verify form contains both inputs and button
      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      const submitButton = screen.getByTestId('submit-button');

      expect(form).toContainElement(emailInput);
      expect(form).toContainElement(passwordInput);
      expect(form).toContainElement(submitButton);
    });

    test('should render with all data-testid attributes for automated testing', () => {
      render(<LoginInputComponent {...mockProps} />);

      // Verify all testid attributes exist (enables automated testing)
      expect(screen.getByTestId('email-input')).toBeInTheDocument();
      expect(screen.getByTestId('password-input')).toBeInTheDocument();
      expect(screen.getByTestId('submit-button')).toBeInTheDocument();
    });

    test('should have empty placeholder text when inputs are empty', () => {
      render(<LoginInputComponent {...mockProps} />);

      const emailInput = screen.getByPlaceholderText('Email');
      const passwordInput = screen.getByPlaceholderText('Password');

      // Verify placeholders are visible and inputs are empty
      expect(emailInput).toHaveValue('');
      expect(passwordInput).toHaveValue('');
    });
  });

  describe('when testing problem input handling', () => {
    test('should display email from props (controlled component)', () => {
      mockProps.email = 'test@example.com';

      render(<LoginInputComponent {...mockProps} />);

      const emailInput = screen.getByTestId('email-input');

      // Verify input displays value from props
      expect(emailInput).toHaveValue('test@example.com');
    });

    test('should display password from props (controlled component)', () => {
      mockProps.password = 'password123';

      render(<LoginInputComponent {...mockProps} />);

      const passwordInput = screen.getByTestId('password-input');

      // Verify input displays value from props (masked as password type)
      expect(passwordInput).toHaveValue('password123');
    });

    test('should call onEmailChange when email input changes', () => {
      render(<LoginInputComponent {...mockProps} />);

      const emailInput = screen.getByTestId('email-input');

      // Change email input value
      fireEvent.change(emailInput, { target: { value: 'newemail@example.com' } });

      // Verify callback was called with new value
      expect(mockProps.onEmailChange).toHaveBeenCalledTimes(1);
      expect(mockProps.onEmailChange).toHaveBeenCalledWith('newemail@example.com');
    });

    test('should call onPasswordChange when password input changes', () => {
      render(<LoginInputComponent {...mockProps} />);

      const passwordInput = screen.getByTestId('password-input');

      // Change password input value
      fireEvent.change(passwordInput, { target: { value: 'newpassword' } });

      // Verify callback was called with new value
      expect(mockProps.onPasswordChange).toHaveBeenCalledTimes(1);
      expect(mockProps.onPasswordChange).toHaveBeenCalledWith('newpassword');
    });

    test('should handle multiple email input changes', () => {
      render(<LoginInputComponent {...mockProps} />);

      const emailInput = screen.getByTestId('email-input');

      // Multiple changes (simulating typing)
      fireEvent.change(emailInput, { target: { value: 't' } });
      fireEvent.change(emailInput, { target: { value: 'te' } });
      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });

      // Verify callback called for each change
      expect(mockProps.onEmailChange).toHaveBeenCalledTimes(3);
      expect(mockProps.onEmailChange).toHaveBeenNthCalledWith(1, 't');
      expect(mockProps.onEmailChange).toHaveBeenNthCalledWith(2, 'te');
      expect(mockProps.onEmailChange).toHaveBeenNthCalledWith(3, 'test@example.com');
    });

    test('should handle clearing input (empty string)', () => {
      mockProps.email = 'test@example.com';

      render(<LoginInputComponent {...mockProps} />);

      const emailInput = screen.getByTestId('email-input');

      // Clear input (simulating user clearing field)
      fireEvent.change(emailInput, { target: { value: '' } });

      // Verify callback called with empty string
      expect(mockProps.onEmailChange).toHaveBeenCalledWith('');
    });
  });

  describe('when testing form submission', () => {
    test('should call onSubmit when submit button is clicked', () => {
      render(<LoginInputComponent {...mockProps} />);

      const submitButton = screen.getByTestId('submit-button');

      // Click submit button
      fireEvent.click(submitButton);

      // Verify onSubmit callback was called
      expect(mockProps.onSubmit).toHaveBeenCalledTimes(1);
    });

    test('should call onSubmit when Enter key pressed in email input', () => {
      render(<LoginInputComponent {...mockProps} />);

      const emailInput = screen.getByTestId('email-input');

      // Press Enter in email input (triggers form submit)
      fireEvent.submit(emailInput.form);

      // Verify onSubmit called
      expect(mockProps.onSubmit).toHaveBeenCalledTimes(1);
    });

    test('should call onSubmit when Enter key pressed in password input', () => {
      render(<LoginInputComponent {...mockProps} />);

      const passwordInput = screen.getByTestId('password-input');

      // Press Enter in password input (triggers form submit)
      fireEvent.submit(passwordInput.form);

      // Verify onSubmit called
      expect(mockProps.onSubmit).toHaveBeenCalledTimes(1);
    });

    test('should not reload page on form submission', () => {
      // Mock onSubmit that does NOT call preventDefault
      const onSubmitWithoutPreventDefault = jest.fn();
      mockProps.onSubmit = onSubmitWithoutPreventDefault;

      render(<LoginInputComponent {...mockProps} />);

      const submitButton = screen.getByTestId('submit-button');

      // Click submit
      fireEvent.click(submitButton);

      // Verify callback was called
      // Note: Parent component (LoginPage/RegisterPage) is responsible for calling e.preventDefault()
      expect(onSubmitWithoutPreventDefault).toHaveBeenCalled();
    });

    test('should handle multiple rapid button clicks', () => {
      render(<LoginInputComponent {...mockProps} />);

      const submitButton = screen.getByTestId('submit-button');

      // Rapid clicks
      fireEvent.click(submitButton);
      fireEvent.click(submitButton);
      fireEvent.click(submitButton);

      // Verify callback called for each click (no built-in debouncing)
      // Note: Parent component should handle debouncing/loading state
      expect(mockProps.onSubmit).toHaveBeenCalledTimes(3);
    });
  });

  describe('when testing error message display', () => {
    test('should display error message when error prop is non-empty', () => {
      mockProps.error = 'Invalid email format';

      render(<LoginInputComponent {...mockProps} />);

      // Verify error message is displayed
      const errorMessage = screen.getByText('Invalid email format');
      expect(errorMessage).toBeInTheDocument();
      expect(errorMessage).toHaveClass('error-message');
    });

    test('should display different error messages based on error prop', () => {
      mockProps.error = 'Password is required';

      const { rerender } = render(<LoginInputComponent {...mockProps} />);

      // Verify first error
      expect(screen.getByText('Password is required')).toBeInTheDocument();

      // Change error prop
      mockProps.error = 'Network error';
      rerender(<LoginInputComponent {...mockProps} />);

      // Verify error updated
      expect(screen.getByText('Network error')).toBeInTheDocument();
      expect(screen.queryByText('Password is required')).not.toBeInTheDocument();
    });

    test('should apply error-message className for styling', () => {
      mockProps.error = 'Some error';

      const { container } = render(<LoginInputComponent {...mockProps} />);

      // Verify error element has correct class
      const errorElement = container.querySelector('.error-message');
      expect(errorElement).toBeInTheDocument();
      expect(errorElement).toHaveTextContent('Some error');
    });

    test('should hide error message when error prop becomes empty', () => {
      mockProps.error = 'Initial error';

      const { rerender } = render(<LoginInputComponent {...mockProps} />);

      // Verify error shown
      expect(screen.getByText('Initial error')).toBeInTheDocument();

      // Clear error
      mockProps.error = '';
      rerender(<LoginInputComponent {...mockProps} />);

      // Verify error hidden
      expect(screen.queryByText('Initial error')).not.toBeInTheDocument();
    });
  });

  describe('when testing button text rendering', () => {
    test('should render "LOGIN" button text for LoginPage', () => {
      mockProps.buttonText = 'LOGIN';

      render(<LoginInputComponent {...mockProps} />);

      const submitButton = screen.getByTestId('submit-button');
      expect(submitButton).toHaveTextContent('LOGIN');
    });

    test('should render "CREATE ACCOUNT" button text for RegisterPage', () => {
      mockProps.buttonText = 'CREATE ACCOUNT';

      render(<LoginInputComponent {...mockProps} />);

      const submitButton = screen.getByTestId('submit-button');
      expect(submitButton).toHaveTextContent('CREATE ACCOUNT');
    });

    test('should update button text when buttonText prop changes', () => {
      mockProps.buttonText = 'LOGIN';

      const { rerender } = render(<LoginInputComponent {...mockProps} />);

      // Verify initial button text
      expect(screen.getByTestId('submit-button')).toHaveTextContent('LOGIN');

      // Change buttonText prop
      mockProps.buttonText = 'CREATE ACCOUNT';
      rerender(<LoginInputComponent {...mockProps} />);

      // Verify button text updated
      expect(screen.getByTestId('submit-button')).toHaveTextContent('CREATE ACCOUNT');
    });

    test('should handle empty button text', () => {
      mockProps.buttonText = '';

      render(<LoginInputComponent {...mockProps} />);

      const submitButton = screen.getByTestId('submit-button');
      expect(submitButton).toHaveTextContent(''); // Empty button (edge case)
    });
  });

  describe('when testing component behavior as presentational component', () => {
    test('should receive all data from props (no internal state)', () => {
      mockProps.email = 'test@example.com';
      mockProps.password = 'password123';
      mockProps.error = 'Some error';
      mockProps.buttonText = 'SUBMIT';

      render(<LoginInputComponent {...mockProps} />);

      // Verify all data from props is rendered
      expect(screen.getByTestId('email-input')).toHaveValue('test@example.com');
      expect(screen.getByTestId('password-input')).toHaveValue('password123');
      expect(screen.getByText('Some error')).toBeInTheDocument();
      expect(screen.getByTestId('submit-button')).toHaveTextContent('SUBMIT');
    });

    test('should delegate all behavior to parent callbacks', () => {
      render(<LoginInputComponent {...mockProps} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');
      const submitButton = screen.getByTestId('submit-button');

      // Trigger all interactions
      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
      fireEvent.change(passwordInput, { target: { value: 'password123' } });
      fireEvent.click(submitButton);

      // Verify all callbacks delegated to parent
      expect(mockProps.onEmailChange).toHaveBeenCalled();
      expect(mockProps.onPasswordChange).toHaveBeenCalled();
      expect(mockProps.onSubmit).toHaveBeenCalled();
    });

    test('should re-render when props change (controlled component pattern)', () => {
      mockProps.email = 'initial@example.com';

      const { rerender } = render(<LoginInputComponent {...mockProps} />);

      // Verify initial value
      expect(screen.getByTestId('email-input')).toHaveValue('initial@example.com');

      // Parent updates props (simulating state change in LoginPage/RegisterPage)
      mockProps.email = 'updated@example.com';
      rerender(<LoginInputComponent {...mockProps} />);

      // Verify component re-rendered with new prop value
      expect(screen.getByTestId('email-input')).toHaveValue('updated@example.com');
    });
  });

  describe('when testing input attributes and accessibility', () => {
    test('should use type="email" for HTML5 email validation', () => {
      render(<LoginInputComponent {...mockProps} />);

      const emailInput = screen.getByTestId('email-input');
      expect(emailInput).toHaveAttribute('type', 'email');
    });

    test('should use type="password" to mask password input', () => {
      render(<LoginInputComponent {...mockProps} />);

      const passwordInput = screen.getByTestId('password-input');
      expect(passwordInput).toHaveAttribute('type', 'password');
    });

    test('should have required attribute on both inputs', () => {
      render(<LoginInputComponent {...mockProps} />);

      const emailInput = screen.getByTestId('email-input');
      const passwordInput = screen.getByTestId('password-input');

      // HTML5 required attribute
      expect(emailInput).toHaveAttribute('required');
      expect(passwordInput).toHaveAttribute('required');
    });

    test('should have correct button type for form submission', () => {
      render(<LoginInputComponent {...mockProps} />);

      const submitButton = screen.getByTestId('submit-button');

      // type="submit" enables form submission on Enter key
      expect(submitButton).toHaveAttribute('type', 'submit');
    });
  });
});
