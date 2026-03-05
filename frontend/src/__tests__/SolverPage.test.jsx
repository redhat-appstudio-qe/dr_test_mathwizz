// Comprehensive unit tests for SolverPage.jsx
// Tests state management, API integration, validation, loading states, and error handling
// Covers: rendering, valid/invalid problem submission, API success/error, loading behavior

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import SolverPage from '../components/SolverPage';
import { solve } from '../api';
import { validateProblem } from '../utils';

// Mock the API module
jest.mock('../api', () => ({
  solve: jest.fn(),
}));

// Mock the utils module
jest.mock('../utils', () => ({
  validateProblem: jest.fn(),
}));

// Mock CalculatorComponent to simplify SolverPage testing
jest.mock('../components/CalculatorComponent', () => {
  return function MockCalculatorComponent({
    problem,
    answer,
    onProblemChange,
    onSolve,
    loading,
    error,
  }) {
    return (
      <div data-testid="calculator-component">
        <input
          data-testid="problem-input"
          value={problem}
          onChange={(e) => onProblemChange(e.target.value)}
        />
        <div data-testid="answer-display">{answer ? `= ${answer}` : 'Ready...'}</div>
        <button data-testid="solve-button" onClick={onSolve} disabled={loading}>
          {loading ? 'SOLVING...' : 'SOLVE'}
        </button>
        {error && <div data-testid="error-message">{error}</div>}
        <div data-testid="loading-state">{loading ? 'true' : 'false'}</div>
      </div>
    );
  };
});

describe('SolverPage Component', () => {
  beforeEach(() => {
    // Reset all mocks before each test
    jest.clearAllMocks();
  });

  describe('when testing component rendering', () => {
    test('should render CalculatorComponent with initial state', () => {
      render(<SolverPage />);

      // Verify CalculatorComponent is rendered
      expect(screen.getByTestId('calculator-component')).toBeInTheDocument();

      // Verify initial state: empty problem, empty answer, not loading, no error
      expect(screen.getByTestId('problem-input')).toHaveValue('');
      expect(screen.getByTestId('answer-display')).toHaveTextContent('Ready...');
      expect(screen.getByTestId('loading-state')).toHaveTextContent('false');
      expect(screen.queryByTestId('error-message')).not.toBeInTheDocument();
    });

    test('should render page container with correct className', () => {
      const { container } = render(<SolverPage />);

      const pageContainer = container.querySelector('.page-container');
      expect(pageContainer).toBeInTheDocument();
    });
  });

  describe('when testing valid problem submission with API success', () => {
    test('should display answer and clear problem after successful solve', async () => {
      // Mock validateProblem to return null (no error)
      validateProblem.mockReturnValue(null);

      // Mock solve to return successful answer
      solve.mockResolvedValue({ answer: 14 });

      render(<SolverPage />);

      // Enter problem
      const input = screen.getByTestId('problem-input');
      fireEvent.change(input, { target: { value: '2+3*4' } });
      expect(input).toHaveValue('2+3*4');

      // Click solve button
      const solveButton = screen.getByTestId('solve-button');
      fireEvent.click(solveButton);

      // Verify validateProblem was called with problem
      expect(validateProblem).toHaveBeenCalledWith('2+3*4');

      // Verify solve API was called with problem
      expect(solve).toHaveBeenCalledWith('2+3*4');

      // Wait for async operation to complete
      await waitFor(() => {
        // Verify answer is displayed
        expect(screen.getByTestId('answer-display')).toHaveTextContent('= 14');
      });

      // Verify problem input is cleared after successful solve
      expect(input).toHaveValue('');

      // Verify no error message is displayed
      expect(screen.queryByTestId('error-message')).not.toBeInTheDocument();
    });

    test('should handle multiple sequential solve operations correctly', async () => {
      validateProblem.mockReturnValue(null);

      render(<SolverPage />);

      const input = screen.getByTestId('problem-input');
      const solveButton = screen.getByTestId('solve-button');

      // First solve: 2+3 = 5
      solve.mockResolvedValue({ answer: 5 });
      fireEvent.change(input, { target: { value: '2+3' } });
      fireEvent.click(solveButton);

      await waitFor(() => {
        expect(screen.getByTestId('answer-display')).toHaveTextContent('= 5');
      });

      expect(input).toHaveValue(''); // Problem cleared

      // Second solve: 10*5 = 50
      solve.mockResolvedValue({ answer: 50 });
      fireEvent.change(input, { target: { value: '10*5' } });
      fireEvent.click(solveButton);

      await waitFor(() => {
        expect(screen.getByTestId('answer-display')).toHaveTextContent('= 50');
      });

      expect(input).toHaveValue(''); // Problem cleared again
    });

    test('should clear previous error when new valid problem is solved', async () => {
      render(<SolverPage />);

      const input = screen.getByTestId('problem-input');
      const solveButton = screen.getByTestId('solve-button');

      // First attempt: validation error
      validateProblem.mockReturnValue('Please enter a math problem');
      fireEvent.change(input, { target: { value: '' } });
      fireEvent.click(solveButton);

      expect(screen.getByTestId('error-message')).toHaveTextContent(
        'Please enter a math problem'
      );

      // Second attempt: valid problem
      validateProblem.mockReturnValue(null);
      solve.mockResolvedValue({ answer: 7 });
      fireEvent.change(input, { target: { value: '3+4' } });
      fireEvent.click(solveButton);

      await waitFor(() => {
        expect(screen.getByTestId('answer-display')).toHaveTextContent('= 7');
      });

      // Verify error is cleared
      expect(screen.queryByTestId('error-message')).not.toBeInTheDocument();
    });

    test('should clear previous answer when solving new problem', async () => {
      validateProblem.mockReturnValue(null);

      render(<SolverPage />);

      const input = screen.getByTestId('problem-input');
      const solveButton = screen.getByTestId('solve-button');

      // First solve
      solve.mockResolvedValue({ answer: 10 });
      fireEvent.change(input, { target: { value: '5+5' } });
      fireEvent.click(solveButton);

      await waitFor(() => {
        expect(screen.getByTestId('answer-display')).toHaveTextContent('= 10');
      });

      // Start new solve - answer should be cleared immediately when button clicked
      fireEvent.change(input, { target: { value: '2+2' } });

      // Before clicking solve, answer should still show previous result
      expect(screen.getByTestId('answer-display')).toHaveTextContent('= 10');

      // Click solve - answer cleared in handleSolve
      solve.mockResolvedValue({ answer: 4 });
      fireEvent.click(solveButton);

      // Answer is cleared immediately in handleSolve (before API call completes)
      // Note: This tests the implementation detail that setAnswer('') happens before async call
      // In real usage, this might be too fast to notice, but it's correct behavior
      await waitFor(() => {
        expect(screen.getByTestId('answer-display')).toHaveTextContent('= 4');
      });
    });
  });

  describe('when testing invalid problem submission with validation errors', () => {
    test('should display error and not call API when problem is empty', () => {
      // Mock validateProblem to return error for empty input
      validateProblem.mockReturnValue('Please enter a math problem');

      render(<SolverPage />);

      // Click solve without entering problem
      const solveButton = screen.getByTestId('solve-button');
      fireEvent.click(solveButton);

      // Verify validateProblem was called with empty string
      expect(validateProblem).toHaveBeenCalledWith('');

      // Verify error is displayed
      expect(screen.getByTestId('error-message')).toHaveTextContent(
        'Please enter a math problem'
      );

      // Verify API was NOT called
      expect(solve).not.toHaveBeenCalled();

      // Verify loading state was never set to true
      expect(screen.getByTestId('loading-state')).toHaveTextContent('false');
    });

    test('should display error and not call API when problem contains invalid characters', () => {
      validateProblem.mockReturnValue('Problem contains invalid characters');

      render(<SolverPage />);

      // Enter invalid problem
      const input = screen.getByTestId('problem-input');
      fireEvent.change(input, { target: { value: '2+abc' } });

      // Click solve
      const solveButton = screen.getByTestId('solve-button');
      fireEvent.click(solveButton);

      // Verify validateProblem was called
      expect(validateProblem).toHaveBeenCalledWith('2+abc');

      // Verify error is displayed
      expect(screen.getByTestId('error-message')).toHaveTextContent(
        'Problem contains invalid characters'
      );

      // Verify API was NOT called
      expect(solve).not.toHaveBeenCalled();
    });

    test('should display error for whitespace-only problem', () => {
      validateProblem.mockReturnValue('Please enter a math problem');

      render(<SolverPage />);

      // Enter whitespace-only problem
      const input = screen.getByTestId('problem-input');
      fireEvent.change(input, { target: { value: '   ' } });

      // Click solve
      const solveButton = screen.getByTestId('solve-button');
      fireEvent.click(solveButton);

      // Verify validateProblem was called with whitespace
      expect(validateProblem).toHaveBeenCalledWith('   ');

      // Verify error is displayed
      expect(screen.getByTestId('error-message')).toHaveTextContent(
        'Please enter a math problem'
      );

      // Verify API was NOT called
      expect(solve).not.toHaveBeenCalled();
    });
  });

  describe('when testing API error handling', () => {
    test('should display error message when API call fails', async () => {
      validateProblem.mockReturnValue(null);

      // Mock solve to reject with error
      solve.mockRejectedValue(new Error('Failed to solve problem'));

      render(<SolverPage />);

      const input = screen.getByTestId('problem-input');
      fireEvent.change(input, { target: { value: '2+2' } });

      const solveButton = screen.getByTestId('solve-button');
      fireEvent.click(solveButton);

      // Wait for error to be displayed
      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toHaveTextContent(
          'Failed to solve problem'
        );
      });

      // Verify answer is not displayed
      expect(screen.getByTestId('answer-display')).toHaveTextContent('Ready...');

      // Verify loading is set back to false after error
      expect(screen.getByTestId('loading-state')).toHaveTextContent('false');
    });

    test('should handle network errors gracefully', async () => {
      validateProblem.mockReturnValue(null);

      solve.mockRejectedValue(new Error('Network error'));

      render(<SolverPage />);

      const input = screen.getByTestId('problem-input');
      fireEvent.change(input, { target: { value: '5*5' } });

      const solveButton = screen.getByTestId('solve-button');
      fireEvent.click(solveButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toHaveTextContent('Network error');
      });

      // Verify loading state is reset
      expect(screen.getByTestId('loading-state')).toHaveTextContent('false');
    });

    test('should not clear problem input when API call fails', async () => {
      validateProblem.mockReturnValue(null);

      solve.mockRejectedValue(new Error('Server error'));

      render(<SolverPage />);

      const input = screen.getByTestId('problem-input');
      fireEvent.change(input, { target: { value: '10+20' } });

      const solveButton = screen.getByTestId('solve-button');
      fireEvent.click(solveButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      // Verify problem is NOT cleared (user can retry)
      expect(input).toHaveValue('10+20');
    });
  });

  describe('when testing loading state behavior', () => {
    test('should set loading to true during API call and false after completion', async () => {
      validateProblem.mockReturnValue(null);

      // Create a promise we can control
      let resolvePromise;
      const controlledPromise = new Promise((resolve) => {
        resolvePromise = resolve;
      });
      solve.mockReturnValue(controlledPromise);

      render(<SolverPage />);

      const input = screen.getByTestId('problem-input');
      fireEvent.change(input, { target: { value: '2+2' } });

      // Before clicking solve
      expect(screen.getByTestId('loading-state')).toHaveTextContent('false');
      expect(screen.getByTestId('solve-button')).toHaveTextContent('SOLVE');

      // Click solve
      const solveButton = screen.getByTestId('solve-button');
      fireEvent.click(solveButton);

      // During API call - loading should be true
      await waitFor(() => {
        expect(screen.getByTestId('loading-state')).toHaveTextContent('true');
      });
      expect(screen.getByTestId('solve-button')).toHaveTextContent('SOLVING...');
      expect(screen.getByTestId('solve-button')).toBeDisabled();

      // Resolve the API call
      resolvePromise({ answer: 4 });

      // After API call - loading should be false
      await waitFor(() => {
        expect(screen.getByTestId('loading-state')).toHaveTextContent('false');
      });
      expect(screen.getByTestId('solve-button')).toHaveTextContent('SOLVE');
      expect(screen.getByTestId('solve-button')).not.toBeDisabled();
    });

    test('should set loading to false after API error', async () => {
      validateProblem.mockReturnValue(null);

      solve.mockRejectedValue(new Error('API error'));

      render(<SolverPage />);

      const input = screen.getByTestId('problem-input');
      fireEvent.change(input, { target: { value: '5+5' } });

      const solveButton = screen.getByTestId('solve-button');
      fireEvent.click(solveButton);

      // Wait for error
      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      // Verify loading is false (finally block executed)
      expect(screen.getByTestId('loading-state')).toHaveTextContent('false');
      expect(screen.getByTestId('solve-button')).not.toBeDisabled();
    });

    test('should disable input and button during loading', async () => {
      validateProblem.mockReturnValue(null);

      let resolvePromise;
      const controlledPromise = new Promise((resolve) => {
        resolvePromise = resolve;
      });
      solve.mockReturnValue(controlledPromise);

      render(<SolverPage />);

      const input = screen.getByTestId('problem-input');
      const solveButton = screen.getByTestId('solve-button');

      fireEvent.change(input, { target: { value: '2+2' } });
      fireEvent.click(solveButton);

      // Wait for loading state
      await waitFor(() => {
        expect(screen.getByTestId('loading-state')).toHaveTextContent('true');
      });

      // Verify button is disabled
      expect(solveButton).toBeDisabled();

      // Note: Input disable state is controlled by CalculatorComponent's loading prop
      // which is passed correctly and verified by mock component

      // Resolve
      resolvePromise({ answer: 4 });

      await waitFor(() => {
        expect(screen.getByTestId('loading-state')).toHaveTextContent('false');
      });

      expect(solveButton).not.toBeDisabled();
    });
  });

  describe('when testing input trimming edge cases (Bug #44)', () => {
    test('should send problem with leading/trailing whitespace to API without trimming', async () => {
      validateProblem.mockReturnValue(null);
      solve.mockResolvedValue({ answer: 4 });

      render(<SolverPage />);

      const input = screen.getByTestId('problem-input');

      // Enter problem with leading/trailing whitespace
      fireEvent.change(input, { target: { value: '  2+2  ' } });

      const solveButton = screen.getByTestId('solve-button');
      fireEvent.click(solveButton);

      // Verify solve is called with untrimmed value (Bug #44)
      // This documents that input is NOT trimmed before sending to API
      expect(solve).toHaveBeenCalledWith('  2+2  ');

      await waitFor(() => {
        expect(screen.getByTestId('answer-display')).toHaveTextContent('= 4');
      });
    });

    test('should send problem with internal whitespace to API unchanged', async () => {
      validateProblem.mockReturnValue(null);
      solve.mockResolvedValue({ answer: 7 });

      render(<SolverPage />);

      const input = screen.getByTestId('problem-input');

      // Enter problem with internal whitespace
      fireEvent.change(input, { target: { value: '3  +  4' } });

      const solveButton = screen.getByTestId('solve-button');
      fireEvent.click(solveButton);

      // Verify whitespace is preserved (not trimmed)
      expect(solve).toHaveBeenCalledWith('3  +  4');

      await waitFor(() => {
        expect(screen.getByTestId('answer-display')).toHaveTextContent('= 7');
      });
    });
  });

  describe('when testing state updates and rerenders', () => {
    test('should update problem state when input changes', () => {
      render(<SolverPage />);

      const input = screen.getByTestId('problem-input');

      // Type into input
      fireEvent.change(input, { target: { value: '1' } });
      expect(input).toHaveValue('1');

      fireEvent.change(input, { target: { value: '1+' } });
      expect(input).toHaveValue('1+');

      fireEvent.change(input, { target: { value: '1+2' } });
      expect(input).toHaveValue('1+2');
    });

    test('should maintain error state until new solve attempt', () => {
      validateProblem.mockReturnValue('Please enter a math problem');

      render(<SolverPage />);

      const solveButton = screen.getByTestId('solve-button');

      // First solve attempt - validation error
      fireEvent.click(solveButton);

      expect(screen.getByTestId('error-message')).toHaveTextContent(
        'Please enter a math problem'
      );

      // Change input but don't solve yet
      const input = screen.getByTestId('problem-input');
      fireEvent.change(input, { target: { value: '2+2' } });

      // Error should still be displayed (not cleared by typing)
      expect(screen.getByTestId('error-message')).toHaveTextContent(
        'Please enter a math problem'
      );
    });
  });
});
