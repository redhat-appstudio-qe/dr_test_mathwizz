/**
 * Styled history screen component for displaying problem history - presentational component only.
 *
 * This is a "dumb" presentational component that receives all data via props and has no internal
 * state or business logic. All state management and data fetching is handled by the parent
 * container (HistoryPage).
 *
 * **Component Type**: Dumb presentational component (no state, no business logic)
 *
 * **Design Pattern**: Container/Presentational Separation
 * - Parent (HistoryPage): Smart container with state and data fetching
 * - This component: Dumb presentation with props only
 * - Benefits: Easier testing, reusable UI, clear separation of concerns
 *
 * **Visual Design**: Pixel art retro terminal theme
 * - Dark blue semi-transparent background with thick 6px borders
 * - Pink/red shadow glow effect (rgba(233, 69, 96, 0.3)) for distinct history screen aesthetic
 * - Monospace fonts (Courier New) for retro terminal look
 * - Green header text (#4ecca3) with terminal-style ">" prompt and underscore cursor
 * - Nested dark blue boxes for individual history items
 * - Consistent with global pixel art theme but unique shadow color (pink vs black)
 *
 * **UI Structure**:
 * 1. Header section - retro terminal header "> HISTORY LOG _ MATHWIZZ v1.0"
 * 2. Content section - conditional rendering based on state:
 *    a) Loading state - "Loading history..." (className="loading" from global.css)
 *    b) Empty state - "No history yet. Solve some problems to get started!"
 *    c) Data state - List of history items with problem, answer, timestamp
 *
 * **Three Rendering States**:
 * - **Loading**: loading=true → Shows "Loading history..." with loading spinner styling
 * - **Empty**: loading=false, history.length===0 → Shows empty state message
 * - **Data**: loading=false, history.length>0 → Shows list of history items
 *
 * **History Item Structure** (each item displayed):
 * - Problem and answer: "{problem} = {answer}" (e.g., "2+2 = 4")
 * - Timestamp: Formatted date (e.g., "1/1/2024, 12:00:00 PM") using formatDate utility
 * - Visual: Dark blue nested box with green problem text, gray timestamp text
 *
 * **Security Considerations**:
 * - Good: All user data properly escaped by React's JSX (no XSS vulnerabilities)
 * - Good: No dangerouslySetInnerHTML usage
 * - Good: Problem, answer, and timestamp all rendered safely via {curly braces}
 * - Note: Relies on backend filtering user's own history (user can only see their data)
 *
 * **Testing Support**:
 * - data-testid="history-list" for testing list rendering
 * - data-testid="history-item" for testing individual items (repeated for each item)
 * - Stateless component makes testing straightforward (pass props, assert rendering)
 *
 * **Accessibility**:
 * - Semantic HTML: <ul> for list, <li> for items
 * - Text content is plain text (screen reader friendly)
 * - No interactive elements within this component (all interaction in parent)
 * - Missing: aria-label on list for screen reader context
 * - Missing: aria-live region for dynamic history updates
 *
 * @component
 * @param {Object} props - Component props
 * @param {Array<Object>} props.history - Array of history items to display
 * @param {number} props.history[].id - Unique identifier for item (used as React key)
 * @param {string} props.history[].problem - Mathematical expression (e.g., "2+2")
 * @param {string} props.history[].answer - Computed answer (e.g., "4")
 * @param {string} props.history[].created_at - ISO 8601 timestamp string
 * @param {boolean} props.loading - If true, shows "Loading history..." instead of data
 * @returns {JSX.Element} Rendered history screen with header and list/loading/empty state
 *
 * @example
 * // Used by HistoryPage (parent container)
 * <HistoryScreenComponent
 *   history={[
 *     {id: 1, problem: "2+2", answer: "4", created_at: "2024-01-01T12:00:00Z"},
 *     {id: 2, problem: "5*10", answer: "50", created_at: "2024-01-01T12:05:00Z"}
 *   ]}
 *   loading={false}
 * />
 *
 * @example
 * // Loading state
 * <HistoryScreenComponent history={[]} loading={true} />
 * // → Displays "Loading history..." with loading spinner styling
 *
 * @example
 * // Empty state
 * <HistoryScreenComponent history={[]} loading={false} />
 * // → Displays "No history yet. Solve some problems to get started!"
 */

import React from 'react';
import '../styles/global.css';
import { formatDate } from '../utils';

const HistoryScreenComponent = ({ history, loading }) => {
  /**
   * History screen container style - outer box for entire history display.
   *
   * **Visual Design**:
   * - Semi-transparent dark blue background (rgba(22, 33, 62, 0.8)) matching app theme
   * - Thick 6px dark blue border (#0f3460) for pronounced pixel art frame (thicker than calculator's 5px)
   * - Pink/red shadow glow effect (rgba(233, 69, 96, 0.3)) - UNIQUE to history screen
   *   - Different from calculator's black shadow (10px 10px 0px rgba(0, 0, 0, 0.5))
   *   - Creates distinct visual identity for history vs solver pages
   *   - 20px blur creates soft glow effect (vs calculator's hard shadow)
   * - Max width 700px (wider than calculator's 500px) to accommodate longer history entries
   * - Min height 400px maintains consistent size even when empty
   * - Centered with auto margin
   *
   * @type {Object}
   */
  const screenStyle = {
    background: 'rgba(22, 33, 62, 0.8)',
    border: '6px solid #0f3460',
    padding: '30px',
    boxShadow: '0px 0px 20px rgba(233, 69, 96, 0.3)',
    maxWidth: '700px',
    margin: '0 auto',
    minHeight: '400px',
  };

  /**
   * Header section style - retro terminal-style header.
   *
   * **Visual Design**:
   * - Green monospace text (#4ecca3) matching theme's accent color
   * - 2px bottom border separator
   * - 18px font size for header prominence
   *
   * **Content**: "> HISTORY LOG _ MATHWIZZ v1.0"
   * - ">" prompt character (terminal style)
   * - "HISTORY LOG" title
   * - "_" underscore as blinking cursor simulation (static, not animated)
   * - "MATHWIZZ v1.0" branding
   *
   * @type {Object}
   */
  const headerStyle = {
    color: '#4ecca3',
    borderBottom: '2px solid #0f3460',
    paddingBottom: '10px',
    marginBottom: '20px',
    fontFamily: 'Courier New, monospace',
    fontSize: '18px',
  };

  /**
   * List container style - removes default list styling.
   *
   * **Purpose**: Clean slate for custom-styled history items
   * - Removes default bullet points (listStyle: 'none')
   * - Removes default padding (padding: 0)
   * - Individual items styled via itemStyle
   *
   * @type {Object}
   */
  const listStyle = {
    listStyle: 'none',
    padding: 0,
  };

  /**
   * Individual history item style - card/box for each problem entry.
   *
   * **Visual Design**:
   * - Darker blue background (rgba(15, 52, 96, 0.7)) than container for inset/nested effect
   * - 2px border matching theme
   * - 10px bottom margin for spacing between items
   * - Light gray text (#eee) for content readability
   * - Monospace font for retro consistency
   *
   * **Contains**:
   * - Problem and answer text (problemStyle)
   * - Timestamp (timestampStyle)
   *
   * @type {Object}
   */
  const itemStyle = {
    background: 'rgba(15, 52, 96, 0.7)',
    border: '2px solid #0f3460',
    padding: '15px',
    marginBottom: '10px',
    fontFamily: 'Courier New, monospace',
    color: '#eee',
  };

  /**
   * Problem and answer text style - main content of each history item.
   *
   * **Visual Design**:
   * - Green color (#4ecca3) for prominence and consistency with theme
   * - 18px font size for easy readability (larger than timestamp)
   * - Bold weight for emphasis
   * - 5px bottom margin for spacing before timestamp
   *
   * **Content Format**: "{problem} = {answer}"
   * - Example: "2+2 = 4"
   * - Example: "25*4 = 100"
   *
   * @type {Object}
   */
  const problemStyle = {
    color: '#4ecca3',
    fontSize: '18px',
    fontWeight: 'bold',
    marginBottom: '5px',
  };

  /**
   * Timestamp text style - date/time when problem was solved.
   *
   * **Visual Design**:
   * - Gray color (#999) for secondary information (less prominent than problem)
   * - 12px font size (smaller than problem text)
   * - 5px top margin for spacing after problem
   *
   * **Content**: Formatted via formatDate() utility from utils.js
   * - Converts ISO 8601 string to locale-specific format
   * - Example input: "2024-01-01T12:00:00Z"
   * - Example output: "1/1/2024, 12:00:00 PM" (varies by locale)
   * - Timezone: UTC to local conversion handled by formatDate
   *
   * @type {Object}
   */
  const timestampStyle = {
    color: '#999',
    fontSize: '12px',
    marginTop: '5px',
  };

  /**
   * JSX Return - renders history screen with conditional content based on loading/data state.
   *
   * **Structure** (2 main sections):
   *
   * 1. **Screen container** (screenStyle):
   *    - Wraps entire history display
   *    - Provides visual box with retro terminal styling
   *    - Pink shadow glow (unique to history screen)
   *
   * 2. **Header section** (headerStyle):
   *    - Terminal-style header: "&gt; HISTORY LOG _ MATHWIZZ v1.0"
   *    - "&gt;" is HTML entity for ">" character (JSX requires escaping <> characters)
   *    - Green monospace text with bottom border separator
   *    - Always displayed regardless of loading/data state
   *
   * 3. **Content section** (conditional rendering - three states):
   *
   *    **a) Loading State** (loading=true):
   *       - Shows: <div className="loading">Loading history...</div>
   *       - className="loading" from global.css provides spinner/animation styling
   *       - Displayed during initial fetch and refresh operations
   *       - Takes precedence (checked first in ternary chain)
   *
   *    **b) Empty State** (loading=false, history.length===0):
   *       - Shows: "No history yet. Solve some problems to get started!"
   *       - Gray centered text (#999, centered, 40px padding)
   *       - Displayed when user has no solved problems yet
   *       - Inline style (not extracted constant) since used only once
   *       - Encourages user to use solver page
   *
   *    **c) Data State** (loading=false, history.length>0):
   *       - Shows: <ul> list of history items
   *       - data-testid="history-list" for testing list element
   *       - Maps over history array to create <li> elements
   *       - Each item contains:
   *         - Problem and answer: "{problem} = {answer}" (green, bold)
   *         - Timestamp: formatDate(created_at) (gray, smaller)
   *       - data-testid="history-item" repeated on each <li> for testing
   *
   * **Conditional Rendering Logic** (nested ternary):
   * ```
   * loading ?
   *   <Loading />
   * : history.length === 0 ?
   *   <Empty />
   * :
   *   <List />
   * ```
   *
   * **React Keys**:
   * - key={item.id || index} - prefers unique ID, falls back to array index
   * - item.id from database (better for React reconciliation)
   * - index fallback acceptable since list is append-only (no reordering/deletion)
   *
   * **Data Flow**:
   * - history prop: passed from HistoryPage (fetched from API)
   * - loading prop: passed from HistoryPage (managed during fetch)
   * - formatDate: imported utility function from utils.js
   *   - Converts ISO 8601 timestamp to locale-specific format
   *   - Handles timezone conversion (UTC to local)
   *
   * **Accessibility**:
   * - Semantic HTML: <ul> and <li> for list structure
   * - Text content is plain and screen reader friendly
   * - Missing: aria-label on <ul> (should describe "History of solved problems")
   * - Missing: aria-live region for dynamic updates (screen readers won't announce new items)
   *
   * **React Patterns**:
   * - Conditional rendering (ternary operator)
   * - Array.map() for list rendering
   * - Props destructuring for clean code
   * - Inline styles (component-specific, not global)
   *
   * **Testing**:
   * - data-testid="history-list" enables list testing
   * - data-testid="history-item" enables item testing (can count items, check content)
   * - Stateless component makes testing straightforward (pass props, assert rendering)
   *
   * @example
   * // Loading state rendering
   * // loading=true
   * // → <div className="loading">Loading history...</div>
   *
   * @example
   * // Empty state rendering
   * // loading=false, history=[]
   * // → "No history yet. Solve some problems to get started!"
   *
   * @example
   * // Data state rendering
   * // loading=false, history=[{id: 1, problem: "2+2", answer: "4", created_at: "2024-01-01T12:00:00Z"}]
   * // → <ul> with 1 <li> containing:
   * //   - "2+2 = 4" (green, bold)
   * //   - "1/1/2024, 12:00:00 PM" (gray, small)
   *
   * @example
   * // Multiple items rendering
   * // history=[{...}, {...}, {...}]
   * // → <ul> with 3 <li> elements, each with problem/answer and timestamp
   * // → Items ordered as received from API (typically newest first from backend)
   */
  return (
    <div style={screenStyle}>
      {/* Terminal-style header - always displayed */}
      <div style={headerStyle}>
        &gt; HISTORY LOG _ MATHWIZZ v1.0
      </div>

      {/* Conditional content rendering - three states: loading, empty, data */}
      {loading ? (
        /* Loading state - shown during fetch operations */
        <div className="loading">Loading history...</div>
      ) : history.length === 0 ? (
        /* Empty state - shown when user has no history yet */
        <div style={{ color: '#999', textAlign: 'center', padding: '40px' }}>
          No history yet. Solve some problems to get started!
        </div>
      ) : (
        /* Data state - list of history items */
        <ul style={listStyle} data-testid="history-list">
          {history.map((item, index) => (
            /* Individual history item - key uses ID or falls back to index */
            <li key={item.id || index} style={itemStyle} data-testid="history-item">
              {/* Problem and answer - green bold text */}
              <div style={problemStyle}>
                {item.problem} = {item.answer}
              </div>
              {/* Timestamp - gray small text, formatted via formatDate utility */}
              <div style={timestampStyle}>
                {formatDate(item.created_at)}
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};

export default HistoryScreenComponent;
