// Playwright global setup - runs before all E2E tests
// Starts backend infrastructure using Docker Compose

const { execSync } = require('child_process');
const path = require('path');

async function globalSetup() {
  console.log('\n🚀 Starting E2E test infrastructure...\n');

  const projectRoot = path.join(__dirname, '..');
  const composeFile = path.join(projectRoot, 'docker-compose.e2e.yml');

  try {
    // Check if Docker/Podman is available
    try {
      execSync('docker --version', { stdio: 'pipe' });
      console.log('✓ Docker is available');
    } catch (error) {
      throw new Error(
        'Docker is not available. Please install Docker or Podman to run E2E tests.\n' +
        'Visit: https://docs.docker.com/get-docker/'
      );
    }

    // Stop and remove any existing E2E containers (cleanup from previous failed runs)
    console.log('Cleaning up any existing E2E containers...');
    try {
      execSync(`docker-compose -f "${composeFile}" down -v`, {
        cwd: projectRoot,
        stdio: 'ignore',
      });
    } catch (error) {
      // Ignore errors - containers might not exist
    }

    // Build Docker images (only if not already built)
    console.log('Building Docker images (this may take a few minutes on first run)...');
    execSync(`docker-compose -f "${composeFile}" build`, {
      cwd: projectRoot,
      stdio: 'inherit',
    });

    // Start all backend services
    console.log('Starting backend services (database, NATS, web-server, history-worker)...');
    execSync(`docker-compose -f "${composeFile}" up -d`, {
      cwd: projectRoot,
      stdio: 'inherit',
    });

    // Wait for services to be healthy
    console.log('Waiting for services to be healthy...');
    const maxRetries = 60; // 60 seconds timeout
    let retries = 0;

    while (retries < maxRetries) {
      try {
        // Check if web-server is healthy
        const result = execSync(`docker-compose -f "${composeFile}" ps --format json`, {
          cwd: projectRoot,
          encoding: 'utf-8',
        });

        // Parse JSON output (one JSON object per line)
        const services = result
          .trim()
          .split('\n')
          .filter(line => line.trim())
          .map(line => JSON.parse(line));

        const webServer = services.find(s => s.Service === 'web-server');

        if (webServer && webServer.Health === 'healthy') {
          console.log('✓ Web server is healthy');
          break;
        }

        retries++;
        await new Promise(resolve => setTimeout(resolve, 1000));
      } catch (error) {
        retries++;
        await new Promise(resolve => setTimeout(resolve, 1000));
      }
    }

    if (retries >= maxRetries) {
      throw new Error('Services failed to become healthy within 60 seconds');
    }

    // Additional wait to ensure history-worker is connected to NATS
    console.log('Waiting for history-worker to connect to NATS...');
    await new Promise(resolve => setTimeout(resolve, 3000));

    console.log('\n✅ All backend services are running and healthy!');
    console.log('📝 Backend available at http://localhost:8080');
    console.log('🗄️  Database available at localhost:5432');
    console.log('📨 NATS available at localhost:4222\n');
    console.log('🎭 Playwright will now start the frontend and run tests...\n');

  } catch (error) {
    console.error('\n❌ Failed to start E2E infrastructure:', error.message);

    // Show container logs on failure
    console.log('\n📋 Container logs:');
    try {
      execSync(`docker-compose -f "${composeFile}" logs --tail=50`, {
        cwd: projectRoot,
        stdio: 'inherit',
      });
    } catch (logError) {
      // Ignore log errors
    }

    // Cleanup on failure
    console.log('\nCleaning up...');
    try {
      execSync(`docker-compose -f "${composeFile}" down -v`, {
        cwd: projectRoot,
        stdio: 'ignore',
      });
    } catch (cleanupError) {
      // Ignore cleanup errors
    }

    throw error;
  }
}

module.exports = globalSetup;
