
import {execSync} from 'child_process';
import * as path from 'path';
import {fileURLToPath} from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

async function globalSetup() {
  console.log('🏗️  Setting up PostgreSQL test database...');

  try {
    // Get the root directory (one level up from frontend)
    const rootDir = path.join(__dirname, '..', '..');

    // Start the test PostgreSQL database container
    console.log('📦 Starting test PostgreSQL database...');
    execSync('docker compose -f docker-compose.test.yml up -d --wait', {
      cwd: rootDir,
      stdio: 'inherit'
    });

    // Wait a moment for the database to be fully ready
    await new Promise(resolve => setTimeout(resolve, 3000));

    // Set up the test database with golden data using PostgreSQL
    console.log('📊 Setting up golden test data...');
    execSync('go run ./cmd/setup-test-db', {
      cwd: path.join(rootDir, 'backend'),
      stdio: 'inherit',
      env: {
        ...process.env,
        DATABASE_URL: 'postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable',
        MIGRATIONS_PATH: 'file://migrations'
      }
    });

    console.log('✅ PostgreSQL test database setup completed');
  } catch (error) {
    console.error('❌ Failed to setup test database:', error);
    throw error;
  }
}

export default globalSetup;
