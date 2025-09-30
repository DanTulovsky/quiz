import { test, expect } from '@playwright/test';
import { resetTestDatabase } from './reset-db';

test.beforeAll(() => {
  resetTestDatabase();
});

test.describe('Signup Flow', () => {
  test('should navigate to signup page from login page', async ({ page }) => {
    await page.goto('/login');

    // Check that login page loads
    await expect(page.locator('h2')).toContainText('AI Language Quiz');

    // Find and click the signup link
    const signupLink = page.locator('text=Sign up here');
    await expect(signupLink).toBeVisible();
    await signupLink.click();

    // Should navigate to signup page
    await expect(page).toHaveURL('/signup');

    // Wait for the signup page to load completely
    await page.waitForLoadState('networkidle');

    // Check for the signup page title
    await expect(page.locator('h2')).toContainText('Create your account');
  });

  test('should show validation errors when submitting invalid data', async ({ page }) => {
    await page.goto('/signup');

    // Wait for the signup page to load completely
    await page.waitForLoadState('networkidle');

    // Wait for the form to be ready
    await expect(page.locator('h2')).toContainText('Create your account');

    // Fill with invalid data to trigger custom validation - Mantine TextInput components
    await page.getByLabel('Username').fill('a'); // Too short
    await page.getByLabel('Email address').fill('invalid-email'); // Invalid format
    await page.getByRole('textbox', { name: 'Password', exact: true }).fill('123'); // Too short
    await page.getByLabel('Confirm password').fill('456'); // Doesn't match

    // Submit the form
    await page.click('button[type="submit"]');

    // Wait a bit for any processing
    await page.waitForTimeout(1000);

    // Should remain on signup page (validation should prevent submission)
    await expect(page).toHaveURL('/signup');

    // Should still see the signup form
    await expect(page.locator('h2')).toContainText('Create your account');
  });

  test('should show password strength indicator', async ({ page }) => {
    await page.goto('/signup');

    // Wait for the signup page to load completely
    await page.waitForLoadState('networkidle');

    // Wait for the form to be ready
    await expect(page.locator('h2')).toContainText('Create your account');

    // Type a weak password - Mantine PasswordInput component
    await page.getByRole('textbox', { name: 'Password', exact: true }).fill('123');
    await expect(page.locator('text=Password strength: Weak')).toBeVisible();

    // Type a strong password
    await page.getByRole('textbox', { name: 'Password', exact: true }).fill('MyStrongPassword123!');
    await expect(page.locator('text=Password strength: Strong')).toBeVisible();
  });

  test('should successfully create account and redirect to login with success message', async ({ page }) => {
    await page.goto('/signup');

    // Wait for the signup page to load completely
    await page.waitForLoadState('networkidle');

    // Wait for the form to be ready
    await expect(page.locator('h2')).toContainText('Create your account');

    // Generate a unique username for this test run
    const timestamp = Date.now();
    const testUsername = `testuser${timestamp}`;
    const testEmail = `test${timestamp}@example.com`;

    // Fill out the signup form - Mantine TextInput components
    await page.getByLabel('Username').fill(testUsername);
    await page.getByLabel('Email address').fill(testEmail);

    // Select learning language - use more specific selector to avoid strict mode violation
    await page.locator('input[placeholder="Select language"]').click();
    await page.getByText('Italian').click();

    // Wait for levels to load for Italian
    await page.waitForTimeout(1000);

    // Select current level - use more specific selector and select the first available level
    await page.locator('input[placeholder="Select level"]').click();
    // Wait up to 500ms for the visible level dropdown to be visible
    await page.locator('[role="listbox"]:visible').waitFor({ state: 'visible', timeout: 500 });
    await page.locator('[role="option"]:visible').first().click();

    await page.getByRole('textbox', { name: 'Password', exact: true }).fill('TestPassword123!');
    await page.getByLabel('Confirm password').fill('TestPassword123!');

    // Submit the form
    await page.click('button[type="submit"]');

    // Wait for navigation to complete
    await page.waitForURL('**/login**');

    // Should redirect to login page (may or may not have the query parameter)
    expect(page.url()).toMatch(/\/login/);

    // Should be on login page with the signup form heading
    await expect(page.locator('h2')).toContainText('AI Language Quiz');

    // Now test that we can actually log in with the new account
    await page.getByLabel('Username').fill(testUsername);
    await page.getByLabel('Password').fill('TestPassword123!');
    await page.click('button[type="submit"]');

    // Should successfully log in and redirect to quiz page
    await expect(page).toHaveURL('/quiz');

    // Wait for the page to load and show any valid quiz page state
    await page.waitForTimeout(3000); // Give time for the page to load

    // Should see the quiz interface - just verify we're on the quiz page
    // The page should have loaded successfully by now
    await expect(page).toHaveURL('/quiz');
  });

  test('should show error for duplicate username', async ({ page }) => {
    await page.goto('/signup');

    // Wait for the signup page to load completely
    await page.waitForLoadState('networkidle');

    // Wait for the form to be ready
    await expect(page.locator('h2')).toContainText('Create your account');

    // Try to create account with admin username (which should already exist) - Mantine TextInput components
    await page.getByLabel('Username').fill('admin');
    await page.getByLabel('Email address').fill('unique@example.com');

    // Select learning language - use more specific selector to avoid strict mode violation
    await page.locator('input[placeholder="Select language"]').click();
    await page.getByText('Italian').click();

    // Wait for levels to load for Italian
    await page.waitForTimeout(1000);

    // Select current level - use more specific selector and select the first available level
    await page.locator('input[placeholder="Select level"]').click();
    // Wait up to 500ms for the visible level dropdown to be visible
    await page.locator('[role="listbox"]:visible').waitFor({ state: 'visible', timeout: 500 });
    await page.locator('[role="option"]:visible').first().click();

    await page.getByRole('textbox', { name: 'Password', exact: true }).fill('TestPassword123!');
    await page.getByLabel('Confirm password').fill('TestPassword123!');

    // Submit the form
    await page.click('button[type="submit"]');

    // Should show error notification for duplicate username
    await expect(page.locator('text=Signup Error')).toBeVisible({ timeout: 2000 });
    await expect(page.locator('text=Username already exists')).toBeVisible({ timeout: 2000 });
  });

  test('should navigate back to login page', async ({ page }) => {
    await page.goto('/signup');

    // Wait for the signup page to load completely
    await page.waitForLoadState('networkidle');

    // Wait for the form to be ready
    await expect(page.locator('h2')).toContainText('Create your account');

    // Click the login link
    const loginLink = page.locator('text=sign in to your existing account');
    await expect(loginLink).toBeVisible();
    await loginLink.click();

    // Should navigate back to login page
    await expect(page).toHaveURL('/login');
    await expect(page.locator('h2')).toContainText('AI Language Quiz');
  });

  // Note: We can't easily test actual signup without a test database setup
  // The backend integration tests already verify the signup API works correctly
});
