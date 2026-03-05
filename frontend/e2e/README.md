# E2E Testing for MathWizz

This directory contains End-to-End (E2E) tests for the MathWizz application using [Playwright](https://playwright.dev/).

## Overview

E2E tests verify complete user journeys through the browser, testing the entire application stack:
- Frontend (React)
- Backend API (Go web-server)
- Database (PostgreSQL)
- Message Queue (NATS)
- History Worker (Go worker)

## Test Files

- `login.e2e.test.js` - Login flow through UI (7 tests)
- `register.e2e.test.js` - Registration flow through UI (6 tests)
- `solve.e2e.test.js` - Problem solving and history flow (3 tests)

## Running E2E Tests

### Prerequisites

1. **Docker or Podman** must be installed
   - Docker: https://docs.docker.com/get-docker/
   - Podman: https://podman.io/getting-started/installation

2. **Node.js and npm** (for frontend)

### Quick Start

From the `frontend/` directory:

```bash
# Run all E2E tests (headless)
npm run e2e

# Run with visible browser
npm run e2e:headed

# Run with Playwright UI (interactive mode)
npm run e2e:ui

# Run in debug mode (step through tests)
npm run e2e:debug
```

### What Happens When You Run E2E Tests

1. **Global Setup** (`global-setup.js`):
   - Builds Docker images for all services
   - Starts backend infrastructure using `docker-compose.e2e.yml`:
     - PostgreSQL database on port 5432
     - NATS message queue on port 4222
     - Web-server API on port 8080
     - History worker (background service)
   - Waits for all services to be healthy (~30-60 seconds on first run)

2. **Playwright Web Server**:
   - Starts React frontend on port 3000 (auto-managed by Playwright)

3. **Test Execution**:
   - Runs all E2E tests in `e2e/` directory
   - Tests interact with real services through the browser

4. **Global Teardown** (`global-teardown.js`):
   - Stops and removes all Docker containers
   - Cleans up volumes and networks

### Manual Infrastructure Management

If you need to manually control the infrastructure:

```bash
# Start backend services (from MathWizz root directory)
docker-compose -f docker-compose.e2e.yml up -d

# View logs
docker-compose -f docker-compose.e2e.yml logs -f

# Check service health
docker-compose -f docker-compose.e2e.yml ps

# Stop services
docker-compose -f docker-compose.e2e.yml down -v
```

## Test Structure

E2E tests follow Playwright best practices:

```javascript
test('User can login and solve a problem', async ({ page }) => {
  // Navigate to app
  await page.goto('http://localhost:3000');

  // Fill login form
  await page.getByTestId('email-input').fill('test@example.com');
  await page.getByTestId('password-input').fill('password123');
  await page.getByTestId('submit-button').click();

  // Verify redirect
  await expect(page.getByRole('button', { name: 'SOLVER' })).toBeVisible();

  // Solve a problem
  await page.getByTestId('problem-input').fill('2+2');
  await page.getByTestId('solve-button').click();

  // Verify answer
  await expect(page.getByTestId('answer-display')).toContainText('= 4');
});
```

## Test Data

E2E tests use:
- **Dynamic test users**: Generated with unique emails (`test${Date.now()}@example.com`)
- **Isolated database**: Each test run uses a fresh database
- **Eventual consistency testing**: Tests handle async event processing with retry logic

## Troubleshooting

### Tests fail with "connection refused"

Services may not be fully started. Check service health:

```bash
docker-compose -f docker-compose.e2e.yml ps
docker-compose -f docker-compose.e2e.yml logs web-server
```

### Tests are slow on first run

Docker images are being built. Subsequent runs will be faster as images are cached.

### Port conflicts

If ports 3000, 4222, 5432, or 8080 are already in use:

```bash
# Check what's using the port
lsof -i :8080

# Stop conflicting services or change ports in docker-compose.e2e.yml
```

### Cleanup after failed test run

If tests fail and containers aren't cleaned up:

```bash
docker-compose -f docker-compose.e2e.yml down -v
```

## CI/CD Integration

In CI environments, set `CI=true` environment variable:
- Playwright will retry failed tests (2 retries)
- Tests run with 1 worker (sequential)
- No browser UI

Example GitHub Actions:

```yaml
- name: Run E2E tests
  run: |
    cd frontend
    npm run e2e
  env:
    CI: true
```

## Performance Notes

- First run: ~2-3 minutes (builds images)
- Subsequent runs: ~30-60 seconds (uses cached images)
- Individual test: 5-10 seconds

## Architecture

```
┌─────────────────────┐
│  Browser (Playwright)│
│    Port 3000        │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  React Frontend     │
│  (npm start)        │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐      ┌─────────────────┐
│  Web Server (Go)    │◄─────┤  NATS Queue     │
│  Port 8080          │      │  Port 4222      │
└──────────┬──────────┘      └────────┬────────┘
           │                          │
           ▼                          ▼
┌─────────────────────┐      ┌─────────────────┐
│  PostgreSQL DB      │◄─────┤ History Worker  │
│  Port 5432          │      │  (Go)           │
└─────────────────────┘      └─────────────────┘
```

All services run in Docker containers, orchestrated by docker-compose.e2e.yml.
