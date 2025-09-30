import { test, expect } from '@playwright/test';

test.describe('Sticky Question Behavior', () => {
  test.beforeEach(async ({ page }) => {
    // Set viewport to desktop size to ensure sidebar is visible
    await page.setViewportSize({ width: 1280, height: 800 });
    // Login first
    await page.goto('/login');
    await page.getByLabel('Username').fill('testuser');
    await page.getByLabel('Password').fill('password');
    await page.locator('form').getByRole('button', {name: 'Sign In'}).click();
    await page.waitForURL('/');

    // Navigate to the quiz page
    await page.goto('/quiz');

    // Wait for the page to load
    await page.waitForLoadState('networkidle');
  });

  test('should keep question when navigating to settings and back', async ({ page }) => {
    // Wait for a question to be loaded
    await page.waitForSelector('[data-testid="question-content"]', { timeout: 10000 });

    // Get the initial question text
    const initialQuestionText = await page.locator('[data-testid="question-content"]').textContent();
    expect(initialQuestionText).toBeTruthy();

    console.log('Initial question:', initialQuestionText);

    // Navigate to settings by clicking the settings icon in header
    await page.click('[aria-label="Settings"]');
    await page.waitForURL('/settings');

    // Navigate back to quiz by clicking the Quiz link in sidebar
    await page.click('[data-testid="nav-quiz"]');
    await page.waitForURL('/quiz');

    // Wait for the question to be visible again
    await page.waitForSelector('[data-testid="question-content"]', { timeout: 10000 });

    // Check that the same question is still there
    const finalQuestionText = await page.locator('[data-testid="question-content"]').textContent();
    expect(finalQuestionText).toBe(initialQuestionText);

    console.log('Final question:', finalQuestionText);
  });

  test('should keep question when navigating to progress and back', async ({ page }) => {
    // Wait for a question to be loaded
    await page.waitForSelector('[data-testid="question-content"]', { timeout: 10000 });

    // Get the initial question text
    const initialQuestionText = await page.locator('[data-testid="question-content"]').textContent();
    expect(initialQuestionText).toBeTruthy();

    console.log('Initial question:', initialQuestionText);

    // Navigate to progress by clicking the progress icon in header
    await page.click('[aria-label="Progress"]');
    await page.waitForURL('/progress');

    // Navigate back to quiz by clicking the Quiz link in sidebar
    await page.click('[data-testid="nav-quiz"]');
    await page.waitForURL('/quiz');

    // Wait for the question to be visible again
    await page.waitForSelector('[data-testid="question-content"]', { timeout: 10000 });

    // Check that the same question is still there
    const finalQuestionText = await page.locator('[data-testid="question-content"]').textContent();
    expect(finalQuestionText).toBe(initialQuestionText);

    console.log('Final question:', finalQuestionText);
  });

  test('should keep question when switching between Quiz and Reading Comprehension modes', async ({ page }) => {
    // Wait for a quiz question to be loaded
    await page.waitForSelector('[data-testid="question-content"]', { timeout: 10000 });

    // Get the initial quiz question text
    const initialQuizQuestionText = await page.locator('[data-testid="question-content"]').textContent();
    expect(initialQuizQuestionText).toBeTruthy();

    console.log('Initial quiz question:', initialQuizQuestionText);

    // Switch to reading comprehension by clicking the Reading Comprehension link in sidebar
    await page.click('[data-testid="nav-reading"]');
    await page.waitForURL('/reading-comprehension');

    // Wait for a reading comprehension question to be loaded
    await page.waitForSelector('[data-testid="question-content"]', { timeout: 10000 });
    const readingQuestionText = await page.locator('[data-testid="question-content"]').textContent();
    expect(readingQuestionText).toBeTruthy();
    console.log('Reading question:', readingQuestionText);

    // Navigate back to quiz by clicking the Quiz link in sidebar
    await page.click('[data-testid="nav-quiz"]');
    await page.waitForURL('/quiz');

    // Wait for the quiz question to be loaded
    await page.waitForSelector('[data-testid="question-content"]', { timeout: 10000 });
    const finalQuizQuestionText = await page.locator('[data-testid="question-content"]').textContent();
    expect(finalQuizQuestionText).toBeTruthy();
    expect(finalQuizQuestionText).toBe(initialQuizQuestionText);

    console.log('Final quiz question:', finalQuizQuestionText);
  });

  test('should keep question when navigating to settings and back in Reading Comprehension mode', async ({ page }) => {
    // Navigate to reading comprehension first by clicking the Reading Comprehension link
    await page.click('[data-testid="nav-reading"]');
    await page.waitForURL('/reading-comprehension');

    // Wait for a question to be loaded
    await page.waitForSelector('[data-testid="question-content"]', { timeout: 10000 });

    // Get the initial question text
    const initialQuestionText = await page.locator('[data-testid="question-content"]').textContent();
    expect(initialQuestionText).toBeTruthy();

    console.log('Initial reading question:', initialQuestionText);

    // Navigate to settings by clicking the settings icon in header
    await page.click('[aria-label="Settings"]');
    await page.waitForURL('/settings');

    // Navigate back to reading comprehension by clicking the Reading Comprehension link in sidebar
    await page.click('[data-testid="nav-reading"]');
    await page.waitForURL('/reading-comprehension');

    // Wait for the question to be visible again
    await page.waitForSelector('[data-testid="question-content"]', { timeout: 10000 });

    // Check that the same question is still there
    const finalQuestionText = await page.locator('[data-testid="question-content"]').textContent();
    expect(finalQuestionText).toBe(initialQuestionText);

    console.log('Final reading question:', finalQuestionText);
  });

  test('should change questions when switching between Quiz and Reading Comprehension modes', async ({ page }) => {
    // Wait for a quiz question to be loaded
    await page.waitForSelector('[data-testid="question-content"]', { timeout: 10000 });

    // Get the initial quiz question text
    const initialQuizQuestionText = await page.locator('[data-testid="question-content"]').textContent();
    expect(initialQuizQuestionText).toBeTruthy();

    console.log('Initial quiz question:', initialQuizQuestionText);

    // Switch to reading comprehension by clicking the Reading Comprehension link in sidebar
    await page.click('[data-testid="nav-reading"]');
    await page.waitForURL('/reading-comprehension');

    // Wait for a reading comprehension question to be loaded
    await page.waitForSelector('[data-testid="question-content"]', { timeout: 10000 });
    const readingQuestionText = await page.locator('[data-testid="question-content"]').textContent();
    expect(readingQuestionText).toBeTruthy();
    console.log('Reading question:', readingQuestionText);

    // Verify that the reading question is different from the quiz question
    expect(readingQuestionText).not.toBe(initialQuizQuestionText);

    // Switch back to quiz by clicking the Quiz link in sidebar
    await page.click('[data-testid="nav-quiz"]');
    await page.waitForURL('/quiz');

    // Wait for the quiz question to be loaded
    await page.waitForSelector('[data-testid="question-content"]', { timeout: 10000 });
    const finalQuizQuestionText = await page.locator('[data-testid="question-content"]').textContent();
    expect(finalQuizQuestionText).toBeTruthy();

    // Verify that the quiz question is the same as the initial quiz question (sticky within mode)
    expect(finalQuizQuestionText).toBe(initialQuizQuestionText);

    // Verify that the quiz question is different from the reading question
    expect(finalQuizQuestionText).not.toBe(readingQuestionText);

    console.log('Final quiz question:', finalQuizQuestionText);
  });
});
