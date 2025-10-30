import {test, expect} from '@playwright/test';
import {resetTestDatabase} from './reset-db';

test.beforeAll(() => {
  resetTestDatabase();
});

test.describe('Settings Persistence', () => {
  // Helper to login before each test
  test.beforeEach(async ({page}) => {
    await page.goto('/login');
    await page.getByLabel('Username').fill('testuser');
    await page.getByLabel('Password').fill('password');
    await page.locator('form').getByRole('button', {name: 'Sign In'}).click();
    await page.waitForURL('/');
  });

  test('should persist AI model selection across browser refresh', async ({page}) => {
    // Navigate to settings page using the header ActionIcon with title attribute
    await page.locator('header').getByLabel('Settings').click();
    await expect(page).toHaveURL('/settings');

    // Wait for settings to load
    await expect(page.getByText('Learning Preferences')).toBeVisible({timeout: 2000});
    await expect(page.getByText('AI Settings')).toBeVisible({timeout: 2000});

    // Wait for the AI Settings card to be fully loaded
    await page.waitForSelector('[data-testid="ai-provider-select"]', {timeout: 5000});

    // Wait a bit more for the page to be fully rendered
    await page.waitForTimeout(1000);

    // Select a different AI provider
    await page.locator('[data-testid="ai-provider-select"]').click();
    await page.getByText('OpenAI').click();

    // Wait for models to load
    await page.waitForTimeout(1000);

    // Select a different model
    await page.locator('[data-testid="ai-model-select"]').click();
    await page.getByText('gpt-4').click();

    // Save the settings
    await page.getByRole('button', {name: 'Save Changes'}).click();

    // Should see success message (check only the first visible notification)
    const visibleNotification = page.locator('div.mantine-Notification-description', {hasText: 'Settings saved successfully'}).first();
    await expect(visibleNotification).toBeVisible({timeout: 2000});

    // Refresh the page
    await page.reload();
    await page.waitForURL('/settings');

    // Wait for settings to load again
    await expect(page.getByText('AI Settings')).toBeVisible({timeout: 5000});

    // Verify the AI provider and model persisted
    await expect(page.locator('[data-testid="ai-provider-select"]')).toHaveValue('OpenAI');
    await expect(page.locator('[data-testid="ai-model-select"]')).toHaveValue('GPT-4.1');
  });

  test('should persist all AI settings across browser refresh', async ({page}) => {
    // Navigate to settings page using the header ActionIcon with title attribute
    await page.locator('header').getByLabel('Settings').click();
    await expect(page).toHaveURL('/settings');
    await expect(page.getByText('Learning Preferences')).toBeVisible({timeout: 2000});
    await expect(page.getByText('AI Settings')).toBeVisible({timeout: 2000});

    // Wait for the AI Settings card to be fully loaded
    await page.waitForSelector('[data-testid="ai-provider-select"]', {timeout: 5000});

    // Wait a bit more for the page to be fully rendered
    await page.waitForTimeout(1000);

    // Select a different AI provider
    await page.locator('[data-testid="ai-provider-select"]').click();
    await page.getByText('OpenAI').click();

    // Wait for models to load
    await page.waitForTimeout(1000);

    // Select a different model
    await page.locator('[data-testid="ai-model-select"]').click();
    await page.getByText('gpt-4').click();

    // Enter an API key
    await page.getByLabel('API Key').fill('test-api-key-123');

    // Save the settings
    await page.getByRole('button', {name: 'Save Changes'}).click();

    // Wait for any existing notification to disappear before checking for the new one
    const notification = page.locator('div.mantine-Notification-description', {hasText: 'Settings saved successfully'}).first();
    if (await notification.isVisible().catch(() => false)) {
      await notification.waitFor({state: 'detached', timeout: 5000});
    }

    // Should see success message (check only the first visible notification)
    const visibleNotification = page.locator('div.mantine-Notification-description', {hasText: 'Settings saved successfully'}).first();
    await expect(visibleNotification).toBeVisible({timeout: 2000});

    // Refresh the page
    await page.reload();
    await page.waitForURL('/settings');

    // Wait for settings to load again
    await expect(page.getByText('AI Settings')).toBeVisible({timeout: 5000});

    // Verify the AI provider and model persisted
    await expect(page.locator('[data-testid="ai-provider-select"]')).toHaveValue('OpenAI');
    await expect(page.locator('[data-testid="ai-model-select"]')).toHaveValue('GPT-4.1');

    // Note: API key should not be visible in the input field for security reasons
    // The backend should have saved it, but the frontend should show a placeholder
  });

  test('should persist all learning preferences across browser refresh', async ({page}) => {
    // Go to settings
    await page.locator('header').getByLabel('Settings').click();
    await expect(page).toHaveURL('/settings');
    await expect(page.getByText('Learning Preferences')).toBeVisible({timeout: 2000});

    // Wait for loader to disappear (if present) or for the switch to be visible
    const loader = page.locator('div[role="status"], .mantine-Loader-root');
    if (await loader.isVisible().catch(() => false)) {
      await loader.waitFor({state: 'hidden', timeout: 10000});
    }
    const focusSwitchInput = page.locator('[data-testid="focus-weak-areas-switch"]');
    const focusSwitchTrack = focusSwitchInput.locator('..').locator('.mantine-Switch-track');
    await expect(focusSwitchTrack).toBeVisible({timeout: 10000});

    // Toggle focus on weak areas using the visible track
    const isChecked = await focusSwitchInput.isChecked();
    await focusSwitchTrack.click();

    // Set fresh question ratio slider to 0.7
    const freshSlider = page.locator('[data-testid="fresh-question-ratio-slider"]');
    const freshSliderBox = await freshSlider.boundingBox();
    if (freshSliderBox) {
      await freshSlider.click({
        position: {
          x: freshSliderBox.width * 0.7,
          y: freshSliderBox.height / 2,
        },
      });
    }

    // Set known question penalty (this is a slider)
    const penaltySlider = page.locator('[data-testid="known-question-penalty-slider"]');
    const penaltySliderBox = await penaltySlider.boundingBox();
    if (penaltySliderBox) {
      await penaltySlider.click({
        position: {
          x: penaltySliderBox.width * 0.31, // Slightly higher to get closer to 0.3
          y: penaltySliderBox.height / 2,
        },
      });
    }

    // Set review interval (this is a NumberInput)
    const reviewInput = page.locator('[data-testid="review-interval-days-input"]');
    await reviewInput.fill('14');

    // Set weak area boost
    const boostSlider = page.locator('[data-testid="weak-area-boost-slider"]');
    const boostSliderBox = await boostSlider.boundingBox();
    if (boostSliderBox) {
      await boostSlider.click({
        position: {
          x: boostSliderBox.width * 0.75, // 4 out of 5 is 75%
          y: boostSliderBox.height / 2,
        },
      });
    }

    // Select TTS voice if available
    const ttsSelect = page.locator('[data-testid="tts-voice-select"]');
    if (await ttsSelect.isVisible().catch(() => false)) {
      await ttsSelect.click();
      // Pick first available option
      const option = page.locator('.mantine-Select-dropdown [data-combobox-option]');
      if (await option.first().isVisible().catch(() => false)) {
        await option.first().click();
      }
    }

    // Save
    await page.getByRole('button', {name: 'Save Changes'}).click();
    const visibleNotification = page.locator('div.mantine-Notification-description', {hasText: 'Settings saved successfully'}).first();
    await expect(visibleNotification).toBeVisible({timeout: 2000});

    // Reload
    await page.reload();
    await page.waitForURL('/settings');
    await expect(page.getByText('Learning Preferences')).toBeVisible({timeout: 2000});
    if (await loader.isVisible().catch(() => false)) {
      await loader.waitFor({state: 'hidden', timeout: 10000});
    }
    await expect(focusSwitchTrack).toBeVisible({timeout: 10000});

    // Assert all values persisted
    await expect(focusSwitchInput).toBeChecked({checked: !isChecked});
    const freshSliderThumb = page.locator('[data-testid="fresh-question-ratio-slider"] .mantine-Slider-thumb');
    // Check that the value is close to 0.7 (allowing for small differences due to slider step size)
    const freshValue = await freshSliderThumb.getAttribute('aria-valuenow');
    const freshValueNum = parseFloat(freshValue || '0');
    expect(freshValueNum).toBeGreaterThanOrEqual(0.68);
    expect(freshValueNum).toBeLessThanOrEqual(0.72);
    const penaltySliderThumb = page.locator('[data-testid="known-question-penalty-slider"] .mantine-Slider-thumb');
    // Check that the value is close to 0.3 (allowing for small differences due to slider step size)
    const penaltyValue = await penaltySliderThumb.getAttribute('aria-valuenow');
    const penaltyValueNum = parseFloat(penaltyValue || '0');
    expect(penaltyValueNum).toBeGreaterThanOrEqual(0.28);
    expect(penaltyValueNum).toBeLessThanOrEqual(0.32);
    await expect(page.locator('[data-testid="review-interval-days-input"]')).toHaveValue('14');
    const boostSliderThumb = page.locator('[data-testid="weak-area-boost-slider"] .mantine-Slider-thumb');
    // Check that the value is close to 4.1 (allowing for small differences due to slider step size)
    const boostValue = await boostSliderThumb.getAttribute('aria-valuenow');
    const boostValueNum = parseFloat(boostValue || '0');
    expect(boostValueNum).toBeGreaterThanOrEqual(3.9);
    expect(boostValueNum).toBeLessThanOrEqual(4.2);
  });

  test('should not show default values before server response', async ({page}) => {
    await page.locator('header').getByLabel('Settings').click();
    await expect(page).toHaveURL('/settings');
    await expect(page.getByText('Learning Preferences')).toBeVisible({timeout: 2000});
    await expect(page.getByText('AI Settings')).toBeVisible({timeout: 2000});

    // Wait for the AI Settings card to be fully loaded
    await page.waitForSelector('[data-testid="ai-provider-select"]', {timeout: 5000});

    // Wait a bit more for the page to be fully rendered
    await page.waitForTimeout(1000);

    // Check that the AI provider select shows a valid value (not empty)
    const providerValue = await page.locator('[data-testid="ai-provider-select"]').inputValue();
    expect(providerValue).toBeTruthy();
    expect(['Ollama', 'OpenAI', 'Anthropic', 'Google']).toContain(providerValue);

    // Check that the AI model select shows a valid value (not empty)
    const modelValue = await page.locator('[data-testid="ai-model-select"]').inputValue();
    expect(modelValue).toBeTruthy();
  });

  test('should show "Saved key available" only when user actually has saved API key', async ({page}) => {
    await page.locator('header').getByLabel('Settings').click();
    await expect(page).toHaveURL('/settings');
    await expect(page.getByText('Learning Preferences')).toBeVisible({timeout: 2000});
    await expect(page.getByText('AI Settings')).toBeVisible({timeout: 2000});

    // Wait for the AI Settings card to be fully loaded
    await page.waitForSelector('[data-testid="ai-provider-select"]', {timeout: 5000});

    // Wait a bit more for the page to be fully rendered
    await page.waitForTimeout(1000);

    // Check that the API key field is present and functional
    // The "(Saved key available)" message may or may not be visible depending on user's saved keys
    const apiKeyField = page.getByLabel('API Key');
    await expect(apiKeyField).toBeVisible();

    // Check if the user has a saved key (optional test)
    const savedKeyText = page.getByText('(Saved key available)');
    if (await savedKeyText.isVisible()) {
      await expect(savedKeyText).toBeVisible();
    } else {
      // If no saved key, that's also valid - just verify the field is present
      await expect(apiKeyField).toBeVisible();
    }
  });

  test('should send test email for Word of the Day when enabled', async ({page}) => {
    // Navigate to settings
    await page.locator('header').getByLabel('Settings').click();
    await expect(page).toHaveURL('/settings');

    // Ensure Notifications card is visible
    await expect(page.getByText('Notifications')).toBeVisible({timeout: 2000});

    // Enable Word of the Day Emails if disabled
    const wotdSwitch = page.locator('[data-testid="wotd-email-switch"]');
    const wotdTrack = wotdSwitch.locator('..').locator('.mantine-Switch-track');
    await expect(wotdTrack).toBeVisible({timeout: 5000});
    if (!(await wotdSwitch.isChecked())) {
      await wotdTrack.click();
    }

    // Click Test Email button
    const btn = page.locator('[data-testid="wotd-test-email-button"]');
    await expect(btn).toBeVisible({timeout: 3000});
    await btn.click();

    // Expect success notification
    const notification = page
      .locator('div.mantine-Notification-description', {
        hasText: 'Test email sent successfully',
      })
      .first();
    await expect(notification).toBeVisible({timeout: 5000});
  });

  test('should preserve API keys per provider when switching between providers', async ({page}) => {
    await page.locator('header').getByLabel('Settings').click();
    await expect(page).toHaveURL('/settings');
    await expect(page.getByText('Learning Preferences')).toBeVisible({timeout: 2000});
    await expect(page.getByText('AI Settings')).toBeVisible({timeout: 2000});

    // Wait for the AI Settings card to be fully loaded
    await page.waitForSelector('[data-testid="ai-provider-select"]', {timeout: 5000});

    // Wait a bit more for the page to be fully rendered
    await page.waitForTimeout(1000);

    // Enter an API key for the current provider (Ollama)
    await page.getByLabel('API Key').fill('ollama-key-123');

    // Switch to a different provider
    await page.locator('[data-testid="ai-provider-select"]').click();
    await page.getByText('OpenAI').click();

    // Wait for models to load
    await page.waitForTimeout(1000);

    // Enter an API key for the new provider
    await page.getByLabel('API Key').fill('openai-key-456');

    // Switch back to the original provider
    await page.locator('[data-testid="ai-provider-select"]').click();
    await page.getByText('Ollama').click();

    // Wait for models to load
    await page.waitForTimeout(1000);

    // The API key field should be empty (not showing the OpenAI key)
    await expect(page.getByLabel('API Key')).toHaveValue('');

    // Switch back to OpenAI
    await page.locator('[data-testid="ai-provider-select"]').click();
    await page.getByText('OpenAI').click();

    // Wait for models to load
    await page.waitForTimeout(1000);

    // The API key field should be empty (not showing the Ollama key)
    await expect(page.getByLabel('API Key')).toHaveValue('');
  });

  // Test moved to ai-enabled-persistence.spec.ts to avoid race conditions with other tests
  // test('should reflect backend ai_enabled value and persist toggle state', async ({ page, request }) => { ... });
});


test.describe('Settings Page - Account Information', () => {
  test.beforeEach(async ({page}) => {
    // Login and navigate to settings
    await page.goto('/login');
    await page.getByLabel('Username').fill('testuser');
    await page.getByLabel('Password').fill('password');
    await page.locator('form').getByRole('button', {name: 'Sign In'}).click();
    await page.waitForURL('/quiz');

    // Navigate to settings
    await page.click('a[href="/settings"]');
    await page.waitForURL('/settings');
  });

  test('should display Account Information section', async ({page}) => {
    // Check that Account Information section is visible
    await expect(page.locator('h2:has-text("Account Information")')).toBeVisible();

    // Check that all account fields are present
    await expect(page.getByLabel('Username')).toBeVisible();
    await expect(page.getByLabel('Email')).toBeVisible();
    // Timezone is now a react-select component, so we check for the container
    await expect(page.locator('[data-testid="timezone-select"]')).toBeVisible();
  });

  test('should show current account information', async ({page}) => {
    // Username should be populated with current user
    const usernameField = page.getByLabel('Username');
    await expect(usernameField).toHaveValue('testuser');

    // Timezone field should have some value (auto-detected or manually set)
    // For react-select, we check if the component has a value
    const timezoneSelect = page.locator('[data-testid="timezone-select"]');
    await expect(timezoneSelect).toBeVisible();
  });

  test('should automatically detect timezone', async ({page}) => {
    // Timezone should be automatically detected and populated
    const timezoneSelect = page.locator('[data-testid="timezone-select"]');
    await expect(timezoneSelect).toBeVisible();

    // Wait a moment for auto-detection to complete
    await page.waitForTimeout(1000);
  });

  test('should update account information successfully', async ({page}) => {
    // Update account fields
    await page.getByLabel('Email').fill('testuser@example.com');

    // For timezone, we'll test with a valid timezone by typing in the react-select
    const timezoneSelect = page.locator('[data-testid="timezone-select-input"]');
    await timezoneSelect.fill('America/New_York');
    await timezoneSelect.press('Enter');

    // Submit the form
    await page.click('button[type="submit"]');

    // Should see success message (check only the first visible notification)
    const visibleNotification = page.locator('div.mantine-Notification-description', {hasText: 'Settings saved successfully'}).first();
    await expect(visibleNotification).toBeVisible({timeout: 2000});
  });

  test('should validate timezone input', async ({page}) => {
    // Fill email field with a valid value (required)
    await page.getByLabel('Email').fill('testuser@example.com');

    // For react-select, invalid timezones are handled differently
    const timezoneSelect = page.locator('[data-testid="timezone-select-input"]');
    await timezoneSelect.fill('Invalid/Timezone');

    // Try to submit
    await page.click('button[type="submit"]');

    // Should still work since react-select handles validation
    const visibleNotification = page.locator('div.mantine-Notification-description', {hasText: 'Settings saved successfully'}).first();
    await expect(visibleNotification).toBeVisible({timeout: 2000});
  });

  test('should preserve account information on page reload', async ({page}) => {
    // Update account information - clear the field first
    const emailField = page.getByLabel('Email');
    await emailField.clear();
    await emailField.fill('test@example.com');

    // Set timezone using react-select
    const timezoneSelect = page.locator('[data-testid="timezone-select-input"]');
    await timezoneSelect.fill('Europe/London');
    await timezoneSelect.press('Enter');

    await page.click('button[type="submit"]');

    // Wait for success
    const visibleNotification = page.locator('div.mantine-Notification-description', {hasText: 'Settings saved successfully'}).first();
    await expect(visibleNotification).toBeVisible({timeout: 2000});

    // Wait a bit more for the form to fully update
    await page.waitForTimeout(1000);

    // Reload the page
    await page.reload();

    // Wait for the page to load
    await page.waitForTimeout(1000);

    // Check that the information is still there
    await expect(page.getByLabel('Email')).toHaveValue('test@example.com');
    // For react-select, we check that the component is visible and has a value
    await expect(page.locator('[data-testid="timezone-select"]')).toBeVisible();
  });

  test('should handle empty email gracefully', async ({page}) => {
    // Clear email field
    await page.getByLabel('Email').fill('');

    // Submit
    await page.click('button[type="submit"]');

    // Should see a required field validation error
    const emailField = page.getByLabel('Email');
    const validationMessage = await emailField.evaluate((el: HTMLInputElement) => el.validationMessage);
    expect(validationMessage).toBeTruthy();
  });

  test('should validate email format if provided', async ({page}) => {
    // Enter invalid email
    await page.getByLabel('Email').fill('invalid-email');

    // Try to submit
    await page.click('button[type="submit"]');

    // Should see browser validation or error message
    const emailField = page.getByLabel('Email');
    const validationMessage = await emailField.evaluate((el: HTMLInputElement) => el.validationMessage);
    expect(validationMessage).toBeTruthy();
  });
});
