import {defineConfig, devices} from '@playwright/test';
import * as dotenv from 'dotenv';
import * as path from 'path';
import {fileURLToPath} from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Read from default ".env" file.
dotenv.config({path: path.resolve(__dirname, '..', '.env')});

/**
 * Get the base URL for Playwright tests.
 * Uses environment variable or determines based on Docker environment.
 */
function getBaseURL(): string {
  // Check if TEST_BASE_URL is explicitly set
  if (process.env.TEST_BASE_URL) {
    const baseURL = process.env.TEST_BASE_URL;
    console.log(`[Playwright Config] Using TEST_BASE_URL: ${baseURL}`);
    return baseURL;
  }

  // Default to localhost
  const baseURL = 'http://localhost:3001';
  console.log(`[Playwright Config] Using default baseURL: ${baseURL}`);
  return baseURL;
}

/**
 * Playwright configuration for E2E tests against Docker containers.
 * This is used by the Makefile's test-e2e target.
 */
export default defineConfig({
  testDir: './tests',
  /* Run tests in files in parallel */
  fullyParallel: true,
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,
  /* Opt out of parallel tests on CI. */
  workers: 1,
  /* Fail fast on first error */
  maxFailures: 1,
  /* Reporter to use. See https://playwright.dev/docs/test-reporters */
  reporter: 'list',
  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: getBaseURL(),

    /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
    trace: 'on-first-retry',
  },
  /* Pass environment variables to test process */
  env: {
    DATABASE_URL: process.env.DATABASE_URL || 'postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable',
    QUIZ_CONFIG_FILE: process.env.QUIZ_CONFIG_FILE || path.resolve(__dirname, '..', 'merged.config.yaml'),
    MIGRATIONS_PATH: process.env.MIGRATIONS_PATH || 'file://migrations',
  },

  /* Configure projects for major browsers */
  projects: [
    {
      name: 'chromium',
      use: {...devices['Desktop Chrome']},
    },
  ],

  /* No global setup needed - test database is embedded in Docker image */

  /*
   * No webServer configuration here - we expect Docker containers
   * to already be running when this config is used.
   */
});
