/**
 * Main application component providing routing and authentication state management.
 *
 * This is the root component of the MathWizz frontend application. It manages global
 * authentication state, client-side routing between pages (login, register, solve, history),
 * and conditional rendering based on authentication status. Implements a simple manual routing
 * system using React state instead of react-router to minimize dependencies.
 *
 * **Architecture**:
 * - Single-page application (SPA) with client-side routing
 * - Authentication state managed in React state (synchronized with localStorage)
 * - Conditional rendering: unauthenticated users see login/register, authenticated see solver/history
 * - Navigation via button clicks that update currentPage state
 * - Pixel art retro theme with custom button styles
 *
 * **Authentication Flow**:
 * 1. On mount: Check localStorage for token via isAuthenticated()
 * 2. If token exists: setAuthenticated(true) and show solver page
 * 3. On login/register success: setAuthenticated(true) and navigate to solver
 * 4. On logout: clearToken(), setAuthenticated(false), navigate to login
 *
 * **Routing Pages**:
 * - Unauthenticated: login (default), register
 * - Authenticated: solve (default), history
 * - Navigation buttons conditionally rendered based on auth state
 *
 * **Styling Approach**:
 * - Pixel art retro theme with custom CSS
 * - Background image with contain sizing
 * - Inline styles for buttons
 * - Global CSS imported from styles/global.css
 *
 * @component
 * @returns {JSX.Element} The application root component with routing and navigation
 *
 * @example
 * // Rendered in index.js
 * import App from './App';
 * root.render(
 *   <React.StrictMode>
 *     <App />
 *   </React.StrictMode>
 * );
 */
import React, { useState, useEffect } from 'react';
import { isAuthenticated, clearToken } from './api';
import RegisterPage from './components/RegisterPage';
import LoginPage from './components/LoginPage';
import SolverPage from './components/SolverPage';
import HistoryPage from './components/HistoryPage';
import './styles/global.css';

/**
 * Main App functional component with routing and authentication state.
 *
 * Uses React hooks (useState, useEffect) to manage local state:
 * - currentPage: string representing active page ('login', 'register', 'solve', 'history')
 * - authenticated: boolean representing whether user has valid token in localStorage
 *
 * **State Management**:
 * - No external state management library (Redux, Context API, etc.)
 * - State local to App component, passed to children via props
 * - Authentication state synchronized with localStorage via isAuthenticated() check
 *
 * **Lifecycle**:
 * 1. Mount: useEffect checks isAuthenticated() and sets initial state
 * 2. User interaction: Button clicks update currentPage state
 * 3. Login/register success: Callbacks update authenticated state and navigate
 * 4. Logout: clearToken(), update state, navigate to login
 * 5. Re-render: renderPage() returns appropriate component based on state
 *
 * @function
 */
const App = () => {
  /**
   * Current active page identifier.
   *
   * Determines which page component to render via renderPage() function.
   * Default is 'login' for unauthenticated users.
   *
   * **Valid Values**:
   * - Unauthenticated: 'login', 'register'
   * - Authenticated: 'solve', 'history'
   *
   * @type {string}
   * @default 'login'
   */
  const [currentPage, setCurrentPage] = useState('login');

  /**
   * Global authentication state.
   *
   * Tracks whether user is authenticated (has valid JWT token in localStorage).
   * Controls conditional rendering of navigation buttons and page access.
   *
   * **Synchronization with localStorage**:
   * - On mount: Set to true if isAuthenticated() returns true
   * - On login/register: Set to true by success callbacks
   * - On logout: Set to false by handleLogout()
   *
   * @type {boolean}
   * @default false
   */
  const [authenticated, setAuthenticated] = useState(false);

  /**
   * Initialization effect: Check authentication status on component mount.
   *
   * Runs once on mount (empty dependency array) to check if user has existing
   * authentication token in localStorage. If token exists, automatically
   * authenticates user and navigates to solver page.
   *
   * **Behavior**:
   * 1. Calls isAuthenticated() to check localStorage for 'authToken'
   * 2. If token exists: setAuthenticated(true) and setCurrentPage('solve')
   * 3. If no token: user remains unauthenticated, stays on default 'login' page
   *
   * **Empty Dependency Array**: Runs only once on mount, never on re-render. This is
   * intentional - we only want to check auth status once at startup.
   *
   * @effect
   * @listens {mount} Component mount event
   */
  useEffect(() => {
    if (isAuthenticated()) {
      setAuthenticated(true);
      setCurrentPage('solve');
    }
  }, []);

  /**
   * Callback handler for successful login.
   *
   * Called by LoginPage component when user successfully authenticates.
   * Updates application state to reflect authenticated status and navigates
   * to solver page as the default landing page for authenticated users.
   *
   * **Flow**:
   * 1. User enters credentials in LoginPage
   * 2. LoginPage calls api.login()
   * 3. api.login() saves token to localStorage and returns user data
   * 4. LoginPage calls this callback
   * 5. App updates state: authenticated=true, currentPage='solve'
   * 6. App re-renders with solver page and authenticated navigation
   *
   * @function
   * @callback
   * @returns {void}
   */
  const handleLoginSuccess = () => {
    setAuthenticated(true);
    setCurrentPage('solve');
  };

  /**
   * Callback handler for successful registration.
   *
   * Called by RegisterPage component when user successfully creates account.
   * Identical behavior to handleLoginSuccess - updates state to authenticated
   * and navigates to solver. Registration endpoint returns auth token just like
   * login, so newly registered users are immediately authenticated.
   *
   * **Flow**:
   * 1. User enters email/password in RegisterPage
   * 2. RegisterPage calls api.register()
   * 3. api.register() creates account, saves token to localStorage
   * 4. RegisterPage calls this callback
   * 5. App updates state: authenticated=true, currentPage='solve'
   * 6. User immediately sees solver page without needing separate login
   *
   * @function
   * @callback
   * @returns {void}
   */
  const handleRegisterSuccess = () => {
    setAuthenticated(true);
    setCurrentPage('solve');
  };

  /**
   * Logout handler: Clears authentication and returns to login page.
   *
   * Called when user clicks LOGOUT button. Removes JWT token from localStorage
   * and updates application state to unauthenticated. Note that this is client-side
   * logout only - token remains valid on server until expiration (24 hours).
   *
   * **Flow**:
   * 1. User clicks LOGOUT button
   * 2. clearToken() removes 'authToken' from localStorage
   * 3. setAuthenticated(false) updates app state
   * 4. setCurrentPage('login') navigates to login page
   * 5. App re-renders with login/register buttons
   *
   * @function
   * @returns {void}
   */
  const handleLogout = () => {
    clearToken();
    setAuthenticated(false);
    setCurrentPage('login');
  };

  /**
   * Page renderer: Returns appropriate component based on authentication and navigation state.
   *
   * Implements client-side routing logic using switch statements. Routes differ based
   * on authentication status:
   * - Unauthenticated: login (default) or register
   * - Authenticated: solve (default) or history
   *
   * **Unauthenticated Routes**:
   * - 'register': <RegisterPage /> with onRegisterSuccess callback
   * - 'login' or default: <LoginPage /> with onLoginSuccess callback
   *
   * **Authenticated Routes**:
   * - 'solve' or default: <SolverPage /> (no props needed)
   * - 'history': <HistoryPage /> (no props needed)
   *
   * **Default Behavior**:
   * - Unauthenticated: Defaults to login page for any unknown currentPage value
   * - Authenticated: Defaults to solver page for any unknown currentPage value
   *
   * **Route Protection**:
   * - Unauthenticated users cannot access solve/history pages (first if block)
   * - Authenticated users cannot access login/register pages (implicit in routing logic)
   * - No URL-based routing - all navigation via state
   *
   * @function
   * @returns {JSX.Element} The page component to render (LoginPage, RegisterPage, SolverPage, or HistoryPage)
   *
   * @example
   * // When authenticated=false and currentPage='login'
   * renderPage() // → <LoginPage onLoginSuccess={handleLoginSuccess} />
   *
   * @example
   * // When authenticated=true and currentPage='history'
   * renderPage() // → <HistoryPage />
   */
  const renderPage = () => {
    if (!authenticated) {
      switch (currentPage) {
        case 'register':
          return <RegisterPage onRegisterSuccess={handleRegisterSuccess} />;
        case 'login':
        default:
          return <LoginPage onLoginSuccess={handleLoginSuccess} />;
      }
    }

    switch (currentPage) {
      case 'solve':
        return <SolverPage />;
      case 'history':
        return <HistoryPage />;
      default:
        return <SolverPage />;
    }
  };

  /**
   * Main container style with retro pixel art background.
   *
   * Applies background image and full-viewport sizing to the root app container.
   *
   * **Background Image**: '/frontend_background.jpg' from public folder.
   *
   * **Sizing**:
   * - minHeight: 100vh (full viewport height, allows scrolling if content exceeds)
   * - width: 100vw (full viewport width)
   * - backgroundSize: contain (image fully visible, may leave empty space)
   * - backgroundPosition: center (image centered)
   * - backgroundRepeat: no-repeat (single image, no tiling)
   *
   * @constant {Object}
   */
  const appStyle = {
    backgroundImage: 'url(/frontend_background.jpg)',
    backgroundSize: 'contain',
    backgroundPosition: 'center',
    backgroundRepeat: 'no-repeat',
    minHeight: '100vh',
    width: '100vw',
  };

  /**
   * LOGIN button style (blue, unauthenticated navigation).
   *
   * Pixel art retro theme with thick borders, monospace font, transparent background.
   * Blue color (#4da6ff) for login action.
   *
   * @constant {Object}
   */
  const loginButtonStyle = {
    background: 'transparent',
    border: '4px solid #4da6ff',
    borderRadius: '0',
    color: '#4da6ff',
    padding: '12px 40px',
    fontFamily: 'Courier New, monospace',
    fontSize: '24px',
    fontWeight: 'bold',
    cursor: 'pointer',
  };

  /**
   * REGISTER button style (green, unauthenticated navigation).
   *
   * Pixel art retro theme with thick borders, monospace font, transparent background.
   * Green color (rgb(59, 218, 72)) for registration action.
   *
   * @constant {Object}
   */
  const registerButtonStyle = {
    background: 'transparent',
    border: '4px solid rgb(59, 218, 72)',
    borderRadius: '0',
    color: 'rgb(59, 218, 72)',
    padding: '12px 40px',
    fontFamily: 'Courier New, monospace',
    fontSize: '24px',
    fontWeight: 'bold',
    cursor: 'pointer',
  };

  /**
   * SOLVER button style (blue, authenticated navigation).
   *
   * Pixel art retro theme with thick borders, monospace font, transparent background.
   * Blue color (#4da6ff) for solver page navigation.
   *
   * @constant {Object}
   */
  const solverButtonStyle = {
    background: 'transparent',
    border: '4px solid #4da6ff',
    borderRadius: '0',
    color: '#4da6ff',
    padding: '12px 40px',
    fontFamily: 'Courier New, monospace',
    fontSize: '24px',
    fontWeight: 'bold',
    cursor: 'pointer',
  };

  /**
   * HISTORY button style (green, authenticated navigation).
   *
   * Pixel art retro theme with thick borders, monospace font, transparent background.
   * Green color (rgb(59, 218, 72)) for history page navigation.
   *
   * @constant {Object}
   */
  const historyButtonStyle = {
    background: 'transparent',
    border: '4px solid rgb(59, 218, 72)',
    borderRadius: '0',
    color: 'rgb(59, 218, 72)',
    padding: '12px 40px',
    fontFamily: 'Courier New, monospace',
    fontSize: '24px',
    fontWeight: 'bold',
    cursor: 'pointer',
  };

  /**
   * LOGOUT button style (yellow, authenticated navigation).
   *
   * Pixel art retro theme with thick borders, monospace font, transparent background.
   * Yellow color (rgb(249, 229, 82)) for logout action - visually distinct from
   * blue (solver) and green (history) to emphasize destructive/exit action.
   *
   * @constant {Object}
   */
  const logoutButtonStyle = {
    background: 'transparent',
    border: '4px solid rgb(249, 229, 82)',
    borderRadius: '0',
    color: 'rgb(249, 229, 82)',
    padding: '12px 40px',
    fontFamily: 'Courier New, monospace',
    fontSize: '24px',
    fontWeight: 'bold',
    cursor: 'pointer',
  };

  /**
   * JSX Return: Renders app container with conditional navigation and page content.
   *
   * **Structure**:
   * 1. Root div with app-container class and background image style
   * 2. Conditional navigation buttons (unauthenticated vs authenticated)
   * 3. Current page component rendered by renderPage()
   *
   * **Conditional Rendering**:
   * - If !authenticated: Show LOGIN and REGISTER buttons
   * - If authenticated: Show SOLVER, HISTORY, and LOGOUT buttons
   *
   * **Navigation Buttons**:
   * - LOGIN: Sets currentPage='login' (triggers re-render with LoginPage)
   * - REGISTER: Sets currentPage='register' (triggers re-render with RegisterPage)
   * - SOLVER: Sets currentPage='solve' (triggers re-render with SolverPage)
   * - HISTORY: Sets currentPage='history' (triggers re-render with HistoryPage)
   * - LOGOUT: Calls handleLogout() which clears token and sets authenticated=false
   *
   * **CSS Classes**:
   * - .app-container: Root container (styled via inline appStyle + global.css)
   * - .nav-container: Navigation button container (defined in global.css)
   *
   * @returns {JSX.Element} The complete app UI with navigation and active page
   */
  return (
    <div className="app-container" style={appStyle}>
      {/* Conditional navigation: unauthenticated users see LOGIN/REGISTER */}
      {!authenticated ? (
        <div className="nav-container">
          <button
            onClick={() => setCurrentPage('login')}
            style={loginButtonStyle}
          >
            LOGIN
          </button>
          <button
            onClick={() => setCurrentPage('register')}
            style={registerButtonStyle}
          >
            REGISTER
          </button>
        </div>
      ) : (
        /* Authenticated users see SOLVER/HISTORY/LOGOUT */
        <div className="nav-container">
          <button
            onClick={() => setCurrentPage('solve')}
            style={solverButtonStyle}
          >
            SOLVER
          </button>
          <button
            onClick={() => setCurrentPage('history')}
            style={historyButtonStyle}
          >
            HISTORY
          </button>
          <button
            onClick={handleLogout}
            style={logoutButtonStyle}
          >
            LOGOUT
          </button>
        </div>
      )}

      {/* Render current page component based on authenticated + currentPage state */}
      {renderPage()}
    </div>
  );
};

export default App;
