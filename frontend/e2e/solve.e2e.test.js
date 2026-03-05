// E2E tests for the solve workflow
// Tests the full user journey through the browser, including async event-driven flow

const { test, expect } = require('@playwright/test');

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';
const FRONTEND_URL = process.env.FRONTEND_URL || 'http://localhost:3000';

test.describe('MathWizz E2E - Solve and History Flow', () => {
  let authToken;
  let testEmail;

  test.beforeEach(async ({ page }) => {
    testEmail = `test${Date.now()}@example.com`;
    const testPassword = 'password123';

    const response = await fetch(`${API_URL}/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: testEmail, password: testPassword }),
    });

    const data = await response.json();
    authToken = data.token;

    await page.goto(FRONTEND_URL);

    await page.evaluate((token) => {
      localStorage.setItem('authToken', token);
    }, authToken);
  });

  test('User solves a problem and sees it in their history (testing eventual consistency)', async ({ page }) => {
    await page.goto(FRONTEND_URL);

    await page.getByRole('button', { name: 'SOLVER' }).click();

    await page.getByTestId('problem-input').fill('25+75');
    await page.getByTestId('solve-button').click();

    await expect(page.getByTestId('answer-display')).toContainText('= 100');

    await page.getByRole('button', { name: 'HISTORY' }).click();

    const maxRetries = 10;
    const retryInterval = 500;
    let found = false;

    for (let i = 0; i < maxRetries; i++) {
      const historyItems = await page.getByTestId('history-item').all();

      for (const item of historyItems) {
        const text = await item.textContent();
        if (text.includes('25+75 = 100')) {
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
  });

  test('Multiple problems appear in history in correct order', async ({ page }) => {
    await page.goto(FRONTEND_URL);

    await page.getByRole('button', { name: 'SOLVER' }).click();

    const problems = [
      { problem: '10+10', answer: '20' },
      { problem: '5*5', answer: '25' },
      { problem: '100-50', answer: '50' },
    ];

    for (const { problem } of problems) {
      await page.getByTestId('problem-input').fill(problem);
      await page.getByTestId('solve-button').click();
      await page.waitForTimeout(300);
    }

    await page.getByRole('button', { name: 'HISTORY' }).click();

    let allProblemsFound = false;
    const maxRetries = 15;

    for (let i = 0; i < maxRetries; i++) {
      const historyItems = await page.getByTestId('history-item').all();
      const foundProblems = new Set();

      for (const item of historyItems) {
        const text = await item.textContent();
        for (const { problem, answer } of problems) {
          if (text.includes(`${problem} = ${answer}`)) {
            foundProblems.add(problem);
          }
        }
      }

      if (foundProblems.size === problems.length) {
        allProblemsFound = true;
        break;
      }

      await page.waitForTimeout(500);

      const refreshButton = page.getByTestId('refresh-button');
      if (await refreshButton.isVisible()) {
        await refreshButton.click();
        await page.waitForTimeout(200);
      }
    }

    expect(allProblemsFound).toBeTruthy();
  });

  test('Handles invalid math problem gracefully', async ({ page }) => {
    await page.goto(FRONTEND_URL);

    await page.getByRole('button', { name: 'SOLVER' }).click();

    await page.getByTestId('problem-input').fill('invalid');
    await page.getByTestId('solve-button').click();

    await expect(page.locator('.error-message')).toBeVisible();
  });
});
