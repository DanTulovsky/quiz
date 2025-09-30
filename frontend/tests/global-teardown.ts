
import { execSync } from 'child_process';
import * as path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

async function globalTeardown() {
  console.log('🧹 Cleaning up PostgreSQL test database...');

  try {
    // Get the root directory (one level up from frontend)
    const rootDir = path.join(__dirname, '..', '..');

    // Stop and remove the test PostgreSQL database container
    execSync('docker compose -f docker-compose.test.yml down -v', {
      cwd: rootDir,
      stdio: 'inherit'
    });

    console.log('✅ PostgreSQL test database cleanup completed');
  } catch (error) {
    console.warn('⚠️  Failed to cleanup test database (this might be okay):', error);
    // Don't throw error here as cleanup failures shouldn't fail the test run
  }
}

export default globalTeardown;
