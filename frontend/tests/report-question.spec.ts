import { test, expect } from '@playwright/test';
import { execSync } from 'child_process';

test.describe('Report Question Functionality', () => {
  test.beforeAll(async () => {
    // Reset test database to ensure clean state
    execSync('task reset-test-db', { stdio: 'inherit' });
  });
  test.beforeEach(async ({ page }) => {
    // Navigate to the quiz page
    await page.goto('/quiz');

    // Login as admin user
    await page.fill('[data-testid="username-input"]', 'admin');
    await page.fill('[data-testid="password-input"]', 'password');
    await page.click('[data-testid="login-button"]');

    // Wait for login to complete and redirect to quiz page
    await page.waitForURL('/quiz');
  });

    test('should open report modal when report button is clicked', async ({ page }) => {
    // Wait for a question to be loaded or create one if none exists
    try {
      await page.waitForSelector('[data-testid="question-card"]', { timeout: 5000 });
    } catch {
      // If no question exists, create one by answering a question
      await page.waitForSelector('[data-testid="question-card"]', { timeout: 10000 });
    }

    // Click the report button
    await page.click('[data-testid="report-button"]');

    // Verify the modal is open
    await expect(page.locator('[data-testid="report-modal"]')).toBeVisible();

    // Verify the textarea is present
    await expect(page.locator('[data-testid="report-reason-input"]')).toBeVisible();
  });

  test('should submit report with custom reason', async ({ page }) => {
    // Wait for a question to be loaded
    await page.waitForSelector('[data-testid="question-card"]', { timeout: 10000 });

    // Click the report button
    await page.click('[data-testid="report-button"]');

    // Type a custom reason
    await page.fill('[data-testid="report-reason-input"]', 'This question has incorrect grammar');

    // Submit the report
    await page.click('[data-testid="submit-report-button"]');

    // Verify the modal is closed
    await expect(page.locator('[data-testid="report-modal"]')).not.toBeVisible();

    // Verify success notification (if you have one)
    // await expect(page.locator('[data-testid="success-notification"]')).toBeVisible();
  });

  test('should submit report without reason (uses default)', async ({ page }) => {
    // Wait for a question to be loaded
    await page.waitForSelector('[data-testid="question-card"]', { timeout: 10000 });

    // Click the report button
    await page.click('[data-testid="report-button"]');

    // Submit without entering a reason
    await page.click('[data-testid="submit-report-button"]');

    // Verify the modal is closed
    await expect(page.locator('[data-testid="report-modal"]')).not.toBeVisible();
  });

  test('should cancel report modal', async ({ page }) => {
    // Wait for a question to be loaded
    await page.waitForSelector('[data-testid="question-card"]');

    // Click the report button
    await page.click('[data-testid="report-button"]');

    // Cancel the modal
    await page.click('[data-testid="cancel-report-button"]');

    // Verify the modal is closed
    await expect(page.locator('[data-testid="report-modal"]')).not.toBeVisible();
  });

  test('should handle keyboard shortcuts in report modal', async ({ page }) => {
    // Wait for a question to be loaded
    await page.waitForSelector('[data-testid="question-card"]');

    // Click the report button
    await page.click('[data-testid="report-button"]');

    // Press 'i' to focus the textarea
    await page.keyboard.press('i');

    // Verify the textarea is focused
    await expect(page.locator('[data-testid="report-reason-input"]')).toBeFocused();

    // Type some text
    await page.keyboard.type('Test reason');

    // Press Enter to submit
    await page.keyboard.press('Enter');

    // Verify the modal is closed
    await expect(page.locator('[data-testid="report-modal"]')).not.toBeVisible();
  });

  test('should handle escape key to cancel', async ({ page }) => {
    // Wait for a question to be loaded
    await page.waitForSelector('[data-testid="question-card"]');

    // Click the report button
    await page.click('[data-testid="report-button"]');

    // Press Escape to cancel
    await page.keyboard.press('Escape');

    // Verify the modal is closed
    await expect(page.locator('[data-testid="report-modal"]')).not.toBeVisible();
  });
});
