// Component tests for HistoryPage
// Tests UI rendering based on props using React Testing Library

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import HistoryPage from '../components/HistoryPage';
import * as api from '../api';

// Mock the API module
jest.mock('../api');

describe('HistoryPage Component Tests', () => {
  test('renders a list based on mock history prop', () => {
    const mockHistory = [
      { id: 1, problem: '1+1', answer: '2', created_at: '2024-01-01T00:00:00Z' },
      { id: 2, problem: '5*10', answer: '50', created_at: '2024-01-02T00:00:00Z' },
      { id: 3, problem: '100-25', answer: '75', created_at: '2024-01-03T00:00:00Z' },
    ];

    render(<HistoryPage mockHistory={mockHistory} />);

    const historyItems = screen.getAllByTestId('history-item');
    expect(historyItems).toHaveLength(3);

    expect(screen.getByText('1+1 = 2')).toBeInTheDocument();
    expect(screen.getByText('5*10 = 50')).toBeInTheDocument();
    expect(screen.getByText('100-25 = 75')).toBeInTheDocument();
  });

  test('renders empty state when no history items', () => {
    render(<HistoryPage mockHistory={[]} />);

    expect(screen.getByText(/No history yet/i)).toBeInTheDocument();
    expect(screen.queryByTestId('history-item')).not.toBeInTheDocument();
  });

  test('renders correct number of items for different array sizes', () => {
    const oneItem = [
      { id: 1, problem: '2+2', answer: '4', created_at: '2024-01-01T00:00:00Z' },
    ];

    const { rerender } = render(<HistoryPage mockHistory={oneItem} />);
    expect(screen.getAllByTestId('history-item')).toHaveLength(1);

    const fiveItems = [
      { id: 1, problem: '1+1', answer: '2', created_at: '2024-01-01T00:00:00Z' },
      { id: 2, problem: '2+2', answer: '4', created_at: '2024-01-01T00:00:00Z' },
      { id: 3, problem: '3+3', answer: '6', created_at: '2024-01-01T00:00:00Z' },
      { id: 4, problem: '4+4', answer: '8', created_at: '2024-01-01T00:00:00Z' },
      { id: 5, problem: '5+5', answer: '10', created_at: '2024-01-01T00:00:00Z' },
    ];

    rerender(<HistoryPage mockHistory={fiveItems} />);
    expect(screen.getAllByTestId('history-item')).toHaveLength(5);
  });

  test('displays refresh button when history exists', () => {
    const mockHistory = [
      { id: 1, problem: '10+10', answer: '20', created_at: '2024-01-01T00:00:00Z' },
    ];

    render(<HistoryPage mockHistory={mockHistory} />);
    expect(screen.getByTestId('refresh-button')).toBeInTheDocument();
  });

  test('does not display refresh button when history is empty', () => {
    render(<HistoryPage mockHistory={[]} />);
    expect(screen.queryByTestId('refresh-button')).not.toBeInTheDocument();
  });
});

describe('HistoryPage API Functionality Tests (without mockHistory prop)', () => {
  beforeEach(() => {
    // Clear all mocks before each test
    jest.clearAllMocks();
  });

  describe('Loading state', () => {
    test('displays loading state while fetching history from API', async () => {
      // Arrange - Mock getHistory to return a promise that we control
      let resolveHistory;
      const historyPromise = new Promise((resolve) => {
        resolveHistory = resolve;
      });
      api.getHistory.mockReturnValue(historyPromise);

      // Act - Render without mockHistory prop
      render(<HistoryPage />);

      // Assert - Loading message should be visible while fetch is in progress
      expect(screen.getByText('Loading history...')).toBeInTheDocument();
      expect(screen.queryByTestId('history-item')).not.toBeInTheDocument();

      // Clean up - Resolve the promise to avoid warnings
      resolveHistory([]);
      await waitFor(() => {
        expect(screen.queryByText('Loading history...')).not.toBeInTheDocument();
      });
    });

    test('hides loading state after successful fetch', async () => {
      // Arrange
      const mockData = [
        { id: 1, problem: '2+2', answer: '4', created_at: '2024-01-01T00:00:00Z' },
      ];
      api.getHistory.mockResolvedValue(mockData);

      // Act
      render(<HistoryPage />);

      // Assert - Loading should disappear after fetch completes
      await waitFor(() => {
        expect(screen.queryByText('Loading history...')).not.toBeInTheDocument();
      });
      expect(screen.getByText('2+2 = 4')).toBeInTheDocument();
    });

    test('hides loading state after fetch error', async () => {
      // Arrange
      api.getHistory.mockRejectedValue(new Error('Network error'));

      // Act
      render(<HistoryPage />);

      // Assert - Loading should disappear even when fetch fails
      await waitFor(() => {
        expect(screen.queryByText('Loading history...')).not.toBeInTheDocument();
      });
      expect(screen.getByText('Network error')).toBeInTheDocument();
    });
  });

  describe('Successful API fetch', () => {
    test('calls getHistory API on mount', async () => {
      // Arrange
      api.getHistory.mockResolvedValue([]);

      // Act
      render(<HistoryPage />);

      // Assert - Verify getHistory was called
      await waitFor(() => {
        expect(api.getHistory).toHaveBeenCalledTimes(1);
      });
    });

    test('displays fetched history items', async () => {
      // Arrange
      const mockData = [
        { id: 1, problem: '5+5', answer: '10', created_at: '2024-01-01T00:00:00Z' },
        { id: 2, problem: '10*2', answer: '20', created_at: '2024-01-02T00:00:00Z' },
      ];
      api.getHistory.mockResolvedValue(mockData);

      // Act
      render(<HistoryPage />);

      // Assert - History items should be displayed
      await waitFor(() => {
        expect(screen.getByText('5+5 = 10')).toBeInTheDocument();
      });
      expect(screen.getByText('10*2 = 20')).toBeInTheDocument();
      expect(screen.getAllByTestId('history-item')).toHaveLength(2);
    });

    test('displays refresh button after successful fetch with data', async () => {
      // Arrange
      const mockData = [
        { id: 1, problem: '3+3', answer: '6', created_at: '2024-01-01T00:00:00Z' },
      ];
      api.getHistory.mockResolvedValue(mockData);

      // Act
      render(<HistoryPage />);

      // Assert - Refresh button should appear
      await waitFor(() => {
        expect(screen.getByTestId('refresh-button')).toBeInTheDocument();
      });
    });
  });

  describe('Empty history response', () => {
    test('displays "No history yet" message when API returns empty array', async () => {
      // Arrange
      api.getHistory.mockResolvedValue([]);

      // Act
      render(<HistoryPage />);

      // Assert - Empty state message should be displayed
      await waitFor(() => {
        expect(screen.getByText(/No history yet/i)).toBeInTheDocument();
      });
      expect(screen.queryByTestId('history-item')).not.toBeInTheDocument();
      expect(screen.queryByTestId('refresh-button')).not.toBeInTheDocument();
    });
  });

  describe('Error handling', () => {
    test('displays error message when API call fails', async () => {
      // Arrange
      const errorMessage = 'Failed to fetch history';
      api.getHistory.mockRejectedValue(new Error(errorMessage));

      // Act
      render(<HistoryPage />);

      // Assert - Error message should be displayed
      await waitFor(() => {
        expect(screen.getByText(errorMessage)).toBeInTheDocument();
      });
    });

    test('displays network error message', async () => {
      // Arrange
      api.getHistory.mockRejectedValue(new Error('Network request failed'));

      // Act
      render(<HistoryPage />);

      // Assert - Network error should be displayed
      await waitFor(() => {
        expect(screen.getByText('Network request failed')).toBeInTheDocument();
      });
    });

    test('does not display history items when fetch fails', async () => {
      // Arrange
      api.getHistory.mockRejectedValue(new Error('Server error'));

      // Act
      render(<HistoryPage />);

      // Assert - No history items should be displayed on error
      await waitFor(() => {
        expect(screen.getByText('Server error')).toBeInTheDocument();
      });
      expect(screen.queryByTestId('history-item')).not.toBeInTheDocument();
    });
  });

  describe('useEffect behavior', () => {
    test('does not call API when mockHistory prop is provided', () => {
      // Arrange
      const mockHistory = [
        { id: 1, problem: '1+1', answer: '2', created_at: '2024-01-01T00:00:00Z' },
      ];

      // Act
      render(<HistoryPage mockHistory={mockHistory} />);

      // Assert - getHistory should NOT be called when mockHistory is provided
      expect(api.getHistory).not.toHaveBeenCalled();
    });

    test('calls API immediately on mount when no mockHistory prop', async () => {
      // Arrange
      api.getHistory.mockResolvedValue([]);

      // Act
      render(<HistoryPage />);

      // Assert - getHistory should be called on mount
      await waitFor(() => {
        expect(api.getHistory).toHaveBeenCalledTimes(1);
      });
    });
  });
});
