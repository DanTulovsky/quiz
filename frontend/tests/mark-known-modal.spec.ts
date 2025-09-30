import {test, expect} from '@playwright/test';

test.describe('Mark Known Modal', () => {
  // Helper to login before each test using the existing admin user
  test.beforeEach(async ({page}) => {
    await page.goto('/login');
    await page.getByLabel('Username').fill('admin');
    await page.getByLabel('Password').fill('password');
    await page.locator('form').getByRole('button', {name: 'Sign In'}).click();
    await page.waitForURL('/');
  });

  test('should open mark known modal when clicking "I know this" button', async ({page}) => {
    // Navigate to the quiz page
    await page.goto('/quiz');

    // Wait for the page to load
    await expect(page.getByText('Loading your next question...')).toBeHidden();

    // Wait for one of the expected states to appear
    await expect(async () => {
      const hasQuizCard = await page.locator('[data-testid="question-content"]').isVisible();
      const hasGenerationMessage = await page.getByText(/generating/i).isVisible();
      const hasErrorMessage = await page.getByText(/No question(s)? available\./i).isVisible();

      // At least one of these should be true
      expect(hasQuizCard || hasGenerationMessage || hasErrorMessage).toBe(true);
    }).toPass({timeout: 2000});

    // Only proceed if we have a question available
    const questionContent = page.locator('[data-testid="question-content"]');
    if (await questionContent.isVisible()) {
      // Click the "I know this" button
      await page.locator('[data-testid="mark-known-btn"]').click();

      // Modal should be visible
      await expect(page.getByText('Adjust Question Frequency')).toBeVisible();

      // Should show confidence level options with icons
      await expect(page.locator('[data-testid="confidence-level-1"]')).toBeVisible();
      await expect(page.locator('[data-testid="confidence-level-2"]')).toBeVisible();
      await expect(page.locator('[data-testid="confidence-level-3"]')).toBeVisible();
      await expect(page.locator('[data-testid="confidence-level-4"]')).toBeVisible();
      await expect(page.locator('[data-testid="confidence-level-5"]')).toBeVisible();
    }
  });
});
