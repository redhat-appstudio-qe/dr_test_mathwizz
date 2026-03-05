// E2E test for complete registration flow through UI
// Tests the critical first-user-experience journey via actual UI interaction
// This addresses Test Gap #92 - verifies registration works end-to-end through the browser

const { test, expect } = require('@playwright/test');

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';
const FRONTEND_URL = process.env.FRONTEND_URL || 'http://localhost:3000';

test.describe('MathWizz E2E - Registration Flow Through UI', () => {
  test('User can register via UI, get authenticated, and solve a problem', async ({ page }) => {
    // Generate unique test user credentials
    const testEmail = `test${Date.now()}@example.com`;
    const testPassword = 'password123';

    // Step 1: Navigate to application (unauthenticated state)
    await page.goto(FRONTEND_URL);

    // Verify user starts on login page (default unauthenticated page)
    await expect(page.getByRole('button', { name: 'LOGIN' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'REGISTER' })).toBeVisible();

    // Step 2: Click REGISTER button in navigation
    await page.getByRole('button', { name: 'REGISTER' }).click();

    // Verify navigation to register page
    await expect(page.getByTestId('submit-button')).toContainText('CREATE ACCOUNT');

    // Step 3: Fill in email and password inputs via UI
    await page.getByTestId('email-input').fill(testEmail);
    await page.getByTestId('password-input').fill(testPassword);

    // Step 4: Click register submit button
    await page.getByTestId('submit-button').click();

    // Step 5: Verify successful registration - token stored in localStorage
    // Wait for navigation to solve page (indicates successful registration)
    await expect(page.getByRole('button', { name: 'SOLVER' })).toBeVisible({ timeout: 5000 });

    // Verify token is stored in localStorage (authToken cookie-based system)
    const authTokenInStorage = await page.evaluate(() => {
      return localStorage.getItem('authToken');
    });
    expect(authTokenInStorage).not.toBeNull();
    expect(authTokenInStorage).toBeTruthy();

    // Step 6: Verify redirected to solve page
    // Check that authenticated navigation is shown
    await expect(page.getByRole('button', { name: 'SOLVER' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'HISTORY' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'LOGOUT' })).toBeVisible();

    // Verify unauthenticated buttons are NOT shown
    await expect(page.getByRole('button', { name: 'LOGIN' })).not.toBeVisible();
    await expect(page.getByRole('button', { name: 'REGISTER' })).not.toBeVisible();

    // Step 7: Verify can solve a problem (authenticated and functional)
    // Fill problem input and click solve button
    await page.getByTestId('problem-input').fill('15+35');
    await page.getByTestId('solve-button').click();

    // Verify answer is displayed (proves authentication works and app is functional)
    await expect(page.getByTestId('answer-display')).toContainText('= 50', { timeout: 5000 });
  });

  test('Registration with invalid email shows validation error', async ({ page }) => {
    // Step 1: Navigate to application
    await page.goto(FRONTEND_URL);

    // Step 2: Click REGISTER button
    await page.getByRole('button', { name: 'REGISTER' }).click();

    // Step 3: Fill invalid email (missing @ symbol)
    await page.getByTestId('email-input').fill('notanemail.com');
    await page.getByTestId('password-input').fill('password123');

    // Step 4: Click submit
    await page.getByTestId('submit-button').click();

    // Step 5: Verify validation error is displayed
    await expect(page.locator('.error-message')).toBeVisible({ timeout: 3000 });
    await expect(page.locator('.error-message')).toContainText('Invalid email');

    // Verify user is NOT redirected (still on register page)
    await expect(page.getByTestId('submit-button')).toContainText('CREATE ACCOUNT');
  });

  test('Registration with short password shows validation error', async ({ page }) => {
    // Step 1: Navigate to application
    await page.goto(FRONTEND_URL);

    // Step 2: Click REGISTER button
    await page.getByRole('button', { name: 'REGISTER' }).click();

    // Step 3: Fill valid email but password too short (< 6 characters)
    const testEmail = `test${Date.now()}@example.com`;
    await page.getByTestId('email-input').fill(testEmail);
    await page.getByTestId('password-input').fill('12345'); // 5 characters - too short

    // Step 4: Click submit
    await page.getByTestId('submit-button').click();

    // Step 5: Verify validation error is displayed
    await expect(page.locator('.error-message')).toBeVisible({ timeout: 3000 });
    await expect(page.locator('.error-message')).toContainText('at least 6 characters');

    // Verify user is NOT redirected (still on register page)
    await expect(page.getByTestId('submit-button')).toContainText('CREATE ACCOUNT');
  });

  test('Registration with duplicate email shows server error', async ({ page }) => {
    // Step 1: Create a user via API first (setup)
    const testEmail = `duplicate${Date.now()}@example.com`;
    const testPassword = 'password123';

    await fetch(`${API_URL}/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: testEmail, password: testPassword }),
    });

    // Step 2: Navigate to application UI
    await page.goto(FRONTEND_URL);

    // Step 3: Click REGISTER button
    await page.getByRole('button', { name: 'REGISTER' }).click();

    // Step 4: Try to register with SAME email via UI
    await page.getByTestId('email-input').fill(testEmail);
    await page.getByTestId('password-input').fill(testPassword);
    await page.getByTestId('submit-button').click();

    // Step 5: Verify server error is displayed (duplicate email)
    await expect(page.locator('.error-message')).toBeVisible({ timeout: 5000 });
    // Note: Current implementation returns generic "failed to create user" (not specific 409 Conflict)
    await expect(page.locator('.error-message')).toContainText('failed');

    // Verify user is NOT redirected (still on register page)
    await expect(page.getByTestId('submit-button')).toContainText('CREATE ACCOUNT');
  });

  test('Registration allows navigation back to login page', async ({ page }) => {
    // Step 1: Navigate to application (starts on login page)
    await page.goto(FRONTEND_URL);

    // Step 2: Navigate to register page
    await page.getByRole('button', { name: 'REGISTER' }).click();
    await expect(page.getByTestId('submit-button')).toContainText('CREATE ACCOUNT');

    // Step 3: Navigate back to login page using LOGIN button
    await page.getByRole('button', { name: 'LOGIN' }).click();

    // Step 4: Verify back on login page
    await expect(page.getByTestId('submit-button')).toContainText('LOGIN');
    await expect(page.getByRole('button', { name: 'LOGIN' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'REGISTER' })).toBeVisible();
  });

  test('Full user journey: Register → Solve → View History → Logout', async ({ page }) => {
    // Generate unique test user
    const testEmail = `journey${Date.now()}@example.com`;
    const testPassword = 'password123';

    // Step 1: Navigate to app
    await page.goto(FRONTEND_URL);

    // Step 2: Register new user via UI
    await page.getByRole('button', { name: 'REGISTER' }).click();
    await page.getByTestId('email-input').fill(testEmail);
    await page.getByTestId('password-input').fill(testPassword);
    await page.getByTestId('submit-button').click();

    // Step 3: Verify redirected to solve page and authenticated
    await expect(page.getByRole('button', { name: 'SOLVER' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('button', { name: 'LOGOUT' })).toBeVisible();

    // Step 4: Solve a math problem
    await page.getByTestId('problem-input').fill('100-25');
    await page.getByTestId('solve-button').click();
    await expect(page.getByTestId('answer-display')).toContainText('= 75');

    // Step 5: Navigate to history page
    await page.getByRole('button', { name: 'HISTORY' }).click();

    // Step 6: Verify problem appears in history (eventual consistency)
    // Use retry logic similar to solve.e2e.test.js
    let found = false;
    const maxRetries = 10;
    const retryInterval = 500;

    for (let i = 0; i < maxRetries; i++) {
      const historyItems = await page.getByTestId('history-item').all();

      for (const item of historyItems) {
        const text = await item.textContent();
        if (text.includes('100-25 = 75')) {
          found = true;
          break;
        }
      }

      if (found) break;

      await page.waitForTimeout(retryInterval);

      const refreshButton = page.getByTestId('refresh-button');
      if (await refreshButton.isVisible()) {
        await refreshButton.click();
        await page.waitForTimeout(200);
      }
    }

    expect(found).toBeTruthy();

    // Step 7: Logout
    await page.getByRole('button', { name: 'LOGOUT' }).click();

    // Step 8: Verify redirected to login page (unauthenticated)
    await expect(page.getByRole('button', { name: 'LOGIN' })).toBeVisible({ timeout: 3000 });
    await expect(page.getByRole('button', { name: 'REGISTER' })).toBeVisible();

    // Verify authenticated buttons are gone
    await expect(page.getByRole('button', { name: 'SOLVER' })).not.toBeVisible();
    await expect(page.getByRole('button', { name: 'LOGOUT' })).not.toBeVisible();

    // Verify token is cleared from localStorage
    const authTokenAfterLogout = await page.evaluate(() => {
      return localStorage.getItem('authToken');
    });
    expect(authTokenAfterLogout).toBeNull();
  });
});
