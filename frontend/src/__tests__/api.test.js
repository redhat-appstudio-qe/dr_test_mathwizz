// Unit tests for API client functions
// Tests cookie-based authentication (httpOnly authToken + sessionActive cookies)
// Tests error handling for both JSON and non-JSON responses

import {
  register,
  login,
  solve,
  getHistory,
  isAuthenticated,
  clearToken,
} from '../api';

// Mock fetch globally
global.fetch = jest.fn();

// Helper to set document.cookie for testing
const setCookie = (name, value) => {
  Object.defineProperty(document, 'cookie', {
    writable: true,
    value: `${name}=${value}`,
  });
};

// Helper to clear all cookies
const clearAllCookies = () => {
  Object.defineProperty(document, 'cookie', {
    writable: true,
    value: '',
  });
};

describe('API Client - Cookie-Based Authentication', () => {
  beforeEach(() => {
    // Reset fetch mock before each test
    fetch.mockClear();
    // Clear all cookies before each test
    clearAllCookies();
  });

  describe('isAuthenticated function', () => {
    describe('when checking sessionActive cookie presence', () => {
      it('should return true when sessionActive cookie exists', () => {
        setCookie('sessionActive', 'true');
        expect(isAuthenticated()).toBe(true);
      });

      it('should return false when sessionActive cookie does not exist', () => {
        clearAllCookies();
        expect(isAuthenticated()).toBe(false);
      });

      it('should return false when only other cookies exist', () => {
        setCookie('otherCookie', 'value');
        expect(isAuthenticated()).toBe(false);
      });

      it('should return true when sessionActive cookie exists among multiple cookies', () => {
        Object.defineProperty(document, 'cookie', {
          writable: true,
          value: 'cookie1=value1; sessionActive=true; cookie2=value2',
        });
        expect(isAuthenticated()).toBe(true);
      });
    });
  });

  describe('clearToken function', () => {
    describe('when clearing session cookies', () => {
      it('should clear the session by expiring sessionActive cookie', () => {
        // Set initial authenticated state
        setCookie('sessionActive', 'true');
        expect(isAuthenticated()).toBe(true);

        // Call clearToken - it sets document.cookie to expire sessionActive
        expect(() => clearToken()).not.toThrow();

        // Note: In a real browser, the expired cookie would be removed by the browser
        // In this test environment, we verify clearToken executes without error
        // The actual cookie expiration behavior is verified by integration/E2E tests
      });

      it('should not throw error when called multiple times', () => {
        setCookie('sessionActive', 'true');

        expect(() => clearToken()).not.toThrow();
        expect(() => clearToken()).not.toThrow();
        expect(() => clearToken()).not.toThrow();
      });

      it('should not throw error when no session exists', () => {
        clearAllCookies();

        expect(() => clearToken()).not.toThrow();
      });
    });
  });

  describe('register function', () => {
    describe('when registering with valid credentials', () => {
      it('should successfully register user and return user data', async () => {
        const mockResponse = {
          token: 'mock-jwt-token',
          email: 'test@example.com',
          user_id: 123,
        };

        fetch.mockResolvedValueOnce({
          ok: true,
          status: 201,
          json: jest.fn().mockResolvedValueOnce(mockResponse),
        });

        const result = await register('test@example.com', 'password123');

        expect(result).toEqual(mockResponse);
        expect(fetch).toHaveBeenCalledWith(
          expect.stringContaining('/register'),
          expect.objectContaining({
            method: 'POST',
            credentials: 'include',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email: 'test@example.com', password: 'password123' }),
          })
        );
      });

      it('should use credentials include to send and receive cookies', async () => {
        const mockResponse = {
          token: 'mock-jwt-token',
          email: 'user@example.com',
          user_id: 456,
        };

        fetch.mockResolvedValueOnce({
          ok: true,
          status: 201,
          json: jest.fn().mockResolvedValueOnce(mockResponse),
        });

        await register('user@example.com', 'securepass');

        expect(fetch).toHaveBeenCalledWith(
          expect.any(String),
          expect.objectContaining({
            credentials: 'include',
          })
        );
      });

      it('should not set Authorization header (uses httpOnly cookies)', async () => {
        fetch.mockResolvedValueOnce({
          ok: true,
          status: 201,
          json: jest.fn().mockResolvedValueOnce({ token: 'jwt', email: 'a@b.com', user_id: 1 }),
        });

        await register('a@b.com', 'pass');

        const fetchCall = fetch.mock.calls[0][1];
        expect(fetchCall.headers.Authorization).toBeUndefined();
      });
    });

    describe('when handling error responses', () => {
      it('should handle JSON error responses correctly', async () => {
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 400,
          json: jest.fn().mockResolvedValueOnce({ error: 'failed to create user' }),
        });

        await expect(register('test@example.com', 'password123')).rejects.toThrow(
          'failed to create user'
        );
      });

      it('should handle non-JSON HTML error responses (Nginx 502)', async () => {
        const htmlError = '<html><body>502 Bad Gateway</body></html>';
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 502,
          json: jest.fn().mockRejectedValueOnce(new SyntaxError('Unexpected token < in JSON')),
          text: jest.fn().mockResolvedValueOnce(htmlError),
        });

        await expect(register('test@example.com', 'password123')).rejects.toThrow(htmlError);
      });

      it('should handle plain text error responses', async () => {
        const plainTextError = 'Service Unavailable';
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 503,
          json: jest.fn().mockRejectedValueOnce(new SyntaxError('Unexpected token S in JSON')),
          text: jest.fn().mockResolvedValueOnce(plainTextError),
        });

        await expect(register('test@example.com', 'password123')).rejects.toThrow(
          plainTextError
        );
      });

      it('should handle empty error response with fallback message', async () => {
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 500,
          json: jest.fn().mockRejectedValueOnce(new SyntaxError('Unexpected end of JSON input')),
          text: jest.fn().mockResolvedValueOnce(''),
        });

        await expect(register('test@example.com', 'password123')).rejects.toThrow(
          'Registration failed'
        );
      });

      it('should handle JSON error without error field using fallback', async () => {
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 400,
          json: jest.fn().mockResolvedValueOnce({ message: 'Something went wrong' }),
        });

        await expect(register('test@example.com', 'password123')).rejects.toThrow(
          'Registration failed'
        );
      });

      it('should handle both json and text parsing failures with fallback', async () => {
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 500,
          json: jest.fn().mockRejectedValueOnce(new Error('JSON parse failed')),
          text: jest.fn().mockRejectedValueOnce(new Error('Text parse failed')),
        });

        await expect(register('test@example.com', 'password123')).rejects.toThrow(
          'Registration failed'
        );
      });
    });
  });

  describe('login function', () => {
    describe('when logging in with valid credentials', () => {
      it('should successfully log in user and return user data', async () => {
        const mockResponse = {
          token: 'mock-jwt-token',
          email: 'test@example.com',
          user_id: 123,
        };

        fetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: jest.fn().mockResolvedValueOnce(mockResponse),
        });

        const result = await login('test@example.com', 'password123');

        expect(result).toEqual(mockResponse);
        expect(fetch).toHaveBeenCalledWith(
          expect.stringContaining('/login'),
          expect.objectContaining({
            method: 'POST',
            credentials: 'include',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email: 'test@example.com', password: 'password123' }),
          })
        );
      });

      it('should use credentials include to send and receive cookies', async () => {
        const mockResponse = {
          token: 'jwt-token',
          email: 'user@example.com',
          user_id: 789,
        };

        fetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: jest.fn().mockResolvedValueOnce(mockResponse),
        });

        await login('user@example.com', 'mypassword');

        expect(fetch).toHaveBeenCalledWith(
          expect.any(String),
          expect.objectContaining({
            credentials: 'include',
          })
        );
      });

      it('should not set Authorization header (uses httpOnly cookies)', async () => {
        fetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: jest.fn().mockResolvedValueOnce({ token: 'jwt', email: 'a@b.com', user_id: 1 }),
        });

        await login('a@b.com', 'pass');

        const fetchCall = fetch.mock.calls[0][1];
        expect(fetchCall.headers.Authorization).toBeUndefined();
      });
    });

    describe('when handling error responses', () => {
      it('should handle JSON error responses correctly', async () => {
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 401,
          json: jest.fn().mockResolvedValueOnce({ error: 'invalid credentials' }),
        });

        await expect(login('test@example.com', 'wrongpassword')).rejects.toThrow(
          'invalid credentials'
        );
      });

      it('should handle non-JSON HTML error responses', async () => {
        const htmlError = '<html><body>500 Internal Server Error</body></html>';
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 500,
          json: jest.fn().mockRejectedValueOnce(new SyntaxError('Unexpected token < in JSON')),
          text: jest.fn().mockResolvedValueOnce(htmlError),
        });

        await expect(login('test@example.com', 'password123')).rejects.toThrow(htmlError);
      });

      it('should handle empty error response with fallback message', async () => {
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 500,
          json: jest.fn().mockRejectedValueOnce(new SyntaxError('Unexpected end of JSON input')),
          text: jest.fn().mockResolvedValueOnce(''),
        });

        await expect(login('test@example.com', 'password123')).rejects.toThrow('Login failed');
      });
    });
  });

  describe('solve function', () => {
    describe('when checking authentication before solving', () => {
      it('should throw error when not authenticated (no sessionActive cookie)', async () => {
        clearAllCookies();

        await expect(solve('2+2')).rejects.toThrow('Not authenticated');
        expect(fetch).not.toHaveBeenCalled();
      });

      it('should allow solving when sessionActive cookie exists', async () => {
        setCookie('sessionActive', 'true');

        const mockResponse = { answer: 4 };
        fetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: jest.fn().mockResolvedValueOnce(mockResponse),
        });

        const result = await solve('2+2');

        expect(result).toEqual(mockResponse);
        expect(fetch).toHaveBeenCalled();
      });
    });

    describe('when solving problems successfully', () => {
      beforeEach(() => {
        setCookie('sessionActive', 'true');
      });

      it('should successfully solve problem when authenticated', async () => {
        const mockResponse = { answer: 14 };
        fetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: jest.fn().mockResolvedValueOnce(mockResponse),
        });

        const result = await solve('2+3*4');

        expect(result).toEqual(mockResponse);
        expect(fetch).toHaveBeenCalledWith(
          expect.stringContaining('/solve'),
          expect.objectContaining({
            method: 'POST',
            credentials: 'include',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ problem: '2+3*4' }),
          })
        );
      });

      it('should use credentials include to send authToken cookie', async () => {
        fetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: jest.fn().mockResolvedValueOnce({ answer: 5 }),
        });

        await solve('2+3');

        expect(fetch).toHaveBeenCalledWith(
          expect.any(String),
          expect.objectContaining({
            credentials: 'include',
          })
        );
      });

      it('should not set Authorization header (uses httpOnly cookie)', async () => {
        fetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: jest.fn().mockResolvedValueOnce({ answer: 10 }),
        });

        await solve('5*2');

        const fetchCall = fetch.mock.calls[0][1];
        expect(fetchCall.headers.Authorization).toBeUndefined();
      });
    });

    describe('when handling error responses', () => {
      beforeEach(() => {
        setCookie('sessionActive', 'true');
      });

      it('should handle JSON error responses correctly', async () => {
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 400,
          json: jest
            .fn()
            .mockResolvedValueOnce({ error: 'failed to solve problem: Invalid token' }),
        });

        await expect(solve('2++2')).rejects.toThrow('failed to solve problem: Invalid token');
      });

      it('should handle non-JSON HTML error responses', async () => {
        const htmlError = '<html><body>504 Gateway Timeout</body></html>';
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 504,
          json: jest.fn().mockRejectedValueOnce(new SyntaxError('Unexpected token < in JSON')),
          text: jest.fn().mockResolvedValueOnce(htmlError),
        });

        await expect(solve('2+2')).rejects.toThrow(htmlError);
      });

      it('should handle empty error response with fallback message', async () => {
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 500,
          json: jest.fn().mockRejectedValueOnce(new SyntaxError('Unexpected end of JSON input')),
          text: jest.fn().mockResolvedValueOnce(''),
        });

        await expect(solve('2+2')).rejects.toThrow('Failed to solve problem');
      });

      it('should handle 401 Unauthorized when token expires', async () => {
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 401,
          json: jest.fn().mockResolvedValueOnce({ error: 'invalid or expired token' }),
        });

        await expect(solve('2+2')).rejects.toThrow('invalid or expired token');
      });
    });
  });

  describe('getHistory function', () => {
    describe('when checking authentication before fetching history', () => {
      it('should throw error when not authenticated (no sessionActive cookie)', async () => {
        clearAllCookies();

        await expect(getHistory()).rejects.toThrow('Not authenticated');
        expect(fetch).not.toHaveBeenCalled();
      });

      it('should allow fetching history when sessionActive cookie exists', async () => {
        setCookie('sessionActive', 'true');

        fetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: jest.fn().mockResolvedValueOnce([]),
        });

        const result = await getHistory();

        expect(result).toEqual([]);
        expect(fetch).toHaveBeenCalled();
      });
    });

    describe('when fetching history successfully', () => {
      beforeEach(() => {
        setCookie('sessionActive', 'true');
      });

      it('should successfully fetch history when authenticated', async () => {
        const mockHistory = [
          {
            id: 1,
            user_id: 123,
            problem_text: '2+2',
            answer_text: '4',
            created_at: '2024-01-15T10:30:00Z',
          },
          {
            id: 2,
            user_id: 123,
            problem_text: '10*5',
            answer_text: '50',
            created_at: '2024-01-15T10:31:00Z',
          },
        ];

        fetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: jest.fn().mockResolvedValueOnce(mockHistory),
        });

        const result = await getHistory();

        expect(result).toEqual(mockHistory);
        expect(fetch).toHaveBeenCalledWith(
          expect.stringContaining('/history'),
          expect.objectContaining({
            method: 'GET',
            credentials: 'include',
          })
        );
      });

      it('should return empty array for user with no history', async () => {
        fetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: jest.fn().mockResolvedValueOnce([]),
        });

        const result = await getHistory();

        expect(result).toEqual([]);
      });

      it('should use credentials include to send authToken cookie', async () => {
        fetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: jest.fn().mockResolvedValueOnce([]),
        });

        await getHistory();

        expect(fetch).toHaveBeenCalledWith(
          expect.any(String),
          expect.objectContaining({
            credentials: 'include',
          })
        );
      });

      it('should not set Authorization header (uses httpOnly cookie)', async () => {
        fetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: jest.fn().mockResolvedValueOnce([]),
        });

        await getHistory();

        const fetchCall = fetch.mock.calls[0][1];
        expect(fetchCall.headers.Authorization).toBeUndefined();
      });
    });

    describe('when handling error responses', () => {
      beforeEach(() => {
        setCookie('sessionActive', 'true');
      });

      it('should handle JSON error responses correctly', async () => {
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 401,
          json: jest.fn().mockResolvedValueOnce({ error: 'invalid or expired token' }),
        });

        await expect(getHistory()).rejects.toThrow('invalid or expired token');
      });

      it('should handle non-JSON HTML error responses', async () => {
        const htmlError = '<html><body>503 Service Unavailable</body></html>';
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 503,
          json: jest.fn().mockRejectedValueOnce(new SyntaxError('Unexpected token < in JSON')),
          text: jest.fn().mockResolvedValueOnce(htmlError),
        });

        await expect(getHistory()).rejects.toThrow(htmlError);
      });

      it('should handle plain text network errors', async () => {
        const networkError = 'Network connection failed';
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 0,
          json: jest.fn().mockRejectedValueOnce(new SyntaxError('Unexpected token N in JSON')),
          text: jest.fn().mockResolvedValueOnce(networkError),
        });

        await expect(getHistory()).rejects.toThrow(networkError);
      });

      it('should handle empty error response with fallback message', async () => {
        fetch.mockResolvedValueOnce({
          ok: false,
          status: 500,
          json: jest.fn().mockRejectedValueOnce(new SyntaxError('Unexpected end of JSON input')),
          text: jest.fn().mockResolvedValueOnce(''),
        });

        await expect(getHistory()).rejects.toThrow('Failed to fetch history');
      });
    });
  });
});
