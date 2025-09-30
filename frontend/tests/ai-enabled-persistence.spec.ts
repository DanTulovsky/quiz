import {test, expect} from '@playwright/test';
import {resetTestDatabase} from './reset-db';

test.beforeAll(() => {
  resetTestDatabase();
});

test.beforeEach(async ({page}) => {
  page.on('requestfailed', request => {
    console.log(`❌ REQUEST FAILED: ${request.method()} ${request.url()} - ${request.failure()?.errorText}`);
  });

  await page.goto('/login');
  await page.getByLabel('Username').fill('adminuser');
  await page.getByLabel('Password').fill('password');
  await page.locator('form').getByRole('button', {name: 'Sign In'}).click();
  await page.waitForURL('/');
});

test('should persist AI enabled toggle state across page refresh', async ({page}) => {
  // Navigate to settings page using the header ActionIcon with title attribute
  await page.locator('header').getByLabel('Settings').click();
  await expect(page).toHaveURL('/settings');

  // Wait for settings to load and ensure AI Settings section is visible
  await expect(page.getByText('AI Settings')).toBeVisible({timeout: 5000});

  // Wait for the AI Settings card to be fully loaded
  await page.waitForSelector('[data-testid="ai-provider-select"]', {timeout: 5000});

  // Wait a bit more for the page to be fully rendered
  await page.waitForTimeout(1000);

  // Debug: Check if the AI toggle exists and its state
  const aiToggle = page.locator('[role="switch"]');
  const toggleCount = await aiToggle.count();

  if (toggleCount > 0) {
    // Remove unused variables isVisible, isHidden, and ariaChecked
  }

  // Try to find the toggle by looking for the Switch component in the AI Settings section
  // The Switch is inside a Card with "Enable AI Features" text
  // Look for the visible part of the Switch (not the hidden input)
  let aiToggleSwitch = page.getByTestId('ai-enabled-switch');
  if (!(await aiToggleSwitch.count())) {
    // Fallback: try to find any switch-like element near the label
    aiToggleSwitch = page.locator('[role="switch"], .mantine-Switch-root, .mantine-Switch-track');
  }

  // Check the visible switch track instead of the hidden input
  const aiToggleSwitchTrack = page.getByTestId('ai-enabled-switch').locator('..').locator('.mantine-Switch-track');
  await expect(aiToggleSwitchTrack).toBeVisible({timeout: 2000});

  // Get initial state from aria-checked (on the hidden input)
  const hiddenInput = page.getByTestId('ai-enabled-switch');
  const initialChecked = await hiddenInput.getAttribute('aria-checked');

  // Toggle the AI enabled switch
  await aiToggleSwitch.locator('..').locator('.mantine-Switch-track').click();
  await expect(aiToggleSwitchTrack).toBeVisible();

  // Wait for the state to change
  await page.waitForTimeout(500);

  // Verify the state changed
  const newChecked = await hiddenInput.getAttribute('aria-checked');
  expect(newChecked).not.toBe(initialChecked);

  // Save the settings
  await page.getByRole('button', {name: 'Save Changes'}).click();

  // Wait for success message
  const visibleNotification = page.locator('div.mantine-Notification-description', {hasText: 'Settings saved successfully'}).first();
  await expect(visibleNotification).toBeVisible({timeout: 2000});

  // Refresh the page
  await page.reload();
  await page.waitForURL('/settings');

  // Wait for settings to load again
  await expect(page.getByText('AI Settings')).toBeVisible({timeout: 5000});

  // Verify the toggle state persisted
  const persistedToggle = page.getByTestId('ai-enabled-switch');
  await expect(persistedToggle.locator('..').locator('.mantine-Switch-track')).toBeVisible({timeout: 2000});
  const persistedHiddenInput = page.getByTestId('ai-enabled-switch');
  const persistedChecked = await persistedHiddenInput.getAttribute('aria-checked');
  expect(persistedChecked).toBe(newChecked);

  // Verify the toggle state persisted by checking the frontend state after refresh
  // The toggle should maintain its state after the page refresh
  const finalToggle = page.getByTestId('ai-enabled-switch');
  const finalToggleTrack = finalToggle.locator('..').locator('.mantine-Switch-track');
  await expect(finalToggleTrack).toBeVisible({timeout: 2000});
  const finalHiddenInput = page.getByTestId('ai-enabled-switch');
  const finalChecked = await finalHiddenInput.getAttribute('aria-checked');
  expect(finalChecked).toBe(newChecked);

  // Toggle the switch back (click the visible track again)
  await aiToggleSwitch.locator('..').locator('.mantine-Switch-track').click();
});
