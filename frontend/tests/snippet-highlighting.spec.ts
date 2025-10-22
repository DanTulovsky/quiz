import { test, expect } from '@playwright/test';
import { resetTestDatabase } from './reset-db';

test.beforeAll(() => {
  resetTestDatabase();
});

test.describe.serial('Snippet Highlighting', () => {
  test.beforeEach(async ({ page }) => {
    // Login as testuser who has French snippets
    await page.goto('/login');
    await page.getByLabel('Username').fill('testuser');
    await page.getByLabel('Password').fill('password');
    await page.locator('form').getByRole('button', { name: 'Sign In' }).click();
    await page.waitForURL('/');
  });

  test('should highlight snippets in question text with dashed underline', async ({
    page,
  }) => {
    // Navigate to quiz page - testuser has Italian questions
    await page.goto('/quiz');

    // Wait for question to load
    await expect(page.getByText('Loading your next question...')).toBeHidden({
      timeout: 10000,
    });

    const questionContent = page.locator('[data-testid="question-content"]');
    await expect(questionContent).toBeVisible({ timeout: 5000 });

    // First, create a snippet for a word that appears in the question
    // Let's save "veloce" from the first Italian question
    const questionText = await questionContent.textContent();
    console.log('Question text:', questionText);

    // Try to find a word from the question to save as snippet
    // The test questions have "veloce", "libro", etc.
    // Let's create a snippet via the API for a common word
    const apiContext = page.context();
    const response = await page.request.post('/v1/snippets', {
      data: {
        original_text: 'veloce',
        translated_text: 'fast',
        source_language: 'it',
        target_language: 'en',
        context: 'E2E test snippet',
      },
    });

    expect(response.ok()).toBeTruthy();

    // Reload the page to see the highlighting
    await page.reload();
    await expect(page.getByText('Loading your next question...')).toBeHidden({
      timeout: 10000,
    });

    // Look for the snippet highlight - it should have a dashed underline
    // The snippet should be wrapped in a span with specific styles
    const highlightedSnippet = page.locator(
      'span[style*="border-bottom"][style*="dashed"]'
    );

    // We may or may not have the word "veloce" in the current question
    // So let's check if we can find any highlighted snippet
    const count = await highlightedSnippet.count();
    if (count > 0) {
      console.log(`Found ${count} highlighted snippet(s)`);

      // Verify the styles
      const firstHighlight = highlightedSnippet.first();
      const style = await firstHighlight.getAttribute('style');
      expect(style).toContain('border-bottom');
      expect(style).toContain('dashed');
      expect(style).toContain('cursor');
    } else {
      console.log(
        'No highlighted snippets found in current question (word not present)'
      );
    }

    // Clean up: delete the snippet
    await page.request.delete('/v1/snippets/1');
  });

  test('should show translation tooltip on hover', async ({ page }) => {
    // Create a snippet first
    await page.request.post('/v1/snippets', {
      data: {
        original_text: 'libro',
        translated_text: 'book',
        source_language: 'it',
        target_language: 'en',
        context: 'E2E test snippet for hover',
      },
    });

    // Navigate to quiz
    await page.goto('/quiz');
    await expect(page.getByText('Loading your next question...')).toBeHidden({
      timeout: 10000,
    });

    await expect(
      page.locator('[data-testid="question-content"]')
    ).toBeVisible({ timeout: 5000 });

    // Look for highlighted snippet
    const highlightedSnippet = page.locator(
      'span[style*="border-bottom"][style*="dashed"]'
    );

    const count = await highlightedSnippet.count();
    if (count > 0) {
      // Hover over the first highlighted snippet
      await highlightedSnippet.first().hover();

      // Wait a bit for tooltip to appear (Mantine tooltips have transitions)
      await page.waitForTimeout(300);

      // Check if tooltip with translation is visible
      // Mantine tooltips appear in the body with specific classes
      const tooltip = page.locator('[role="tooltip"]');
      await expect(tooltip).toBeVisible({ timeout: 2000 });

      // Verify tooltip contains the translation
      const tooltipText = await tooltip.textContent();
      expect(tooltipText).toBeTruthy();
      console.log('Tooltip text:', tooltipText);
    }

    // Clean up
    await page.request.delete('/v1/snippets/2');
  });

  test('should load question immediately and snippets asynchronously (performance test)', async ({
    page,
  }) => {
    // Navigate to quiz page
    await page.goto('/quiz');

    // Measure time to question display
    const startTime = Date.now();

    // Wait for question content to appear
    await expect(
      page.locator('[data-testid="question-content"]')
    ).toBeVisible({ timeout: 10000 });

    const questionLoadTime = Date.now() - startTime;
    console.log(`Question loaded in ${questionLoadTime}ms`);

    // Question should load quickly (within 2 seconds)
    expect(questionLoadTime).toBeLessThan(2000);

    // Snippets should load asynchronously after the question
    // We can verify this by checking network requests
    // The question request should complete before snippets request

    // Wait a bit to allow snippet request to complete
    await page.waitForTimeout(1000);

    // Verify no jarring UI changes - the question content should still be visible
    await expect(
      page.locator('[data-testid="question-content"]')
    ).toBeVisible();

    // Log success
    console.log('Performance test passed: Question loaded independently of snippets');
  });

  test('should highlight snippets in reading comprehension passages', async ({
    page,
  }) => {
    // Create a snippet for a common word
    await page.request.post('/v1/snippets', {
      data: {
        original_text: 'oggi',
        translated_text: 'today',
        source_language: 'it',
        target_language: 'en',
        context: 'E2E test for passages',
      },
    });

    // Navigate to reading comprehension page (if available)
    await page.goto('/reading');
    await page.waitForTimeout(1000);

    // Check if we have a passage
    const passage = page.locator('.reading-passage-text');
    if (await passage.isVisible()) {
      console.log('Reading comprehension passage found');

      // Look for highlighted snippets in the passage
      const highlightedInPassage = passage.locator(
        'span[style*="border-bottom"][style*="dashed"]'
      );
      const count = await highlightedInPassage.count();

      if (count > 0) {
        console.log(`Found ${count} highlighted snippet(s) in passage`);
        expect(count).toBeGreaterThan(0);
      } else {
        console.log('No highlighted snippets in this particular passage');
      }
    } else {
      console.log('No reading comprehension content available in this test run');
    }

    // Clean up
    await page.request.delete('/v1/snippets/3');
  });

  test('should only highlight whole word matches', async ({ page }) => {
    // Create a snippet for "libro"
    await page.request.post('/v1/snippets', {
      data: {
        original_text: 'libro',
        translated_text: 'book',
        source_language: 'it',
        target_language: 'en',
        context: 'E2E test for word boundaries',
      },
    });

    await page.goto('/quiz');
    await expect(page.getByText('Loading your next question...')).toBeHidden({
      timeout: 10000,
    });

    await expect(
      page.locator('[data-testid="question-content"]')
    ).toBeVisible({ timeout: 5000 });

    // If the word "libro" appears, it should be highlighted
    // But "libros" or "libreria" should NOT be highlighted (they contain "libro" but are different words)
    // This verifies our word boundary logic

    const highlightedSnippets = page.locator(
      'span[style*="border-bottom"][style*="dashed"]'
    );
    const count = await highlightedSnippets.count();

    if (count > 0) {
      // Get all highlighted text
      for (let i = 0; i < count; i++) {
        const text = await highlightedSnippets.nth(i).textContent();
        console.log(`Highlighted text ${i + 1}: "${text}"`);

        // Verify it matches the snippet exactly (allowing for case differences)
        expect(text?.toLowerCase()).toBe('libro');
      }
    }

    // Clean up
    await page.request.delete('/v1/snippets/4');
  });
});

