import {test, expect} from '@playwright/test';
import {resetTestDatabase} from './reset-db';

test.beforeAll(() => {
  resetTestDatabase();
});

test.describe('Authentication', () => {
  test('should allow a user to log in with golden test data', async ({page}) => {
    // Navigate to the login page
    await page.goto('/login');
    await expect(page).toHaveTitle(/Quiz/);

    // Fill in the login form using golden test data
    await page.getByLabel('Username').fill('testuser');
    await page.getByLabel('Password').fill('password'); // Using the password that matches the bcrypt hash in database

    // Click the login button - use the submit button specifically
    await page.locator('form').getByRole('button', {name: 'Sign In'}).click();

    // Wait for navigation and assert the URL - should now go to /quiz
    await page.waitForURL('/quiz');
    await expect(page).toHaveURL('/quiz');

    // Wait for the loading indicator to disappear first
    await expect(page.getByText('Loading your next question...')).toBeHidden();

    // Check that the main quiz card is visible - Mantine Card component
    await expect(page.locator('[data-testid="question-content"]')).toBeVisible();
  });

  test('should reject invalid credentials', async ({page}) => {
    await page.goto('/login');

    // Try with invalid credentials
    await page.getByLabel('Username').fill('invaliduser');
    await page.getByLabel('Password').fill('wrongpassword');
    await page.locator('form').getByRole('button', {name: 'Sign In'}).click();

    // Should stay on login page and show error
    await expect(page).toHaveURL('/login');
    // Note: Add error message expectations once backend provides them
  });

  test('should allow admin user login', async ({page}) => {
    await page.goto('/login');

    // Login with admin user from golden data
    await page.getByLabel('Username').fill('adminuser');
    await page.getByLabel('Password').fill('password');
    await page.locator('form').getByRole('button', {name: 'Sign In'}).click();

    // Should redirect to quiz page
    await page.waitForURL('/quiz');
    await expect(page).toHaveURL('/quiz');

    // Check user context shows admin user
    await expect(page.getByText('adminuser')).toBeVisible();
  });
});

test.describe('Signup Disable Feature', () => {
  test('should hide signup link when signups are disabled', async ({page}) => {
    // Mock the signup status API to return disabled
    await page.route('/v1/auth/signup/status', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({signups_disabled: true})
      });
    });

    await page.goto('/login');

    // Wait for the page to load
    await page.waitForLoadState('networkidle');

    // The signup link should not be visible
    await expect(page.locator('text=Sign up here')).not.toBeVisible();

    // The "Don't have an account?" text should not be visible
    await expect(page.locator('text=Don\'t have an account?')).not.toBeVisible();
  });

  test('should show signup link when signups are enabled', async ({page}) => {
    // Mock the signup status API to return enabled
    await page.route('/v1/auth/signup/status', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({signups_disabled: false})
      });
    });

    await page.goto('/login');

    // Wait for the page to load
    await page.waitForLoadState('networkidle');

    // The signup link should be visible
    await expect(page.locator('text=Sign up here')).toBeVisible();

    // The "Don't have an account?" text should be visible
    await expect(page.locator('text=Don\'t have an account?')).toBeVisible();
  });

  test('should show disabled message on signup page when signups are disabled', async ({page}) => {
    // Mock the signup status API to return disabled
    await page.route('/v1/auth/signup/status', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({signups_disabled: true})
      });
    });

    await page.goto('/signup');

    // Wait for the page to load
    await page.waitForLoadState('networkidle');

    // Should show the disabled message
    await expect(page.locator('text=Signups Disabled')).toBeVisible();
    await expect(page.locator('text=User registration is currently disabled')).toBeVisible();
    await expect(page.locator('text=Please contact an administrator if you need access')).toBeVisible();

    // Should show return to login link
    await expect(page.locator('text=Return to login')).toBeVisible();
  });

  test('should allow navigation back to login from disabled signup page', async ({page}) => {
    // Mock the signup status API to return disabled
    await page.route('/v1/auth/signup/status', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({signups_disabled: true})
      });
    });

    await page.goto('/signup');

    // Wait for the page to load
    await page.waitForLoadState('networkidle');

    // Click the return to login link
    await page.locator('text=Return to login').click();

    // Should navigate back to login page
    await expect(page).toHaveURL('/login');
    await expect(page.locator('h2')).toContainText('Language Quiz');
  });

  test('should show normal signup form when signups are enabled', async ({page}) => {
    // Mock the signup status API to return enabled
    await page.route('/v1/auth/signup/status', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({signups_disabled: false})
      });
    });

    await page.goto('/signup');

    // Wait for the page to load
    await page.waitForLoadState('networkidle');

    // Should show the normal signup form
    await expect(page.locator('h2')).toContainText('Create your account');
    await expect(page.getByLabel('Username')).toBeVisible();
    await expect(page.getByLabel('Email address')).toBeVisible();
    await expect(page.getByRole('button', {name: 'Create Account'})).toBeVisible();
  });
});
