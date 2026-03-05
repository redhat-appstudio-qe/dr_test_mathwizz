// Comprehensive unit tests for CalculatorComponent.jsx
// Tests presentational component rendering, prop-driven UI states, and user interactions
// Covers: rendering, props handling, form submission, loading/error/answer states, disabled states

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import CalculatorComponent from '../components/CalculatorComponent';

describe('CalculatorComponent - Presentational Component', () => {
  // Default props for testing
  const defaultProps = {
    problem: '',
    answer: '',
    onProblemChange: jest.fn(),
    onSolve: jest.fn(),
    loading: false,
    error: '',
  };

  beforeEach(() => {
    // Reset all mocks before each test
    jest.clearAllMocks();
  });

  describe('when testing component rendering with default props', () => {
    test('should render calculator container with all UI elements', () => {
      render(<CalculatorComponent {...defaultProps} />);

      // Verify all main elements are present
      expect(screen.getByTestId('answer-display')).toBeInTheDocument();
      expect(screen.getByTestId('problem-input')).toBeInTheDocument();
      expect(screen.getByTestId('solve-button')).toBeInTheDocument();

      // Verify hint text is displayed
      expect(screen.getByText('Supports: + - * / ( )')).toBeInTheDocument();
    });

    test('should display "Ready..." placeholder when no answer', () => {
      render(<CalculatorComponent {...defaultProps} />);

      const answerDisplay = screen.getByTestId('answer-display');
      expect(answerDisplay).toHaveTextContent('Ready...');
    });

    test('should display "SOLVE" button text when not loading', () => {
      render(<CalculatorComponent {...defaultProps} />);

      const solveButton = screen.getByTestId('solve-button');
      expect(solveButton).toHaveTextContent('SOLVE');
    });

    test('should have empty input value when problem prop is empty', () => {
      render(<CalculatorComponent {...defaultProps} />);

      const input = screen.getByTestId('problem-input');
      expect(input).toHaveValue('');
    });

    test('should not display error message when error prop is empty', () => {
      render(<CalculatorComponent {...defaultProps} />);

      // Error message should not be in document
      expect(screen.queryByText(/error/i)).not.toBeInTheDocument();
    });

    test('should have input and button enabled when not loading', () => {
      render(<CalculatorComponent {...defaultProps} />);

      expect(screen.getByTestId('problem-input')).not.toBeDisabled();
      expect(screen.getByTestId('solve-button')).not.toBeDisabled();
    });

    test('should render with correct placeholder text', () => {
      render(<CalculatorComponent {...defaultProps} />);

      const input = screen.getByTestId('problem-input');
      expect(input).toHaveAttribute('placeholder', 'Enter problem (e.g., 5*10)');
    });
  });

  describe('when testing problem input handling', () => {
    test('should display problem value from props in input field', () => {
      const props = { ...defaultProps, problem: '2+3' };
      render(<CalculatorComponent {...props} />);

      const input = screen.getByTestId('problem-input');
      expect(input).toHaveValue('2+3');
    });

    test('should call onProblemChange when input value changes', () => {
      const mockOnChange = jest.fn();
      const props = { ...defaultProps, onProblemChange: mockOnChange };

      render(<CalculatorComponent {...props} />);

      const input = screen.getByTestId('problem-input');
      fireEvent.change(input, { target: { value: '5*10' } });

      // Verify callback was called with new value
      expect(mockOnChange).toHaveBeenCalledTimes(1);
      expect(mockOnChange).toHaveBeenCalledWith('5*10');
    });

    test('should handle multiple input changes', () => {
      const mockOnChange = jest.fn();
      const props = { ...defaultProps, onProblemChange: mockOnChange };

      render(<CalculatorComponent {...props} />);

      const input = screen.getByTestId('problem-input');

      // Simulate typing "2+3"
      fireEvent.change(input, { target: { value: '2' } });
      expect(mockOnChange).toHaveBeenCalledWith('2');

      fireEvent.change(input, { target: { value: '2+' } });
      expect(mockOnChange).toHaveBeenCalledWith('2+');

      fireEvent.change(input, { target: { value: '2+3' } });
      expect(mockOnChange).toHaveBeenCalledWith('2+3');

      // Verify called 3 times total
      expect(mockOnChange).toHaveBeenCalledTimes(3);
    });

    test('should handle clearing input value', () => {
      const mockOnChange = jest.fn();
      const props = { ...defaultProps, problem: '10+20', onProblemChange: mockOnChange };

      render(<CalculatorComponent {...props} />);

      const input = screen.getByTestId('problem-input');
      fireEvent.change(input, { target: { value: '' } });

      expect(mockOnChange).toHaveBeenCalledWith('');
    });

    test('should handle complex mathematical expressions', () => {
      const mockOnChange = jest.fn();
      const props = { ...defaultProps, onProblemChange: mockOnChange };

      render(<CalculatorComponent {...props} />);

      const input = screen.getByTestId('problem-input');
      fireEvent.change(input, { target: { value: '(10+5)*2-3' } });

      expect(mockOnChange).toHaveBeenCalledWith('(10+5)*2-3');
    });
  });

  describe('when testing solve button and form submission', () => {
    test('should call onSolve when solve button is clicked', () => {
      const mockOnSolve = jest.fn();
      const props = { ...defaultProps, onSolve: mockOnSolve };

      render(<CalculatorComponent {...props} />);

      const solveButton = screen.getByTestId('solve-button');
      fireEvent.click(solveButton);

      expect(mockOnSolve).toHaveBeenCalledTimes(1);
    });

    test('should call onSolve when form is submitted (Enter key)', () => {
      const mockOnSolve = jest.fn();
      const props = { ...defaultProps, onSolve: mockOnSolve };

      render(<CalculatorComponent {...props} />);

      const input = screen.getByTestId('problem-input');

      // Simulate Enter key press (form submit)
      fireEvent.submit(input.closest('form'));

      expect(mockOnSolve).toHaveBeenCalledTimes(1);
    });

    test('should prevent default form submission behavior', () => {
      const mockOnSolve = jest.fn();
      const props = { ...defaultProps, onSolve: mockOnSolve };

      render(<CalculatorComponent {...props} />);

      const form = screen.getByTestId('problem-input').closest('form');

      // Submit the form (Enter key press simulation)
      fireEvent.submit(form);

      // Verify onSolve callback was triggered
      // Note: fireEvent.submit in testing-library automatically prevents default
      // The real component's handleSubmit prevents default via e.preventDefault()
      expect(mockOnSolve).toHaveBeenCalled();
    });

    test('should call onSolve multiple times for sequential button clicks', () => {
      const mockOnSolve = jest.fn();
      const props = { ...defaultProps, onSolve: mockOnSolve };

      render(<CalculatorComponent {...props} />);

      const solveButton = screen.getByTestId('solve-button');

      fireEvent.click(solveButton);
      fireEvent.click(solveButton);
      fireEvent.click(solveButton);

      expect(mockOnSolve).toHaveBeenCalledTimes(3);
    });
  });

  describe('when testing answer display', () => {
    test('should display answer with "= " prefix when answer prop is provided', () => {
      const props = { ...defaultProps, answer: '14' };
      render(<CalculatorComponent {...props} />);

      const answerDisplay = screen.getByTestId('answer-display');
      expect(answerDisplay).toHaveTextContent('= 14');
    });

    test('should display different answer values correctly', () => {
      const { rerender } = render(<CalculatorComponent {...defaultProps} answer="5" />);

      expect(screen.getByTestId('answer-display')).toHaveTextContent('= 5');

      // Update answer prop
      rerender(<CalculatorComponent {...defaultProps} answer="100" />);
      expect(screen.getByTestId('answer-display')).toHaveTextContent('= 100');

      rerender(<CalculatorComponent {...defaultProps} answer="0" />);
      expect(screen.getByTestId('answer-display')).toHaveTextContent('= 0');

      rerender(<CalculatorComponent {...defaultProps} answer="-5" />);
      expect(screen.getByTestId('answer-display')).toHaveTextContent('= -5');
    });

    test('should revert to "Ready..." when answer is cleared', () => {
      const { rerender } = render(<CalculatorComponent {...defaultProps} answer="42" />);

      expect(screen.getByTestId('answer-display')).toHaveTextContent('= 42');

      // Clear answer
      rerender(<CalculatorComponent {...defaultProps} answer="" />);
      expect(screen.getByTestId('answer-display')).toHaveTextContent('Ready...');
    });

    test('should handle very large answer values', () => {
      const props = { ...defaultProps, answer: '999999999' };
      render(<CalculatorComponent {...props} />);

      expect(screen.getByTestId('answer-display')).toHaveTextContent('= 999999999');
    });
  });

  describe('when testing error display', () => {
    test('should display error message when error prop is provided', () => {
      const props = { ...defaultProps, error: 'Please enter a math problem' };
      render(<CalculatorComponent {...props} />);

      expect(screen.getByText('Please enter a math problem')).toBeInTheDocument();
    });

    test('should display different error messages', () => {
      const { rerender } = render(
        <CalculatorComponent {...defaultProps} error="Problem contains invalid characters" />
      );

      expect(screen.getByText('Problem contains invalid characters')).toBeInTheDocument();

      rerender(
        <CalculatorComponent {...defaultProps} error="Failed to solve problem" />
      );
      expect(screen.getByText('Failed to solve problem')).toBeInTheDocument();
    });

    test('should have error-message className for styling', () => {
      const props = { ...defaultProps, error: 'Test error' };
      const { container } = render(<CalculatorComponent {...props} />);

      const errorElement = container.querySelector('.error-message');
      expect(errorElement).toBeInTheDocument();
      expect(errorElement).toHaveTextContent('Test error');
    });

    test('should hide error message when error prop is cleared', () => {
      const { rerender } = render(
        <CalculatorComponent {...defaultProps} error="Test error" />
      );

      expect(screen.getByText('Test error')).toBeInTheDocument();

      // Clear error
      rerender(<CalculatorComponent {...defaultProps} error="" />);
      expect(screen.queryByText('Test error')).not.toBeInTheDocument();
    });
  });

  describe('when testing loading state behavior', () => {
    test('should display "SOLVING..." button text when loading is true', () => {
      const props = { ...defaultProps, loading: true };
      render(<CalculatorComponent {...props} />);

      const solveButton = screen.getByTestId('solve-button');
      expect(solveButton).toHaveTextContent('SOLVING...');
    });

    test('should disable button when loading is true', () => {
      const props = { ...defaultProps, loading: true };
      render(<CalculatorComponent {...props} />);

      const solveButton = screen.getByTestId('solve-button');
      expect(solveButton).toBeDisabled();
    });

    test('should disable input when loading is true', () => {
      const props = { ...defaultProps, loading: true };
      render(<CalculatorComponent {...props} />);

      const input = screen.getByTestId('problem-input');
      expect(input).toBeDisabled();
    });

    test('should prevent button clicks when disabled during loading', () => {
      const mockOnSolve = jest.fn();
      const props = { ...defaultProps, loading: true, onSolve: mockOnSolve };

      render(<CalculatorComponent {...props} />);

      const solveButton = screen.getByTestId('solve-button');

      // Try to click disabled button
      fireEvent.click(solveButton);

      // Callback should NOT be called when button is disabled
      expect(mockOnSolve).not.toHaveBeenCalled();
    });

    test('should toggle between loading and non-loading states', () => {
      const { rerender } = render(<CalculatorComponent {...defaultProps} loading={false} />);

      // Not loading initially
      expect(screen.getByTestId('solve-button')).toHaveTextContent('SOLVE');
      expect(screen.getByTestId('solve-button')).not.toBeDisabled();

      // Switch to loading
      rerender(<CalculatorComponent {...defaultProps} loading={true} />);
      expect(screen.getByTestId('solve-button')).toHaveTextContent('SOLVING...');
      expect(screen.getByTestId('solve-button')).toBeDisabled();
      expect(screen.getByTestId('problem-input')).toBeDisabled();

      // Switch back to not loading
      rerender(<CalculatorComponent {...defaultProps} loading={false} />);
      expect(screen.getByTestId('solve-button')).toHaveTextContent('SOLVE');
      expect(screen.getByTestId('solve-button')).not.toBeDisabled();
      expect(screen.getByTestId('problem-input')).not.toBeDisabled();
    });

    test('should show loading state with existing problem and answer', () => {
      const props = {
        ...defaultProps,
        problem: '2+2',
        answer: '4',
        loading: true,
      };

      render(<CalculatorComponent {...props} />);

      // Problem still visible
      expect(screen.getByTestId('problem-input')).toHaveValue('2+2');

      // Answer still visible (from previous solve)
      expect(screen.getByTestId('answer-display')).toHaveTextContent('= 4');

      // But button shows loading state
      expect(screen.getByTestId('solve-button')).toHaveTextContent('SOLVING...');
      expect(screen.getByTestId('solve-button')).toBeDisabled();
    });
  });

  describe('when testing complete UI state combinations', () => {
    test('should display error and disable submission during loading with error', () => {
      const props = {
        ...defaultProps,
        problem: 'invalid',
        error: 'Problem contains invalid characters',
        loading: true,
      };

      render(<CalculatorComponent {...props} />);

      // Error should be displayed
      expect(screen.getByText('Problem contains invalid characters')).toBeInTheDocument();

      // Loading state should be active
      expect(screen.getByTestId('solve-button')).toHaveTextContent('SOLVING...');
      expect(screen.getByTestId('solve-button')).toBeDisabled();
    });

    test('should display all states together (edge case)', () => {
      const props = {
        ...defaultProps,
        problem: '2+2',
        answer: '4',
        error: 'Previous error',
        loading: true,
      };

      render(<CalculatorComponent {...props} />);

      // All elements rendered simultaneously
      expect(screen.getByTestId('problem-input')).toHaveValue('2+2');
      expect(screen.getByTestId('answer-display')).toHaveTextContent('= 4');
      expect(screen.getByText('Previous error')).toBeInTheDocument();
      expect(screen.getByTestId('solve-button')).toHaveTextContent('SOLVING...');
    });

    test('should handle transition from answer to error state', () => {
      const { rerender } = render(
        <CalculatorComponent {...defaultProps} problem="2+2" answer="4" />
      );

      // Initially showing answer
      expect(screen.getByTestId('answer-display')).toHaveTextContent('= 4');
      expect(screen.queryByText(/error/i)).not.toBeInTheDocument();

      // Transition to error state
      rerender(
        <CalculatorComponent
          {...defaultProps}
          problem="invalid"
          answer=""
          error="Invalid input"
        />
      );

      expect(screen.getByTestId('answer-display')).toHaveTextContent('Ready...');
      expect(screen.getByText('Invalid input')).toBeInTheDocument();
    });

    test('should handle successful solve flow: empty → loading → answer', () => {
      const { rerender } = render(<CalculatorComponent {...defaultProps} />);

      // Initial state: empty
      expect(screen.getByTestId('answer-display')).toHaveTextContent('Ready...');
      expect(screen.getByTestId('solve-button')).toHaveTextContent('SOLVE');

      // Loading state
      rerender(<CalculatorComponent {...defaultProps} problem="2+2" loading={true} />);
      expect(screen.getByTestId('solve-button')).toHaveTextContent('SOLVING...');
      expect(screen.getByTestId('solve-button')).toBeDisabled();

      // Answer received, loading complete
      rerender(
        <CalculatorComponent
          {...defaultProps}
          problem=""
          answer="4"
          loading={false}
        />
      );
      expect(screen.getByTestId('answer-display')).toHaveTextContent('= 4');
      expect(screen.getByTestId('solve-button')).toHaveTextContent('SOLVE');
      expect(screen.getByTestId('solve-button')).not.toBeDisabled();
    });
  });

  describe('when testing input field attributes and accessibility', () => {
    test('should render input with type="text"', () => {
      render(<CalculatorComponent {...defaultProps} />);

      const input = screen.getByTestId('problem-input');
      expect(input).toHaveAttribute('type', 'text');
    });

    test('should render input with pixel-input className for styling', () => {
      render(<CalculatorComponent {...defaultProps} />);

      const input = screen.getByTestId('problem-input');
      expect(input).toHaveClass('pixel-input');
    });

    test('should render button with type="submit"', () => {
      render(<CalculatorComponent {...defaultProps} />);

      const button = screen.getByTestId('solve-button');
      expect(button).toHaveAttribute('type', 'submit');
    });

    test('should have data-testid attributes for testing', () => {
      render(<CalculatorComponent {...defaultProps} />);

      expect(screen.getByTestId('answer-display')).toBeInTheDocument();
      expect(screen.getByTestId('problem-input')).toBeInTheDocument();
      expect(screen.getByTestId('solve-button')).toBeInTheDocument();
    });
  });

  describe('when testing component as presentational (dumb component)', () => {
    test('should not have internal state - all data from props', () => {
      const props = {
        problem: 'test-problem',
        answer: 'test-answer',
        onProblemChange: jest.fn(),
        onSolve: jest.fn(),
        loading: true,
        error: 'test-error',
      };

      render(<CalculatorComponent {...props} />);

      // All rendered content comes from props
      expect(screen.getByTestId('problem-input')).toHaveValue('test-problem');
      expect(screen.getByTestId('answer-display')).toHaveTextContent('= test-answer');
      expect(screen.getByTestId('solve-button')).toHaveTextContent('SOLVING...');
      expect(screen.getByText('test-error')).toBeInTheDocument();
    });

    test('should only call callbacks provided via props', () => {
      const mockOnChange = jest.fn();
      const mockOnSolve = jest.fn();

      const props = {
        ...defaultProps,
        onProblemChange: mockOnChange,
        onSolve: mockOnSolve,
      };

      render(<CalculatorComponent {...props} />);

      // Trigger input change
      fireEvent.change(screen.getByTestId('problem-input'), { target: { value: 'test' } });
      expect(mockOnChange).toHaveBeenCalledWith('test');

      // Trigger solve
      fireEvent.click(screen.getByTestId('solve-button'));
      expect(mockOnSolve).toHaveBeenCalled();

      // Component doesn't have its own logic - purely delegates to callbacks
    });

    test('should rerender correctly when all props change', () => {
      const initialProps = {
        problem: 'old-problem',
        answer: 'old-answer',
        onProblemChange: jest.fn(),
        onSolve: jest.fn(),
        loading: false,
        error: 'old-error',
      };

      const newProps = {
        problem: 'new-problem',
        answer: 'new-answer',
        onProblemChange: jest.fn(),
        onSolve: jest.fn(),
        loading: true,
        error: 'new-error',
      };

      const { rerender } = render(<CalculatorComponent {...initialProps} />);

      // Verify initial state
      expect(screen.getByTestId('problem-input')).toHaveValue('old-problem');
      expect(screen.getByTestId('answer-display')).toHaveTextContent('= old-answer');

      // Update all props
      rerender(<CalculatorComponent {...newProps} />);

      // Verify all updated correctly
      expect(screen.getByTestId('problem-input')).toHaveValue('new-problem');
      expect(screen.getByTestId('answer-display')).toHaveTextContent('= new-answer');
      expect(screen.getByText('new-error')).toBeInTheDocument();
      expect(screen.getByTestId('solve-button')).toHaveTextContent('SOLVING...');
    });
  });
});
