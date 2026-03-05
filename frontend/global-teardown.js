// Playwright global teardown - runs after all E2E tests
// Stops and cleans up backend infrastructure

const { execSync } = require('child_process');
const path = require('path');

async function globalTeardown() {
  console.log('\n🧹 Cleaning up E2E test infrastructure...\n');

  const projectRoot = path.join(__dirname, '..');
  const composeFile = path.join(projectRoot, 'docker-compose.e2e.yml');

  try {
    // Stop and remove all containers
    console.log('Stopping backend services...');
    execSync(`docker-compose -f "${composeFile}" down -v`, {
      cwd: projectRoot,
      stdio: 'inherit',
    });

    console.log('\n✅ E2E infrastructure cleaned up successfully!\n');
  } catch (error) {
    console.error('\n⚠️  Warning: Failed to cleanup E2E infrastructure:', error.message);
    console.error('You may need to manually stop containers with:');
    console.error(`  docker-compose -f ${composeFile} down -v\n`);
  }
}

module.exports = globalTeardown;
