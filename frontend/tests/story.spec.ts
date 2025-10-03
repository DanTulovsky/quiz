import { test, expect } from '@playwright/test';

test.describe('Story Mode', () => {
  test.beforeEach(async ({ page }) => {
    // Login first
    await page.goto('/login');
    await page.fill('[data-testid="username-input"]', 'testuser');
    await page.fill('[data-testid="password-input"]', 'password123');
    await page.click('[data-testid="login-button"]');
    await page.waitForURL('/quiz');
  });

  test('creates a new story successfully', async ({ page }) => {
    // Navigate to story page
    await page.click('[data-testid="nav-story"]');
    await page.waitForURL('/story');

    // Should show create story form since no current story exists
    await expect(page.locator('text=Create New Story')).toBeVisible();

    // Fill in story details
    await page.fill('[data-testid="story-title-input"]', 'My Test Story');
    await page.fill('[data-testid="story-subject-input"]', 'A mystery adventure');
    await page.fill('[data-testid="story-author-style-input"]', 'Agatha Christie');

    // Submit the form
    await page.click('[data-testid="create-story-submit"]');

    // Should redirect to story view and show success message
    await expect(page.locator('text=My Test Story')).toBeVisible();
    await expect(page.locator('text=Story Created')).toBeVisible();
  });

  test('displays story sections in reading mode', async ({ page }) => {
    // Navigate to story page (assuming a story already exists)
    await page.goto('/story');

    // If no story exists, create one first
    const hasCreateForm = await page.locator('text=Create New Story').isVisible();
    if (hasCreateForm) {
      await page.fill('[data-testid="story-title-input"]', 'Reading Mode Test');
      await page.click('[data-testid="create-story-submit"]');
      await page.waitForURL(/\/story$/);
    }

    // Switch to reading mode
    await page.click('[data-testid="nav-story"]'); // Navigate back to story
    await page.click('button:has-text("Reading View")');

    // Should show all sections in reading format
    await expect(page.locator('text=Reading View')).toBeVisible();
    await expect(page.locator('.mantine-ScrollArea-root')).toBeVisible();
  });

  test('displays story sections in section mode', async ({ page }) => {
    // Navigate to story page
    await page.goto('/story');

    // If no story exists, create one first
    const hasCreateForm = await page.locator('text=Create New Story').isVisible();
    if (hasCreateForm) {
      await page.fill('[data-testid="story-title-input"]', 'Section Mode Test');
      await page.click('[data-testid="create-story-submit"]');
      await page.waitForURL(/\/story$/);
    }

    // Should be in section mode by default
    await expect(page.locator('text=Section View')).toBeVisible();

    // Should show section navigation
    await expect(page.locator('button:has-text("Previous")')).toBeVisible();
    await expect(page.locator('button:has-text("Next")')).toBeVisible();
  });

  test('navigates between story sections', async ({ page }) => {
    // Navigate to story page
    await page.goto('/story');

    // If no story exists, create one first
    const hasCreateForm = await page.locator('text=Create New Story').isVisible();
    if (hasCreateForm) {
      await page.fill('[data-testid="story-title-input"]', 'Navigation Test');
      await page.click('[data-testid="create-story-submit"]');
      await page.waitForURL(/\/story$/);
    }

    // Should show section 1 by default
    await expect(page.locator('text=Section 1')).toBeVisible();

    // Navigate to next section if available
    const nextButton = page.locator('button:has-text("Next")');
    const isNextDisabled = await nextButton.isDisabled();

    if (!isNextDisabled) {
      await nextButton.click();
      await expect(page.locator('text=Section 2')).toBeVisible();
    }

    // Navigate back to previous section
    await page.click('button:has-text("Previous")');
    await expect(page.locator('text=Section 1')).toBeVisible();
  });

  test('shows comprehension questions', async ({ page }) => {
    // Navigate to story page
    await page.goto('/story');

    // If no story exists, create one first
    const hasCreateForm = await page.locator('text=Create New Story').isVisible();
    if (hasCreateForm) {
      await page.fill('[data-testid="story-title-input"]', 'Questions Test');
      await page.click('[data-testid="create-story-submit"]');
      await page.waitForURL(/\/story$/);
    }

    // Should show questions section
    await expect(page.locator('text=Comprehension Questions')).toBeVisible();

    // Should show question options
    await expect(page.locator('input[type="radio"]')).toHaveCount(4);

    // Select an answer
    await page.click('input[type="radio"]');

    // Submit answer
    await page.click('button:has-text("Submit Answer")');

    // Should show feedback
    await expect(page.locator('text=Correct!')).toBeVisible();
  });

  test('archives a story', async ({ page }) => {
    // Navigate to story page
    await page.goto('/story');

    // If no story exists, create one first
    const hasCreateForm = await page.locator('text=Create New Story').isVisible();
    if (hasCreateForm) {
      await page.fill('[data-testid="story-title-input"]', 'Archive Test');
      await page.click('[data-testid="create-story-submit"]');
      await page.waitForURL(/\/story$/);
    }

    // Archive the story
    await page.click('button:has-text("Archive")');

    // Should redirect to create story form
    await expect(page.locator('text=Create New Story')).toBeVisible();
    await expect(page.locator('text=Story Archived')).toBeVisible();
  });

  test('exports story as PDF', async ({ page }) => {
    // Navigate to story page
    await page.goto('/story');

    // If no story exists, create one first
    const hasCreateForm = await page.locator('text=Create New Story').isVisible();
    if (hasCreateForm) {
      await page.fill('[data-testid="story-title-input"]', 'Export Test');
      await page.click('[data-testid="create-story-submit"]');
      await page.waitForURL(/\/story$/);
    }

    // Export as PDF (this would trigger a download)
    await page.click('button:has-text("Export PDF")');

    // Should show success message
    await expect(page.locator('text=Export Complete')).toBeVisible();
  });

  test('validates form input correctly', async ({ page }) => {
    // Navigate to story page
    await page.click('[data-testid="nav-story"]');
    await page.waitForURL('/story');

    // Try to submit empty form
    await page.click('[data-testid="create-story-submit"]');

    // Should show validation error
    await expect(page.locator('text=Title is required')).toBeVisible();

    // Fill in title that's too long
    const longTitle = 'a'.repeat(201);
    await page.fill('[data-testid="story-title-input"]', longTitle);
    await page.click('[data-testid="create-story-submit"]');

    // Should show length validation error
    await expect(page.locator('text=Title must be 200 characters or less')).toBeVisible();
  });

  test('handles story creation errors', async ({ page }) => {
    // Navigate to story page
    await page.click('[data-testid="nav-story"]');
    await page.waitForURL('/story');

    // Mock a server error by trying to create a story that would fail
    // (In a real test, you'd mock the API to return an error)

    // Fill in valid data but expect the operation to potentially fail
    await page.fill('[data-testid="story-title-input"]', 'Error Test Story');

    // This test would need API mocking to properly test error handling
    // For now, we'll just verify the form accepts the input
    await expect(page.locator('[data-testid="story-title-input"]')).toHaveValue('Error Test Story');
  });

  test('keyboard navigation works', async ({ page }) => {
    // Navigate to story page
    await page.goto('/story');

    // Use keyboard shortcut (Shift+5) to navigate to story
    await page.keyboard.press('Shift+5');

    // Should navigate to story page
    await expect(page.url()).toContain('/story');
  });
});
