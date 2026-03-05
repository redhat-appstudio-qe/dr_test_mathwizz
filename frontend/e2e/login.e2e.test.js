// E2E test for complete login flow through UI
// Tests the critical returning-user-experience journey via actual UI interaction
// This addresses Test Gap #93 - verifies login works end-to-end through the browser
// This is the MOST COMMON user flow (returning users) and was previously untested

const { test, expect } = require('@playwright/test');

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';
const FRONTEND_URL = process.env.FRONTEND_URL || 'http://localhost:3000';

test.describe('MathWizz E2E - Login Flow Through UI', () => {
  test('User can login via UI, get authenticated, and solve a problem', async ({ page }) => {
    // Setup: Create a user via API first (acceptable for login test per Gap #93)
    const testEmail = `logintest${Date.now()}@example.com`;
    const testPassword = 'password123';

    const registerResponse = await fetch(`${API_URL}/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: testEmail, password: testPassword }),
    });
    expect(registerResponse.ok).toBeTruthy();

    // Step 1: Navigate to application (unauthenticated state)
    await page.goto(FRONTEND_URL);

    // Step 2: Verify user starts on login page (default unauthenticated page)
    await expect(page.getByRole('button', { name: 'LOGIN' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'REGISTER' })).toBeVisible();
    await expect(page.getByTestId('submit-button')).toContainText('LOGIN');

    // Step 3: Fill in email and password inputs via UI
    await page.getByTestId('email-input').fill(testEmail);
    await page.getByTestId('password-input').fill(testPassword);

    // Step 4: Click login submit button
    await page.getByTestId('submit-button').click();

    // Step 5: Verify successful login - token stored in localStorage
    // Wait for navigation to solve page (indicates successful login)
    await expect(page.getByRole('button', { name: 'SOLVER' })).toBeVisible({ timeout: 5000 });

    // Step 6: Verify token is stored in localStorage
    const authTokenInStorage = await page.evaluate(() => {
      return localStorage.getItem('authToken');
    });
    expect(authTokenInStorage).not.toBeNull();
    expect(authTokenInStorage).toBeTruthy();

    // Step 7: Verify redirected to solve page
    // Check that authenticated navigation is shown
    await expect(page.getByRole('button', { name: 'SOLVER' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'HISTORY' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'LOGOUT' })).toBeVisible();

    // Verify unauthenticated buttons are NOT shown
    await expect(page.getByRole('button', { name: 'LOGIN' })).not.toBeVisible();
    await expect(page.getByRole('button', { name: 'REGISTER' })).not.toBeVisible();

    // Step 8: Verify authenticated state - can solve a problem
    await page.getByTestId('problem-input').fill('20+30');
    await page.getByTestId('solve-button').click();

    // Verify answer is displayed (proves authentication works and app is functional)
    await expect(page.getByTestId('answer-display')).toContainText('= 50', { timeout: 5000 });
  });

  test('Login with invalid credentials shows error message', async ({ page }) => {
    // Setup: Create a user first
    const testEmail = `loginwrongpw${Date.now()}@example.com`;
    const testPassword = 'correctpassword';

    await fetch(`${API_URL}/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: testEmail, password: testPassword }),
    });

    // Step 1: Navigate to application
    await page.goto(FRONTEND_URL);

    // Step 2: Verify on login page
    await expect(page.getByTestId('submit-button')).toContainText('LOGIN');

    // Step 3: Fill email with correct email but WRONG password
    await page.getByTestId('email-input').fill(testEmail);
    await page.getByTestId('password-input').fill('wrongpassword');

    // Step 4: Click submit
    await page.getByTestId('submit-button').click();

    // Step 5: Verify error message is displayed
    await expect(page.locator('.error-message')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('.error-message')).toContainText('invalid');

    // Verify user is NOT redirected (still on login page)
    await expect(page.getByTestId('submit-button')).toContainText('LOGIN');

    // Verify no token in localStorage
    const authToken = await page.evaluate(() => localStorage.getItem('authToken'));
    expect(authToken).toBeNull();
  });

  test('Login with non-existent email shows error message', async ({ page }) => {
    // Step 1: Navigate to application
    await page.goto(FRONTEND_URL);

    // Step 2: Try to login with email that doesn't exist
    const nonExistentEmail = `doesnotexist${Date.now()}@example.com`;
    await page.getByTestId('email-input').fill(nonExistentEmail);
    await page.getByTestId('password-input').fill('somepassword');

    // Step 3: Click submit
    await page.getByTestId('submit-button').click();

    // Step 4: Verify error message is displayed
    await expect(page.locator('.error-message')).toBeVisible({ timeout: 5000 });

    // Verify user is NOT redirected (still on login page)
    await expect(page.getByTestId('submit-button')).toContainText('LOGIN');

    // Verify no token in localStorage
    const authToken = await page.evaluate(() => localStorage.getItem('authToken'));
    expect(authToken).toBeNull();
  });

  test('Login with invalid email format shows validation error', async ({ page }) => {
    // Step 1: Navigate to application
    await page.goto(FRONTEND_URL);

    // Step 2: Fill invalid email (missing @ symbol)
    await page.getByTestId('email-input').fill('notanemail.com');
    await page.getByTestId('password-input').fill('password123');

    // Step 3: Click submit
    await page.getByTestId('submit-button').click();

    // Step 4: Verify validation error is displayed
    await expect(page.locator('.error-message')).toBeVisible({ timeout: 3000 });
    await expect(page.locator('.error-message')).toContainText('Invalid email');

    // Verify user is NOT redirected (still on login page)
    await expect(page.getByTestId('submit-button')).toContainText('LOGIN');
  });

  test('Login allows navigation to register page', async ({ page }) => {
    // Step 1: Navigate to application (starts on login page)
    await page.goto(FRONTEND_URL);

    // Step 2: Verify on login page
    await expect(page.getByTestId('submit-button')).toContainText('LOGIN');
    await expect(page.getByRole('button', { name: 'LOGIN' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'REGISTER' })).toBeVisible();

    // Step 3: Navigate to register page using REGISTER button
    await page.getByRole('button', { name: 'REGISTER' }).click();

    // Step 4: Verify navigated to register page
    await expect(page.getByTestId('submit-button')).toContainText('CREATE ACCOUNT');

    // Step 5: Navigate back to login page using LOGIN button
    await page.getByRole('button', { name: 'LOGIN' }).click();

    // Step 6: Verify back on login page
    await expect(page.getByTestId('submit-button')).toContainText('LOGIN');
  });

  test('Login with Enter key (form submission) works correctly', async ({ page }) => {
    // Setup: Create a user first
    const testEmail = `enterkey${Date.now()}@example.com`;
    const testPassword = 'password123';

    await fetch(`${API_URL}/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: testEmail, password: testPassword }),
    });

    // Step 1: Navigate to application
    await page.goto(FRONTEND_URL);

    // Step 2: Fill credentials
    await page.getByTestId('email-input').fill(testEmail);
    await page.getByTestId('password-input').fill(testPassword);

    // Step 3: Press Enter key instead of clicking button (form submission)
    await page.getByTestId('password-input').press('Enter');

    // Step 4: Verify successful login
    await expect(page.getByRole('button', { name: 'SOLVER' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('button', { name: 'LOGOUT' })).toBeVisible();

    // Verify token stored
    const authToken = await page.evaluate(() => localStorage.getItem('authToken'));
    expect(authToken).toBeTruthy();
  });

  test('Full user journey: Login → Solve → History → Logout → Login again', async ({ page }) => {
    // Setup: Create a user and solve a problem first (simulating returning user)
    const testEmail = `journey${Date.now()}@example.com`;
    const testPassword = 'password123';

    // Register user
    const registerResponse = await fetch(`${API_URL}/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: testEmail, password: testPassword }),
    });
    const registerData = await registerResponse.json();
    const initialToken = registerData.token;

    // Solve a problem to create history (using API)
    await fetch(`${API_URL}/solve`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${initialToken}`,
      },
      body: JSON.stringify({ problem: '50+50' }),
    });

    // Step 1: Navigate to app (fresh browser session)
    await page.goto(FRONTEND_URL);

    // Step 2: Verify starts on login page
    await expect(page.getByRole('button', { name: 'LOGIN' })).toBeVisible();

    // Step 3: Login via UI
    await page.getByTestId('email-input').fill(testEmail);
    await page.getByTestId('password-input').fill(testPassword);
    await page.getByTestId('submit-button').click();

    // Step 4: Verify redirected to solve page and authenticated
    await expect(page.getByRole('button', { name: 'SOLVER' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('button', { name: 'LOGOUT' })).toBeVisible();

    // Step 5: Solve a NEW problem
    await page.getByTestId('problem-input').fill('25*4');
    await page.getByTestId('solve-button').click();
    await expect(page.getByTestId('answer-display')).toContainText('= 100');

    // Step 6: Navigate to history page
    await page.getByRole('button', { name: 'HISTORY' }).click();

    // Step 7: Verify PREVIOUS problem (50+50) appears in history (eventual consistency)
    // This proves returning users can access their previous history
    let foundPreviousProblem = false;
    const maxRetries = 10;
    const retryInterval = 500;

    for (let i = 0; i < maxRetries; i++) {
      const historyItems = await page.getByTestId('history-item').all();

      for (const item of historyItems) {
        const text = await item.textContent();
        if (text.includes('50+50 = 100')) {
          foundPreviousProblem = true;
          break;
        }
      }

      if (foundPreviousProblem) break;

      await page.waitForTimeout(retryInterval);

      const refreshButton = page.getByTestId('refresh-button');
      if (await refreshButton.isVisible()) {
        await refreshButton.click();
        await page.waitForTimeout(200);
      }
    }

    expect(foundPreviousProblem).toBeTruthy();

    // Step 8: Logout
    await page.getByRole('button', { name: 'LOGOUT' }).click();

    // Step 9: Verify redirected to login page
    await expect(page.getByRole('button', { name: 'LOGIN' })).toBeVisible({ timeout: 3000 });
    await expect(page.getByRole('button', { name: 'REGISTER' })).toBeVisible();

    // Verify token cleared
    const authTokenAfterLogout = await page.evaluate(() => {
      return localStorage.getItem('authToken');
    });
    expect(authTokenAfterLogout).toBeNull();

    // Step 10: Login AGAIN with same credentials
    await page.getByTestId('email-input').fill(testEmail);
    await page.getByTestId('password-input').fill(testPassword);
    await page.getByTestId('submit-button').click();

    // Step 11: Verify successful re-login
    await expect(page.getByRole('button', { name: 'SOLVER' })).toBeVisible({ timeout: 5000 });

    // Step 12: Verify token stored again
    const newAuthToken = await page.evaluate(() => {
      return localStorage.getItem('authToken');
    });
    expect(newAuthToken).toBeTruthy();

    // Step 13: Verify can still access history (persistence across login sessions)
    await page.getByRole('button', { name: 'HISTORY' }).click();

    // Wait for history to load and verify previous problem still exists
    let foundPersisted = false;
    for (let i = 0; i < maxRetries; i++) {
      const historyItems = await page.getByTestId('history-item').all();

      for (const item of historyItems) {
        const text = await item.textContent();
        if (text.includes('50+50 = 100')) {
          foundPersisted = true;
          break;
        }
      }

      if (foundPersisted) break;

      await page.waitForTimeout(retryInterval);

      const refreshButton = page.getByTestId('refresh-button');
      if (await refreshButton.isVisible()) {
        await refreshButton.click();
        await page.waitForTimeout(200);
      }
    }

    expect(foundPersisted).toBeTruthy();
  });
});
