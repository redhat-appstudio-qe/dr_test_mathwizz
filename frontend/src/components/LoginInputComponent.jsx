/**
 * Reusable presentational component for login and registration forms.
 *
 * Provides a styled form with email and password inputs, error message display, and
 * submit button. Implements pixel art retro theme with dark blue transparent background,
 * thick borders, and yellow submit button. Pure presentation component with no business
 * logic - receives all data and handlers via props from parent containers (LoginPage,
 * RegisterPage).
 *
 * **Component Type**: Dumb presentational component (no state, no business logic)
 * **Parent Components**: LoginPage (buttonText="LOGIN"), RegisterPage (buttonText="CREATE ACCOUNT")
 *
 * **Design Pattern**: Container/Presentational component separation
 * - Smart containers (LoginPage, RegisterPage): Manage state, validation, API calls
 * - Dumb component (this): Only handles presentation and user input forwarding
 * - Benefits: Reusability, testability, clear separation of concerns
 *
 * **Styling Approach**:
 * - Inline styles for component-specific styling (pixel art theme)
 * - Global CSS for error message styling (.error-message class)
 * - No CSS modules or styled-components (minimal dependencies)
 * - Pixel art retro theme: dark blue, thick borders, monospace font, sharp corners (borderRadius: 0)
 *
 * **Accessibility**:
 * - Uses semantic HTML5 input types (type="email", type="password")
 * - Form element with onSubmit handler (Enter key works)
 * - Required attributes on inputs (HTML5 validation)
 * - Placeholder text for input hints
 * - data-testid attributes for automated testing
 *
 * 
 *
 * **Security Considerations**:
 * - **Good**: Uses type="password" to mask password input
 * - **Good**: No XSS vulnerabilities - all content properly escaped via JSX
 * - **Good**: Uses controlled components (React manages input values, prevents some attacks)
 * - **Good**: Form submission via onSubmit handler (prevents raw form POST)
 * - **Note**: Actual security handled by parent components (validation, API calls, token storage)
 *
 * @component
 * @param {Object} props - Component props
 * @param {string} props.email - Current email input value (controlled component)
 * @param {string} props.password - Current password input value (controlled component)
 * @param {Function} props.onEmailChange - Callback for email input changes (receives e.target.value)
 * @param {Function} props.onPasswordChange - Callback for password input changes (receives e.target.value)
 * @param {Function} props.onSubmit - Form submit handler (receives Event, should call e.preventDefault())
 * @param {string} props.buttonText - Text displayed on submit button ("LOGIN" or "CREATE ACCOUNT")
 * @param {string} props.error - Error message to display (empty string if no error)
 * @returns {JSX.Element} Styled authentication form
 *
 * @example
 * // Usage in LoginPage
 * <LoginInputComponent
 *   email={email}
 *   password={password}
 *   onEmailChange={setEmail}
 *   onPasswordChange={setPassword}
 *   onSubmit={handleSubmit}
 *   buttonText="LOGIN"
 *   error={error}
 * />
 *
 * @example
 * // Usage in RegisterPage (same component, different buttonText)
 * <LoginInputComponent
 *   email={email}
 *   password={password}
 *   onEmailChange={setEmail}
 *   onPasswordChange={setPassword}
 *   onSubmit={handleSubmit}
 *   buttonText="CREATE ACCOUNT"
 *   error={error}
 * />
 */
import React from 'react';
import '../styles/global.css';

/**
 * LoginInputComponent functional presentation component.
 *
 * Stateless functional component that renders authentication form UI.
 * All data and behavior passed via props from parent containers.
 *
 * @function
 * @param {Object} props - All props destructured in function signature
 */
const LoginInputComponent = ({ email, password, onEmailChange, onPasswordChange, onSubmit, buttonText, error }) => {
  /**
   * Container style with pixel art retro theme.
   *
   * Semi-transparent dark blue background with thick border and drop shadow effect.
   * Creates "floating panel" appearance over background image.
   *
   * **Colors**:
   * - background: rgba(22, 33, 62, 0.8) - dark blue with 80% opacity
   * - border: #0f3460 - darker blue for contrast
   * - boxShadow: 8px offset creates retro "3D" effect
   *
   * **Sizing**:
   * - maxWidth: 400px (constrained for readability)
   * - margin: 0 auto (centered horizontally)
   * - padding: 30px (internal spacing)
   *
   * **Shape**:
   * - borderRadius: 0 (sharp corners, pixel art style)
   * - border: 4px solid (thick border, pixel art aesthetic)
   *
   * @constant {Object}
   */
  const containerStyle = {
    background: 'rgba(22, 33, 62, 0.8)',
    border: '4px solid #0f3460',
    padding: '30px',
    boxShadow: '8px 8px 0px #0f3460',
    maxWidth: '400px',
    margin: '0 auto',
    borderRadius: '0',
  };

  /**
   * Input field style for email and password inputs.
   *
   * Dark blue background with lighter blue border, monospace font for retro theme.
   * Applied to both email and password inputs for consistent styling.
   *
   * **Colors**:
   * - background: rgba(15, 52, 96, 0.9) - slightly lighter blue than container, 90% opacity
   * - border: #0f3460 - matches container border color
   * - color: #eee - light gray text for readability on dark background
   *
   * **Typography**:
   * - fontFamily: Courier New, monospace (retro terminal aesthetic)
   * - fontSize: 16px (readable size)
   *
   * **Spacing**:
   * - padding: 15px (comfortable input area)
   * - margin: 10px 0 (vertical spacing between inputs)
   * - width: 100% (fills container width)
   *
   * @constant {Object}
   */
  const inputStyle = {
    background: 'rgba(15, 52, 96, 0.9)',
    border: '3px solid #0f3460',
    padding: '15px',
    margin: '10px 0',
    width: '100%',
    color: '#eee',
    fontSize: '16px',
    fontFamily: 'Courier New, monospace',
  };

  /**
   * Submit button style with yellow accent color.
   *
   * Transparent background with thick yellow border creates hollow button effect.
   * Yellow color stands out against blue theme, emphasizing primary action.
   *
   * **Colors**:
   * - background: transparent (shows container background through)
   * - border: 4px solid rgb(249, 229, 82) - bright yellow
   * - color: rgb(249, 229, 82) - matches border (text is yellow)
   *
   * **Typography**:
   * - fontFamily: Courier New, monospace (matches input fields)
   * - fontSize: 18px (slightly larger than inputs for emphasis)
   * - fontWeight: bold (emphasizes action)
   *
   * **Spacing**:
   * - padding: 12px 24px (comfortable click target)
   * - width: 100% (fills container width)
   * - marginTop: 10px (spacing from inputs/error message)
   *
   * **Interaction**:
   * - cursor: pointer (indicates clickable)
   * - borderRadius: 0 (sharp corners, pixel art consistency)
   *
   * @constant {Object}
   */
  const submitButtonStyle = {
    background: 'transparent',
    border: '4px solid rgb(249, 229, 82)',
    color: 'rgb(249, 229, 82)',
    padding: '12px 24px',
    width: '100%',
    marginTop: '10px',
    fontFamily: 'Courier New, monospace',
    fontSize: '18px',
    fontWeight: 'bold',
    cursor: 'pointer',
    borderRadius: '0',
  };

  /**
   * JSX Return: Renders authentication form with inputs and submit button.
   *
   * **Structure**:
   * 1. Outer div with containerStyle (dark blue panel)
   * 2. HTML form element with onSubmit handler
   * 3. Email input (type="email", HTML5 validation)
   * 4. Password input (type="password", masked text)
   * 5. Conditional error message (only shown if error prop is non-empty)
   * 6. Submit button with dynamic buttonText ("LOGIN" or "CREATE ACCOUNT")
   *
   * **HTML5 Input Types**:
   * - type="email": Browser validates basic email format, provides email keyboard on mobile
   * - type="password": Masks input characters (•••), prevents copy-paste in some browsers
   * - required attribute: HTML5 validation (browser prevents empty submission)
   *
   * **Controlled Components**:
   * - Email and password are controlled components (React manages value via state)
   * - value prop: Current input value from parent state
   * - onChange: Calls parent callback with e.target.value on each keystroke
   * - This pattern prevents some XSS attacks and ensures React state is source of truth
   *
   * **Form Submission**:
   * - onSubmit handler from parent (LoginPage.handleSubmit or RegisterPage.handleSubmit)
   * - Parent must call e.preventDefault() to prevent default browser form POST
   * - Enter key in either input triggers form submission (standard HTML form behavior)
   *
   * **Error Display**:
   * - Conditional rendering: {error && <div>} only renders if error is truthy
   * - .error-message class defined in global.css (red text styling)
   * - Positioned between inputs and submit button
   * - Displayed for both validation errors and API errors
   *
   * **Testing Support**:
   * - data-testid attributes on all interactive elements
   * - Enables automated testing with Testing Library (screen.getByTestId('email-input'))
   * - Test IDs: "email-input", "password-input", "submit-button"
   *
   * **Accessibility Issues** (not implemented):
   * - Missing <label> elements (screen readers can't associate labels with inputs)
   * - Missing aria-label or aria-labelledby attributes
   * - Error message not associated with inputs via aria-describedby
   * - No autocomplete attributes (autocomplete="email", autocomplete="current-password", etc.)
   * - No password visibility toggle (users can't verify typed password)
   *
   * @returns {JSX.Element} Form with email/password inputs, error display, and submit button
   */
  return (
    <div style={containerStyle}>
      <form onSubmit={onSubmit}>
        {/* Email input: HTML5 validation, controlled component */}
        <input
          type="email"
          placeholder="Email"
          value={email}
          onChange={(e) => onEmailChange(e.target.value)}
          style={inputStyle}
          required
          data-testid="email-input"
        />

        {/* Password input: Masked text, controlled component */}
        <input
          type="password"
          placeholder="Password"
          value={password}
          onChange={(e) => onPasswordChange(e.target.value)}
          style={inputStyle}
          required
          data-testid="password-input"
        />

        {/* Conditional error message: Only shown when error prop is non-empty */}
        {error && <div className="error-message">{error}</div>}

        {/* Submit button: Text changes based on context (LOGIN vs CREATE ACCOUNT) */}
        <button
          type="submit"
          style={submitButtonStyle}
          data-testid="submit-button"
        >
          {buttonText}
        </button>
      </form>
    </div>
  );
};

export default LoginInputComponent;
