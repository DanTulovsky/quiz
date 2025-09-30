/**
 * Artillery Processors for Quiz Application Load Testing
 * ====================================================
 *
 * https://www.artillery.io/docs/reference/engines/http#writing-custom-logic-in-js--ts
 *
 * This file contains utility functions for Artillery load testing scenarios.
 * These processors help create realistic test data and manage test state
 * across different load testing scenarios.
 *
 * Key Features:
 * - Generate unique test user credentials from load testing users
 * - Store and retrieve user data across test scenarios
 * - Create admin user accounts for admin-specific testing
 * - Generate CSRF tokens for security testing
 *
 * Usage in Artillery YAML files:
 * - Reference these functions in the 'processor' field
 * - Use in 'function' steps to generate test data
 * - Access generated data in subsequent request steps
 *
 * Example:
 * config:
 *   processor: './artillery/processors.ts'
 * scenarios:
 *   - flow:
 *     - function: "generateRandomUser"
 *     - post:
 *         url: "/v1/auth/login"
 *         json:
 *           username: "{{ username }}"
 *           password: "{{ userPassword }}"
 */

/**
 * Get an existing random user from the load testing pool
 *
 * This function selects from a pool of 100 pre-created load testing users
 * to simulate realistic user behavior in login tests.
 *
 * Features:
 * - Uses 100 pre-created load testing users (loaduser001-loaduser100)
 * - Constructs usernames dynamically instead of hardcoding
 * - Creates predictable test data for consistent testing
 * - Avoids database conflicts with unique user IDs
 *
 * @param context - Artillery context object
 */
async function getExistingRandomUser(context: { vars: Record<string, unknown> }): Promise<void> {
  // Generate random user number (1-100)
  const userNumber = Math.floor(Math.random() * 100) + 1;
  const paddedNumber = userNumber.toString().padStart(3, '0');

  // Construct user data dynamically
  const username = `loaduser${paddedNumber}`;
  const email = `loaduser${paddedNumber}@example.com`;
  const password = 'password';

  // Set variables in Artillery context
  context.vars['userEmail'] = email;
  context.vars['userPassword'] = password;
  context.vars['username'] = username;
}

/**
 * Generate a truly unique random user for signup tests
 *
 * This function creates unique usernames using timestamps and random numbers
 * to avoid conflicts with existing users in the database.
 *
 * Features:
 * - Uses timestamp + random number for unique usernames
 * - Creates unique email addresses
 * - Avoids database conflicts completely
 * - Suitable for signup/registration tests
 *
 * @param context - Artillery context object
 */
async function generateRandomUser(context: { vars: Record<string, unknown> }): Promise<void> {
  // Generate unique username using timestamp and random number
  const timestamp = Date.now();
  const randomNum = Math.floor(Math.random() * 10000);
  const username = `artillery_user_${timestamp}_${randomNum}`;
  const email = `artillery_${timestamp}_${randomNum}@example.com`;
  const password = 'password123';

  // Set variables in Artillery context
  context.vars['userEmail'] = email;
  context.vars['userPassword'] = password;
  context.vars['username'] = username;
}

export { generateRandomUser, getExistingRandomUser };
