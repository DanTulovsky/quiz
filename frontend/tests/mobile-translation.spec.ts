import { test, expect } from '@playwright/test';

// Mobile-like viewport
test.use({ viewport: { width: 390, height: 844 }, userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.0 Mobile/15E148 Safari/604.1' });

test('translation popup appears on mobile after selecting text', async ({ page }) => {
  await page.goto('/m/daily');

  // Wait for any selectable text region
  const region = page.locator('[data-allow-translate], .selectable-text').first();
  await expect(region).toBeVisible({ timeout: 15000 });

  // Programmatically select a small range of text within the region
  await page.evaluate(() => {
    const container = document.querySelector('[data-allow-translate], .selectable-text');
    if (!container) throw new Error('No selectable region found');
    const walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT);
    let node: Node | null = null;
    while ((node = walker.nextNode())) {
      const text = node.textContent || '';
      if (text.trim().length > 5) {
        const start = Math.max(text.indexOf(' '), 0) + 1;
        const range = document.createRange();
        range.setStart(node, start);
        range.setEnd(node, Math.min(start + 4, text.length));
        const sel = window.getSelection();
        sel?.removeAllRanges();
        sel?.addRange(range);
        document.dispatchEvent(new Event('selectionchange'));
        const touchEnd = new TouchEvent('touchend', { bubbles: true, cancelable: true });
        document.dispatchEvent(touchEnd);
        break;
      }
    }
  });

  // The popup should appear
  await expect(page.locator('.translation-popup')).toBeVisible({ timeout: 10000 });
});


