/**
 * Styled calculator component for the solver page - presentational component only.
 *
 * This is a "dumb" presentational component that receives all data and behavior via props.
 * It has no internal state or business logic, making it highly reusable and testable.
 * All state management and logic is handled by the parent container (SolverPage).
 *
 * **Component Type**: Dumb presentational component (no state, no business logic)
 *
 * **Design Pattern**: Container/Presentational Separation
 * - Parent (SolverPage): Smart container with state and logic
 * - This component: Dumb presentation with props only
 * - Benefits: Easier testing, reusable UI, clear separation of concerns
 *
 * **Visual Design**: Pixel art retro calculator theme
 * - Dark blue semi-transparent background with thick borders
 * - Monospace fonts (Courier New) for retro aesthetic
 * - Yellow transparent button for primary action (SOLVE)
 * - Green text for answer display (#4ecca3)
 * - Consistent with global pixel art theme across app
 *
 * **UI Structure**:
 * 1. Answer display screen (top) - shows result or "Ready..." placeholder
 * 2. Input area (middle) - text input for mathematical expression
 * 3. Error message (conditional) - displayed if validation/API error occurs
 * 4. SOLVE button (bottom) - triggers solve operation
 * 5. Supported operations hint (bottom) - "Supports: + - * / ( )"
 *
 * **Accessibility Features**:
 * - HTML5 form element for proper Enter key handling
 * - type="text" for expression input (not type="number" to allow operators)
 * - required attribute implied by validation
 * - Placeholder text for UX guidance
 * - Disabled states during loading prevent accidental submissions
 * - data-testid attributes for testing (answer-display, problem-input, solve-button)
 *
 * **Missing Accessibility**:
 * - No <label> element for input field (should have <label htmlFor="problem-input">)
 * - No aria-label or aria-labelledby attributes
 * - No autocomplete attribute (autocomplete="off" recommended for calculator)
 * - No aria-live region for answer display (screen readers won't announce new answers)
 * - No focus management (should focus input after solve completes)
 *
 * **Security Considerations**:
 * - Good: All user input properly escaped by React's JSX
 * - Good: No dangerouslySetInnerHTML usage
 * - Good: Uses controlled component pattern (React manages value)
 * - Good: Form onSubmit prevents default and calls parent handler
 * - Note: All security validation handled by parent (SolverPage) and backend
 *
 * **Testing Support**:
 * - data-testid="answer-display" for answer screen
 * - data-testid="problem-input" for input field
 * - data-testid="solve-button" for submit button
 * - Controlled component makes testing easy (set props, assert rendering)
 *
 * @component
 * @param {Object} props - Component props
 * @param {string} props.problem - Current mathematical expression being entered
 * @param {string} props.answer - Computed result to display (or empty string)
 * @param {Function} props.onProblemChange - Callback when input changes, receives new value
 * @param {Function} props.onSolve - Callback when form submitted (Enter key or button click)
 * @param {boolean} props.loading - If true, disables input/button and shows "SOLVING..." text
 * @param {string} props.error - Error message to display (or empty string for no error)
 * @returns {JSX.Element} Rendered calculator interface
 *
 * @example
 * // Used by SolverPage (parent container)
 * <CalculatorComponent
 *   problem="2+2"
 *   answer="4"
 *   onProblemChange={(val) => setProblem(val)}
 *   onSolve={handleSolve}
 *   loading={false}
 *   error=""
 * />
 *
 * @example
 * // Loading state
 * <CalculatorComponent
 *   problem="25*4"
 *   answer=""
 *   onProblemChange={setProblem}
 *   onSolve={handleSolve}
 *   loading={true}  // Disables input and button
 *   error=""
 * />
 *
 * @example
 * // Error state
 * <CalculatorComponent
 *   problem=""
 *   answer=""
 *   onProblemChange={setProblem}
 *   onSolve={handleSolve}
 *   loading={false}
 *   error="Please enter a mathematical expression"
 * />
 */

import React from 'react';
import '../styles/global.css';

const CalculatorComponent = ({ problem, answer, onProblemChange, onSolve, loading, error }) => {
  /**
   * Calculator container style - outer box for entire calculator component.
   *
   * **Visual Design**:
   * - Semi-transparent dark blue background (rgba(22, 33, 62, 0.8)) for layered effect over page background
   * - Thick 5px dark blue border (#0f3460) for pixel art aesthetic
   * - 10px box shadow offset for 3D depth effect (retro game UI style)
   * - Max width 500px prevents excessive stretching on wide screens
   * - Centered with auto margin
   *
   * @type {Object}
   */
  const calculatorStyle = {
    background: 'rgba(22, 33, 62, 0.8)',
    border: '5px solid #0f3460',
    padding: '20px',
    boxShadow: '10px 10px 0px rgba(0, 0, 0, 0.5)',
    maxWidth: '500px',
    margin: '0 auto',
  };

  /**
   * Answer display screen style - top screen showing result or placeholder.
   *
   * **Visual Design**:
   * - Darker blue background (rgba(15, 52, 96, 0.9)) than container for inset effect
   * - Green monospace text (#4ecca3) for retro terminal/calculator aesthetic
   * - 24px font size for easy readability
   * - Right-aligned text (typical calculator display alignment)
   * - Min height 80px maintains consistent size regardless of content
   * - word-break: break-all prevents overflow if answer is very long
   *
   * **Content**:
   * - Shows "Ready..." when no answer (placeholder)
   * - Shows "= [answer]" when answer exists (e.g., "= 4")
   *
   * @type {Object}
   */
  const screenStyle = {
    background: 'rgba(15, 52, 96, 0.9)',
    border: '3px solid #0f3460',
    padding: '20px',
    minHeight: '80px',
    marginBottom: '20px',
    fontFamily: 'Courier New, monospace',
    fontSize: '24px',
    color: '#4ecca3',
    textAlign: 'right',
    wordBreak: 'break-all',
  };

  /**
   * Input area container style - wraps the text input field.
   *
   * **Visual Design**:
   * - Same dark blue background as screen for visual consistency
   * - 3px border with padding creates inset input area
   * - Contains the actual <input> element with .pixel-input class
   *
   * **Note**: The input field itself uses .pixel-input class from global.css
   *
   * @type {Object}
   */
  const inputAreaStyle = {
    background: 'rgba(15, 52, 96, 0.9)',
    border: '3px solid #0f3460',
    padding: '15px',
    marginBottom: '15px',
  };

  /**
   * SOLVE button style - primary action button to submit expression.
   *
   * **Visual Design**:
   * - Transparent background with thick yellow border (hollow button effect)
   * - Yellow color (rgb(249, 229, 82)) for high contrast and primary action emphasis
   * - 100% width for easy clicking (full-width button)
   * - Monospace bold font for consistency with retro theme
   * - border-radius: 0 for sharp pixel art corners (no rounding)
   *
   * **States**:
   * - Normal: Shows "SOLVE" text, clickable cursor
   * - Loading: Shows "SOLVING..." text, disabled (handled in JSX, not style)
   *
   * @type {Object}
   */
  const solveButtonStyle = {
    background: 'transparent',
    border: '4px solid rgb(249, 229, 82)',
    color: 'rgb(249, 229, 82)',
    padding: '12px 24px',
    width: '100%',
    fontFamily: 'Courier New, monospace',
    fontSize: '18px',
    fontWeight: 'bold',
    cursor: 'pointer',
    borderRadius: '0',
  };

  /**
   * Form submission handler - prevents default form submit and calls parent callback.
   *
   * **Flow**:
   * 1. Receives form submit event (triggered by Enter key or button click)
   * 2. Prevents default form submission (e.preventDefault) - stops page reload
   * 3. Calls parent's onSolve callback which handles validation and API call
   *
   * **Why preventDefault?**
   * - Default form submit would reload the page (traditional HTML behavior)
   * - SPA pattern requires JavaScript-only handling (no page reload)
   * - Parent (SolverPage) handles all logic - this component just delegates
   *
   * **Triggered by**:
   * - User clicks SOLVE button (type="submit")
   * - User presses Enter while input is focused (form default behavior)
   *
   * @param {Event} e - Form submit event
   * @returns {void}
   *
   * @example
   * // User presses Enter in input field
   * // → <form onSubmit={handleSubmit}> receives event
   * // → e.preventDefault() prevents page reload
   * // → onSolve() called (parent's handleSolve function)
   * // → Parent validates and calls API
   */
  const handleSubmit = (e) => {
    e.preventDefault();
    onSolve();
  };

  /**
   * JSX Return - renders calculator UI with all interactive elements.
   *
   * **Structure** (5 main sections):
   *
   * 1. **Calculator container** (calculatorStyle):
   *    - Wraps entire component
   *    - Provides visual calculator box with retro styling
   *
   * 2. **Answer display screen** (screenStyle):
   *    - Shows computed result or placeholder
   *    - Conditional rendering: answer ? "= {answer}" : "Ready..."
   *    - data-testid="answer-display" for testing
   *    - Example displays: "Ready...", "= 4", "= 100"
   *
   * 3. **Form** (handleSubmit prevents default):
   *    - Wraps input and button for Enter key support
   *    - onSubmit calls handleSubmit which calls parent's onSolve
   *
   *    a) **Input area** (inputAreaStyle):
   *       - <input type="text"> for mathematical expression
   *       - Controlled component: value={problem}, onChange updates parent state
   *       - className="pixel-input" from global.css for consistent styling
   *       - placeholder provides UX guidance ("Enter problem (e.g., 5*10)")
   *       - disabled={loading} prevents editing during API call
   *       - data-testid="problem-input" for testing
   *       - Missing: <label> element for accessibility (should be added)
   *       - Missing: autocomplete="off" attribute (recommended for calculator)
   *
   *    b) **Error message** (conditional):
   *       - Rendered only if error prop is truthy (error &&...)
   *       - className="error-message" from global.css (red text styling)
   *       - Displays validation errors or API errors
   *       - Examples: "Please enter a mathematical expression", "Invalid characters..."
   *
   *    c) **SOLVE button** (type="submit"):
   *       - Primary action button with yellow transparent styling
   *       - Conditional text: loading ? "SOLVING..." : "SOLVE"
   *       - disabled={loading} prevents multiple submissions
   *       - type="submit" triggers form onSubmit (Enter key works)
   *       - data-testid="solve-button" for testing
   *
   * 4. **Supported operations hint** (inline style):
   *    - Small gray text at bottom: "Supports: + - * / ( )"
   *    - Informs users of allowed operators
   *    - 12px font, #999 color, centered
   *
   * **Accessibility Issues**:
   * - No <label htmlFor="problem-input"> for screen readers
   * - No aria-label on input field
   * - No aria-live on answer display (screen readers won't announce new answers)
   * - No focus management (should focus input after solve)
   * - Missing autocomplete attribute
   *
   * **React Patterns**:
   * - Controlled components (value and onChange for input)
   * - Conditional rendering (error &&..., ternary for answer/button text)
   * - Event delegation (onChange, onSubmit)
   * - Props destructuring for clean code
   *
   * **Testing**:
   * - data-testid attributes enable reliable testing
   * - Controlled component pattern makes state testing easy
   * - All UI states testable via props (normal, loading, error, answer)
   *
   * @example
   * // Normal state (ready for input)
   * // answer="" → displays "Ready..."
   * // loading=false → input enabled, button shows "SOLVE"
   * // error="" → no error message displayed
   *
   * @example
   * // Loading state (API call in progress)
   * // loading=true → input disabled, button disabled
   * // button text changes to "SOLVING..."
   *
   * @example
   * // Success state (answer received)
   * // answer="4" → displays "= 4"
   * // problem cleared by parent after success
   *
   * @example
   * // Error state (validation or API error)
   * // error="Please enter a mathematical expression"
   * // Error message displayed in red below input
   */
  return (
    <div style={calculatorStyle}>
      {/* Answer display screen - shows result or "Ready..." placeholder */}
      <div style={screenStyle} data-testid="answer-display">
        {answer ? `= ${answer}` : 'Ready...'}
      </div>

      {/* Form wraps input and button for Enter key submission support */}
      <form onSubmit={handleSubmit}>
        {/* Input area container */}
        <div style={inputAreaStyle}>
          {/* Mathematical expression input - controlled component, no label (accessibility issue) */}
          <input
            type="text"
            placeholder="Enter problem (e.g., 5*10)"
            value={problem}
            onChange={(e) => onProblemChange(e.target.value)}
            className="pixel-input"
            style={{ marginBottom: '0' }}
            disabled={loading}
            data-testid="problem-input"
          />
        </div>

        {/* Conditional error message - displayed if error prop is truthy */}
        {error && <div className="error-message">{error}</div>}

        {/* SOLVE button - shows "SOLVING..." when loading, disabled during API call */}
        <button
          type="submit"
          style={solveButtonStyle}
          disabled={loading}
          data-testid="solve-button"
        >
          {loading ? 'SOLVING...' : 'SOLVE'}
        </button>
      </form>

      {/* Supported operations hint - informs users of allowed operators */}
      <div style={{ marginTop: '15px', fontSize: '12px', color: '#999', textAlign: 'center' }}>
        Supports: + - * / ( )
      </div>
    </div>
  );
};

export default CalculatorComponent;
