/**
 * React 18 application entry point.
 *
 * This module initializes the React application and mounts it to the DOM. It uses
 * React 18's createRoot API (introduced in React 18.0) for concurrent rendering features.
 * The App component is wrapped in React.StrictMode for development-time checks.
 *
 * **React 18 Features**:
 * - Concurrent rendering: Allows React to prepare multiple versions of UI simultaneously
 * - Automatic batching: Multiple state updates batched into single re-render
 * - Transitions API: Priority-based updates for better UX
 * - Suspense improvements: Better async component loading
 *
 * **React.StrictMode Benefits** (development only, stripped in production):
 * - Detects components with unsafe lifecycles
 * - Warns about legacy string ref API usage
 * - Warns about deprecated findDOMNode usage
 * - Detects unexpected side effects by double-invoking functions
 * - Detects legacy context API
 * - Ensures reusable state (React 18 feature)
 *
 * **DOM Mounting**:
 * - Target element: <div id="root"></div> in public/index.html
 * - If #root doesn't exist, ReactDOM.createRoot throws error
 * - Single-page application (SPA): Entire app rendered client-side
 *
 * **File Structure**:
 * - public/index.html: HTML template with <div id="root"></div>
 * - src/index.js: This file - React entry point
 * - src/App.jsx: Root React component with routing/auth
 * - src/components/*: Page and feature components
 *
 * **Build Process**:
 * - Development: webpack-dev-server bundles and serves with hot reload
 * - Production: `npm run build` creates optimized bundle in build/ folder
 * - React.StrictMode removed in production builds (tree-shaking)
 *
 * @module index
 * @see {@link https://react.dev/blog/2022/03/29/react-v18|React 18 Release}
 * @see {@link https://react.dev/reference/react/StrictMode|React.StrictMode}
 */
import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';

/**
 * React root instance for the application.
 *
 * Created using React 18's createRoot API, which enables concurrent rendering features.
 * Replaces the legacy ReactDOM.render() pattern used in React 17 and earlier.
 *
 * **Migration from React 17**:
 * ```javascript
 * // React 17 (legacy)
 * ReactDOM.render(<App />, document.getElementById('root'));
 *
 * // React 18 (current)
 * const root = ReactDOM.createRoot(document.getElementById('root'));
 * root.render(<App />);
 * ```
 *
 * **Error Handling**:
 * - If document.getElementById('root') returns null (element doesn't exist),
 *   createRoot throws error: "createRoot(...): Target container is not a DOM element"
 * - This indicates public/index.html is missing <div id="root"></div>
 *
 * @constant {ReactDOMRoot}
 * @see {@link https://react.dev/reference/react-dom/client/createRoot|createRoot API}
 */
const root = ReactDOM.createRoot(document.getElementById('root'));

/**
 * Renders the React application into the DOM.
 *
 * Mounts the App component wrapped in React.StrictMode to the root container.
 * This is the single point where the React virtual DOM connects to the actual DOM.
 *
 * **React.StrictMode Wrapper**:
 * - Development only (removed in production builds)
 * - Activates additional checks and warnings
 * - Double-invokes component functions to detect side effects
 * - Helps prepare codebase for future React features
 * - No visual UI changes (renders children normally)
 *
 * **Rendering Behavior**:
 * - Initial render: Creates virtual DOM tree and commits to real DOM
 * - Subsequent updates: Triggered by state changes in App or child components
 * - Concurrent mode: React can interrupt, pause, or restart rendering
 * - Batching: Multiple setState calls batched into single re-render
 *
 * **Note**: In development mode with StrictMode, components render twice to detect
 * side effects. This is intentional behavior - production renders only once.
 *
 * @function
 * @returns {void}
 */
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
