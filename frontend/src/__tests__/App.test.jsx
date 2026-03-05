// Comprehensive unit tests for App.jsx
// Tests authentication state management, routing logic, and user interactions
// Covers: initialization, login/register flows, logout, navigation, route protection

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import App from '../App';
import { isAuthenticated, clearToken } from '../api';

// Mock the API module
jest.mock('../api', () => ({
  isAuthenticated: jest.fn(),
  clearToken: jest.fn(),
}));

// Mock all page components to simplify testing
jest.mock('../components/LoginPage', () => {
  return function MockLoginPage({ onLoginSuccess }) {
    return (
      <div data-testid="login-page">
        <button data-testid="mock-login-submit" onClick={onLoginSuccess}>
          Submit Login
        </button>
      </div>
    );
  };
});

jest.mock('../components/RegisterPage', () => {
  return function MockRegisterPage({ onRegisterSuccess }) {
    return (
      <div data-testid="register-page">
        <button data-testid="mock-register-submit" onClick={onRegisterSuccess}>
          Submit Register
        </button>
      </div>
    );
  };
});

jest.mock('../components/SolverPage', () => {
  return function MockSolverPage() {
    return <div data-testid="solver-page">Solver Page</div>;
  };
});

jest.mock('../components/HistoryPage', () => {
  return function MockHistoryPage() {
    return <div data-testid="history-page">History Page</div>;
  };
});

describe('App Component - Authentication and Routing', () => {
  beforeEach(() => {
    // Reset all mocks before each test
    jest.clearAllMocks();
  });

  describe('when testing authentication state initialization on mount', () => {
    test('should start authenticated on solve page when token exists in localStorage', () => {
      // Mock isAuthenticated to return true (user has valid token)
      isAuthenticated.mockReturnValue(true);

      render(<App />);

      // Verify isAuthenticated was called on mount
      expect(isAuthenticated).toHaveBeenCalledTimes(1);

      // Verify app renders solver page (default authenticated landing page)
      expect(screen.getByTestId('solver-page')).toBeInTheDocument();

      // Verify authenticated navigation buttons are shown
      expect(screen.getByText('SOLVER')).toBeInTheDocument();
      expect(screen.getByText('HISTORY')).toBeInTheDocument();
      expect(screen.getByText('LOGOUT')).toBeInTheDocument();

      // Verify unauthenticated buttons are NOT shown
      expect(screen.queryByText('LOGIN')).not.toBeInTheDocument();
      expect(screen.queryByText('REGISTER')).not.toBeInTheDocument();
    });

    test('should start unauthenticated on login page when no token exists', () => {
      // Mock isAuthenticated to return false (no token)
      isAuthenticated.mockReturnValue(false);

      render(<App />);

      // Verify isAuthenticated was called on mount
      expect(isAuthenticated).toHaveBeenCalledTimes(1);

      // Verify app renders login page (default unauthenticated page)
      expect(screen.getByTestId('login-page')).toBeInTheDocument();

      // Verify unauthenticated navigation buttons are shown
      expect(screen.getByText('LOGIN')).toBeInTheDocument();
      expect(screen.getByText('REGISTER')).toBeInTheDocument();

      // Verify authenticated buttons are NOT shown
      expect(screen.queryByText('SOLVER')).not.toBeInTheDocument();
      expect(screen.queryByText('HISTORY')).not.toBeInTheDocument();
      expect(screen.queryByText('LOGOUT')).not.toBeInTheDocument();
    });
  });

  describe('when testing login success flow', () => {
    test('should update state to authenticated and navigate to solve page on login success', () => {
      // Start unauthenticated
      isAuthenticated.mockReturnValue(false);

      render(<App />);

      // Verify starts on login page
      expect(screen.getByTestId('login-page')).toBeInTheDocument();

      // Simulate successful login by clicking mock login submit button
      const mockLoginSubmit = screen.getByTestId('mock-login-submit');
      fireEvent.click(mockLoginSubmit);

      // Verify app navigates to solver page
      expect(screen.getByTestId('solver-page')).toBeInTheDocument();
      expect(screen.queryByTestId('login-page')).not.toBeInTheDocument();

      // Verify authenticated navigation is now shown
      expect(screen.getByText('SOLVER')).toBeInTheDocument();
      expect(screen.getByText('HISTORY')).toBeInTheDocument();
      expect(screen.getByText('LOGOUT')).toBeInTheDocument();
    });
  });

  describe('when testing registration success flow', () => {
    test('should update state to authenticated and navigate to solve page on registration success', () => {
      // Start unauthenticated
      isAuthenticated.mockReturnValue(false);

      render(<App />);

      // Navigate to register page
      const registerButton = screen.getByText('REGISTER');
      fireEvent.click(registerButton);

      // Verify register page is shown
      expect(screen.getByTestId('register-page')).toBeInTheDocument();

      // Simulate successful registration by clicking mock register submit button
      const mockRegisterSubmit = screen.getByTestId('mock-register-submit');
      fireEvent.click(mockRegisterSubmit);

      // Verify app navigates to solver page (new users immediately authenticated)
      expect(screen.getByTestId('solver-page')).toBeInTheDocument();
      expect(screen.queryByTestId('register-page')).not.toBeInTheDocument();

      // Verify authenticated navigation is now shown
      expect(screen.getByText('SOLVER')).toBeInTheDocument();
      expect(screen.getByText('HISTORY')).toBeInTheDocument();
      expect(screen.getByText('LOGOUT')).toBeInTheDocument();
    });
  });

  describe('when testing logout flow', () => {
    test('should clear token, reset state, and navigate to login page on logout', () => {
      // Start authenticated
      isAuthenticated.mockReturnValue(true);

      render(<App />);

      // Verify starts on solver page (authenticated)
      expect(screen.getByTestId('solver-page')).toBeInTheDocument();

      // Click logout button
      const logoutButton = screen.getByText('LOGOUT');
      fireEvent.click(logoutButton);

      // Verify clearToken was called
      expect(clearToken).toHaveBeenCalledTimes(1);

      // Verify app navigates to login page
      expect(screen.getByTestId('login-page')).toBeInTheDocument();
      expect(screen.queryByTestId('solver-page')).not.toBeInTheDocument();

      // Verify unauthenticated navigation is now shown
      expect(screen.getByText('LOGIN')).toBeInTheDocument();
      expect(screen.getByText('REGISTER')).toBeInTheDocument();

      // Verify authenticated buttons are gone
      expect(screen.queryByText('SOLVER')).not.toBeInTheDocument();
      expect(screen.queryByText('HISTORY')).not.toBeInTheDocument();
      expect(screen.queryByText('LOGOUT')).not.toBeInTheDocument();
    });
  });

  describe('when testing page navigation for authenticated users', () => {
    test('should navigate from solver to history page when HISTORY button clicked', () => {
      // Start authenticated on solver page
      isAuthenticated.mockReturnValue(true);

      render(<App />);

      // Verify starts on solver page
      expect(screen.getByTestId('solver-page')).toBeInTheDocument();

      // Click HISTORY button
      const historyButton = screen.getByText('HISTORY');
      fireEvent.click(historyButton);

      // Verify navigates to history page
      expect(screen.getByTestId('history-page')).toBeInTheDocument();
      expect(screen.queryByTestId('solver-page')).not.toBeInTheDocument();
    });

    test('should navigate from history back to solver page when SOLVER button clicked', () => {
      // Start authenticated on solver page
      isAuthenticated.mockReturnValue(true);

      render(<App />);

      // Navigate to history
      const historyButton = screen.getByText('HISTORY');
      fireEvent.click(historyButton);
      expect(screen.getByTestId('history-page')).toBeInTheDocument();

      // Navigate back to solver
      const solverButton = screen.getByText('SOLVER');
      fireEvent.click(solverButton);

      // Verify back on solver page
      expect(screen.getByTestId('solver-page')).toBeInTheDocument();
      expect(screen.queryByTestId('history-page')).not.toBeInTheDocument();
    });

    test('should allow multiple navigation cycles between solver and history', () => {
      // Start authenticated
      isAuthenticated.mockReturnValue(true);

      render(<App />);

      // Cycle 1: Solver → History
      fireEvent.click(screen.getByText('HISTORY'));
      expect(screen.getByTestId('history-page')).toBeInTheDocument();

      // Cycle 2: History → Solver
      fireEvent.click(screen.getByText('SOLVER'));
      expect(screen.getByTestId('solver-page')).toBeInTheDocument();

      // Cycle 3: Solver → History
      fireEvent.click(screen.getByText('HISTORY'));
      expect(screen.getByTestId('history-page')).toBeInTheDocument();

      // Cycle 4: History → Solver
      fireEvent.click(screen.getByText('SOLVER'));
      expect(screen.getByTestId('solver-page')).toBeInTheDocument();
    });
  });

  describe('when testing page navigation for unauthenticated users', () => {
    test('should navigate from login to register page when REGISTER button clicked', () => {
      // Start unauthenticated on login page
      isAuthenticated.mockReturnValue(false);

      render(<App />);

      // Verify starts on login page
      expect(screen.getByTestId('login-page')).toBeInTheDocument();

      // Click REGISTER button
      const registerButton = screen.getByText('REGISTER');
      fireEvent.click(registerButton);

      // Verify navigates to register page
      expect(screen.getByTestId('register-page')).toBeInTheDocument();
      expect(screen.queryByTestId('login-page')).not.toBeInTheDocument();
    });

    test('should navigate from register back to login page when LOGIN button clicked', () => {
      // Start unauthenticated
      isAuthenticated.mockReturnValue(false);

      render(<App />);

      // Navigate to register
      fireEvent.click(screen.getByText('REGISTER'));
      expect(screen.getByTestId('register-page')).toBeInTheDocument();

      // Navigate back to login
      fireEvent.click(screen.getByText('LOGIN'));

      // Verify back on login page
      expect(screen.getByTestId('login-page')).toBeInTheDocument();
      expect(screen.queryByTestId('register-page')).not.toBeInTheDocument();
    });

    test('should allow multiple navigation cycles between login and register', () => {
      // Start unauthenticated
      isAuthenticated.mockReturnValue(false);

      render(<App />);

      // Cycle 1: Login → Register
      fireEvent.click(screen.getByText('REGISTER'));
      expect(screen.getByTestId('register-page')).toBeInTheDocument();

      // Cycle 2: Register → Login
      fireEvent.click(screen.getByText('LOGIN'));
      expect(screen.getByTestId('login-page')).toBeInTheDocument();

      // Cycle 3: Login → Register
      fireEvent.click(screen.getByText('REGISTER'));
      expect(screen.getByTestId('register-page')).toBeInTheDocument();

      // Cycle 4: Register → Login
      fireEvent.click(screen.getByText('LOGIN'));
      expect(screen.getByTestId('login-page')).toBeInTheDocument();
    });
  });

  describe('when testing conditional navigation rendering', () => {
    test('should only show unauthenticated navigation when not authenticated', () => {
      isAuthenticated.mockReturnValue(false);

      render(<App />);

      // Verify unauthenticated buttons present
      expect(screen.getByText('LOGIN')).toBeInTheDocument();
      expect(screen.getByText('REGISTER')).toBeInTheDocument();

      // Verify authenticated buttons absent
      expect(screen.queryByText('SOLVER')).not.toBeInTheDocument();
      expect(screen.queryByText('HISTORY')).not.toBeInTheDocument();
      expect(screen.queryByText('LOGOUT')).not.toBeInTheDocument();
    });

    test('should only show authenticated navigation when authenticated', () => {
      isAuthenticated.mockReturnValue(true);

      render(<App />);

      // Verify authenticated buttons present
      expect(screen.getByText('SOLVER')).toBeInTheDocument();
      expect(screen.getByText('HISTORY')).toBeInTheDocument();
      expect(screen.getByText('LOGOUT')).toBeInTheDocument();

      // Verify unauthenticated buttons absent
      expect(screen.queryByText('LOGIN')).not.toBeInTheDocument();
      expect(screen.queryByText('REGISTER')).not.toBeInTheDocument();
    });

    test('should switch navigation buttons after login', () => {
      isAuthenticated.mockReturnValue(false);

      render(<App />);

      // Initially unauthenticated navigation
      expect(screen.getByText('LOGIN')).toBeInTheDocument();
      expect(screen.getByText('REGISTER')).toBeInTheDocument();

      // Login
      fireEvent.click(screen.getByTestId('mock-login-submit'));

      // Now authenticated navigation
      expect(screen.getByText('SOLVER')).toBeInTheDocument();
      expect(screen.getByText('HISTORY')).toBeInTheDocument();
      expect(screen.getByText('LOGOUT')).toBeInTheDocument();
      expect(screen.queryByText('LOGIN')).not.toBeInTheDocument();
      expect(screen.queryByText('REGISTER')).not.toBeInTheDocument();
    });

    test('should switch navigation buttons after logout', () => {
      isAuthenticated.mockReturnValue(true);

      render(<App />);

      // Initially authenticated navigation
      expect(screen.getByText('SOLVER')).toBeInTheDocument();
      expect(screen.getByText('LOGOUT')).toBeInTheDocument();

      // Logout
      fireEvent.click(screen.getByText('LOGOUT'));

      // Now unauthenticated navigation
      expect(screen.getByText('LOGIN')).toBeInTheDocument();
      expect(screen.getByText('REGISTER')).toBeInTheDocument();
      expect(screen.queryByText('SOLVER')).not.toBeInTheDocument();
      expect(screen.queryByText('LOGOUT')).not.toBeInTheDocument();
    });
  });

  describe('when testing complete user journey flows', () => {
    test('should handle full unauthenticated → register → authenticated → logout → unauthenticated flow', () => {
      // Start unauthenticated
      isAuthenticated.mockReturnValue(false);

      render(<App />);

      // Step 1: Verify unauthenticated state
      expect(screen.getByTestId('login-page')).toBeInTheDocument();
      expect(screen.getByText('LOGIN')).toBeInTheDocument();

      // Step 2: Navigate to register
      fireEvent.click(screen.getByText('REGISTER'));
      expect(screen.getByTestId('register-page')).toBeInTheDocument();

      // Step 3: Complete registration
      fireEvent.click(screen.getByTestId('mock-register-submit'));
      expect(screen.getByTestId('solver-page')).toBeInTheDocument();

      // Step 4: Navigate to history
      fireEvent.click(screen.getByText('HISTORY'));
      expect(screen.getByTestId('history-page')).toBeInTheDocument();

      // Step 5: Logout
      fireEvent.click(screen.getByText('LOGOUT'));
      expect(clearToken).toHaveBeenCalled();
      expect(screen.getByTestId('login-page')).toBeInTheDocument();
      expect(screen.getByText('LOGIN')).toBeInTheDocument();
    });

    test('should handle full unauthenticated → login → solve → history → logout flow', () => {
      // Start unauthenticated
      isAuthenticated.mockReturnValue(false);

      render(<App />);

      // Step 1: Start on login page
      expect(screen.getByTestId('login-page')).toBeInTheDocument();

      // Step 2: Login
      fireEvent.click(screen.getByTestId('mock-login-submit'));
      expect(screen.getByTestId('solver-page')).toBeInTheDocument();

      // Step 3: Navigate to history
      fireEvent.click(screen.getByText('HISTORY'));
      expect(screen.getByTestId('history-page')).toBeInTheDocument();

      // Step 4: Navigate back to solver
      fireEvent.click(screen.getByText('SOLVER'));
      expect(screen.getByTestId('solver-page')).toBeInTheDocument();

      // Step 5: Logout
      fireEvent.click(screen.getByText('LOGOUT'));
      expect(screen.getByTestId('login-page')).toBeInTheDocument();
    });
  });

  describe('when testing UI structure and styling', () => {
    test('should render app container with correct className', () => {
      isAuthenticated.mockReturnValue(false);

      const { container } = render(<App />);

      const appContainer = container.querySelector('.app-container');
      expect(appContainer).toBeInTheDocument();
    });

    test('should render navigation container with correct className', () => {
      isAuthenticated.mockReturnValue(false);

      const { container } = render(<App />);

      const navContainer = container.querySelector('.nav-container');
      expect(navContainer).toBeInTheDocument();
    });

    test('should have background image style on app container', () => {
      isAuthenticated.mockReturnValue(false);

      const { container } = render(<App />);

      const appContainer = container.querySelector('.app-container');
      expect(appContainer).toHaveStyle({
        backgroundImage: 'url(/frontend_background.jpg)',
      });
    });
  });
});
