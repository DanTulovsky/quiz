import {test, expect} from '@playwright/test';
import {execSync} from 'child_process';

test.describe('Admin AI Fix flow', () => {
    test.beforeAll(() => {
        // reset test DB to a known state
        execSync('task reset-test-db', {stdio: 'inherit'});
    });

    test.beforeEach(async ({page}) => {
        // login as admin
        await page.goto('/login');
        await page.fill('[data-testid="username-input"]', 'adminuser');
        await page.fill('[data-testid="password-input"]', 'password');
        await page.locator('form').getByRole('button', {name: 'Sign In'}).click();
        await page.waitForURL('/');
    });

    test('opens AI Fix modal and applies suggestion', async ({page}) => {
        // navigate to admin data explorer
        await page.goto('/admin/backend/data-explorer');
        await page.waitForSelector('text=Reported Questions', {timeout: 10000});

        // click the AI Fix button for the first reported question
        await page.locator('button', {hasText: 'AI Fix'}).first().click();

        // modal should open and show loading message or content
        await expect(page.locator('text=AI Suggestion')).toBeVisible({timeout: 5000});

        // wait for suggestion to load
        await page.waitForSelector('text=Apply Suggestion', {timeout: 15000});

        // click Apply Suggestion
        await page.locator('button', {hasText: 'Apply Suggestion'}).click();

        // expect success notification
        const notification = page.locator('div.mantine-Notification-description', {hasText: 'AI suggestion applied'}).first();
        await expect(notification).toBeVisible({timeout: 10000});
    });
});


