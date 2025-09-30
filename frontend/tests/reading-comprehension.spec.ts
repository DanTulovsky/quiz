import {test, expect} from '@playwright/test';
import {resetTestDatabase} from './reset-db';

test.beforeAll(() => {
    resetTestDatabase();
});

test.describe('Reading Comprehension', () => {
    // Helper to login before each test
    test.beforeEach(async ({page}) => {
        await page.goto('/login');
        await expect(page.getByLabel('Username')).toBeVisible({timeout: 5000});
        await page.getByLabel('Username').fill('testuser');
        await page.getByLabel('Password').fill('password');
        await page.locator('form').getByRole('button', {name: 'Sign In'}).click();
        await page.waitForURL('/');
    });

    test('should show Reading Comprehension link in navigation', async ({page}) => {
        // Wait for the page to be fully loaded and navigation to be available
        await expect(page.getByText('Loading your next question...')).toBeHidden();

        // Check that the Reading Comprehension link is visible in the navigation
        const readingComprehensionLink = page.getByRole('link', {name: 'Reading Comprehension'});
        await expect(readingComprehensionLink).toBeVisible();
    });

    test('should navigate to Reading Comprehension page', async ({page}) => {
        // Wait for the page to be fully loaded
        await expect(page.getByText('Loading your next question...')).toBeHidden();

        // Click on the Reading Comprehension link
        await page.getByRole('link', {name: 'Reading Comprehension'}).click();

        // Should navigate to reading comprehension page
        await expect(page).toHaveURL('/reading-comprehension');

        // Should show the reading comprehension page content
        await expect(page.getByText('Loading your next question...')).toBeHidden();

        // Wait for either a question to appear or generation to complete
        await expect(page.getByText('Generating your personalized question...')).toBeHidden({timeout: 15000});

        // Check if we have a question available or an error state
        const questionContent = page.locator('[data-testid="question-content"]');
        const errorText = page.locator('p').filter({hasText: 'No questions available'});
        const tryAgainButton = page.getByRole('button', {name: 'Try Again'});

        // At least one of these should be visible
        await expect(async () => {
            const hasQuestion = await questionContent.isVisible();
            const hasError = await errorText.isVisible();
            const hasTryAgain = await tryAgainButton.isVisible();
            expect(hasQuestion || hasError || hasTryAgain).toBe(true);
        }).toPass({timeout: 5000});
    });

    test('should load reading comprehension questions only', async ({page}) => {
        // Navigate to reading comprehension page
        await page.getByRole('link', {name: 'Reading Comprehension'}).click();
        await expect(page).toHaveURL('/reading-comprehension');

        // Wait for loading to complete
        await expect(page.getByText('Loading your next question...')).toBeHidden();
        await expect(page.getByText('Generating your personalized question...')).toBeHidden({timeout: 15000});

        // Check if we have a question available
        const questionContent = page.locator('[data-testid="question-content"]');
        if (await questionContent.isVisible()) {
            // The question should be a reading comprehension question
            // We can verify this by checking for the passage field in the question content
            const questionText = await questionContent.textContent();
            expect(questionText).toBeTruthy();

            // Reading comprehension questions should have a passage
            // The passage should be longer than a typical vocabulary question
            if (questionText) {
                // Reading comprehension questions should have a passage displayed
                // Look for the passage content which should be much longer than the question
                const passageElement = page.locator('.reading-passage-text');
                await expect(passageElement).toBeVisible();

                const passageText = await passageElement.textContent();
                expect(passageText?.length).toBeGreaterThan(100);

                // Verify it contains passage-like content (should be a substantial text)
                // Since we now have multiple reading comprehension questions, we'll check for
                // common Italian words that would appear in any of our passages
                const hasItalianContent = passageText && (
                    passageText.includes('è') ||
                    passageText.includes('del') ||
                    passageText.includes('della') ||
                    passageText.includes('che') ||
                    passageText.includes('con')
                );
                expect(hasItalianContent).toBe(true);
            }
        }
    });

    test('should handle reading comprehension question interaction', async ({page}) => {
        // Navigate to reading comprehension page
        await page.getByRole('link', {name: 'Reading Comprehension'}).click();
        await expect(page).toHaveURL('/reading-comprehension');

        // Wait for loading to complete
        await expect(page.getByText('Loading your next question...')).toBeHidden();
        await expect(page.getByText('Generating your personalized question...')).toBeHidden({timeout: 15000});

        // Check if we have a question available
        const questionContent = page.locator('[data-testid="question-content"]');
        if (await questionContent.isVisible()) {
            // All questions use multiple choice radio buttons
            const multipleChoiceOption = page.locator('input[type="radio"]').first();
            const submitButton = page.getByRole('button', {name: 'Submit'});

            if (await multipleChoiceOption.isVisible()) {
                // Click the first radio button option
                await multipleChoiceOption.click();
                // Wait for the submit button to be enabled
                await expect(submitButton).toBeEnabled({timeout: 5000});
                await submitButton.click();

                // Should show feedback after submission
                // Look for the feedback alert within the question card specifically
                // The alert has a title that says "Correct!" or "Incorrect"
                const questionCard = page.locator('[data-testid="question-card"]');
                await expect(questionCard.locator('.mantine-Alert-title')).toBeVisible({timeout: 5000});
            }
        }
    });

    test('should show next question button after answering reading comprehension', async ({page}) => {
        // Navigate to reading comprehension page
        await page.getByRole('link', {name: 'Reading Comprehension'}).click();
        await expect(page).toHaveURL('/reading-comprehension');

        // Wait for loading to complete
        await expect(page.getByText('Loading your next question...')).toBeHidden();
        await expect(page.getByText('Generating your personalized question...')).toBeHidden({timeout: 15000});

        const questionContent = page.locator('[data-testid="question-content"]');
        if (await questionContent.isVisible()) {
            // All questions use multiple choice radio buttons
            const multipleChoiceOption = page.locator('input[type="radio"]').first();
            const submitButton = page.getByRole('button', {name: 'Submit'});

            if (await multipleChoiceOption.isVisible()) {
                await multipleChoiceOption.click();
                await expect(submitButton).toBeEnabled({timeout: 5000});
                await submitButton.click();

                // Should show next question button after feedback
                const nextButton = page.getByRole('button', {name: 'Next Question'});
                await expect(nextButton).toBeVisible({timeout: 2000});

                await nextButton.click();
                // Should show loading again
                await expect(async () => {
                    const hasLoadingText = await page.getByText('Loading your next question...').isVisible();
                    const hasGeneratingText = await page.getByText('Generating your personalized question...').isVisible();
                    const hasTransitioningState = await page.locator('[data-testid="question-content"]').isVisible() === false;
                    expect(hasLoadingText || hasGeneratingText || hasTransitioningState).toBe(true);
                }).toPass({timeout: 5000});
            }
        }
    });

    test('should handle new question button on reading comprehension page', async ({page}) => {
        // Navigate to reading comprehension page
        await page.getByRole('link', {name: 'Reading Comprehension'}).click();
        await expect(page).toHaveURL('/reading-comprehension');

        // Wait for loading to complete
        await expect(page.getByText('Loading your next question...')).toBeHidden();
        await expect(page.getByText('Generating your personalized question...')).toBeHidden({timeout: 15000});

        // Require the New Question button to be visible
        await expect(page.getByRole('button', {name: 'New Question'})).toBeVisible();

        // Click the button and check loading
        await page.getByRole('button', {name: 'New Question'}).click();

        // Check for either loading state
        await expect(async () => {
            const hasLoadingText = await page.getByText('Loading your next question...').isVisible();
            const hasGeneratingText = await page.getByText('Generating your personalized question...').isVisible();
            const hasTransitioningState = await page.locator('[data-testid="question-content"]').isVisible() === false;
            expect(hasLoadingText || hasGeneratingText || hasTransitioningState).toBe(true);
        }).toPass({timeout: 5000});
    });

    test('should maintain reading comprehension state on page refresh', async ({page}) => {
        // Navigate to reading comprehension page
        await page.getByRole('link', {name: 'Reading Comprehension'}).click();
        await expect(page).toHaveURL('/reading-comprehension');

        // Wait for loading to complete
        await expect(page.getByText('Loading your next question...')).toBeHidden();
        await expect(page.getByText('Generating your personalized question...')).toBeHidden({timeout: 15000});

        // Refresh the page
        await page.reload();

        // Should still be on reading comprehension page after refresh
        await expect(page).toHaveURL('/reading-comprehension');
        await expect(page.getByText('Learning Preferences')).not.toBeVisible();
        await expect(page.getByText('Performance by Topic')).not.toBeVisible();
    });

    test('should switch between Quiz and Reading Comprehension sections', async ({page}) => {
        // Start on the main quiz page
        await expect(page.getByText('Loading your next question...')).toBeHidden();

        // Navigate to Reading Comprehension
        await page.getByRole('link', {name: 'Reading Comprehension'}).click();
        await expect(page).toHaveURL('/reading-comprehension');

        // Wait for loading to complete
        await expect(page.getByText('Loading your next question...')).toBeHidden();
        await expect(page.getByText('Generating your personalized question...')).toBeHidden({timeout: 15000});

        // Navigate back to Quiz
        await page.getByRole('link', {name: 'Quiz'}).click();
        await expect(page).toHaveURL('/quiz');

        // If redirected to login, perform login again
        if (page.url().includes('/login') || await page.getByLabel('Username').isVisible({timeout: 1000}).catch(() => false)) {
            await expect(page.getByLabel('Username')).toBeVisible({timeout: 5000});
            await page.getByLabel('Username').fill('testuser');
            await page.getByLabel('Password').fill('password');
            await page.locator('form').getByRole('button', {name: 'Sign In'}).click();
            await page.waitForURL('/quiz');
        }

        // Should show loading state as it loads a new question
        await expect(page.getByText('Loading your next question...')).toBeHidden();
        await expect(page.getByText('Generating your personalized question...')).toBeHidden({timeout: 15000});

        // Should be back on quiz page
        await expect(page).toHaveURL('/quiz');
    });

    test('should show appropriate header information on reading comprehension page', async ({page}) => {
        // Navigate to reading comprehension page
        await page.getByRole('link', {name: 'Reading Comprehension'}).click();
        await expect(page).toHaveURL('/reading-comprehension');

        // Wait for all loading states to complete
        await expect(page.getByText('Loading your next question...')).toBeHidden();
        await expect(page.getByText('Generating your personalized question...')).toBeHidden({timeout: 15000});

        // Wait for either a question to appear or an error state
        const questionContent = page.locator('[data-testid="question-content"]');
        const errorText = page.locator('p').filter({hasText: 'No questions available'});
        const tryAgainButton = page.getByRole('button', {name: 'Try Again'});

        // Check if we have a question available (which means the header should be visible)
        if (await questionContent.isVisible()) {
            // Should show language in header
            await expect(page.locator('[data-testid="quiz-title"]').filter({hasText: 'Quiz'})).toBeVisible({timeout: 2000});

            // Should show current level
            await expect(page.locator('[data-testid="quiz-level"]')).toBeVisible();
        } else if (await errorText.isVisible() || await tryAgainButton.isVisible()) {
            // If there's an error or no questions, we can't test the header - skip this part
            console.log('Skipping header test due to error state or no questions available');
        } else {
            // Wait a bit more and try again
            await page.waitForTimeout(2000);

            // Try to get a new question if possible
            const newQuestionButton = page.getByRole('button', {name: 'New Question'});
            if (await newQuestionButton.isVisible()) {
                await newQuestionButton.click();
                await expect(page.getByText('Loading your next question...')).toBeHidden();
                await expect(page.getByText('Generating your personalized question...')).toBeHidden({timeout: 15000});

                // Check again after getting a new question
                if (await questionContent.isVisible()) {
                    await expect(page.locator('[data-testid="quiz-title"]')).toBeVisible({timeout: 2000});
                    await expect(page.locator('[data-testid="quiz-level"]')).toContainText('Level:');
                }
            }
        }
    });

    test('should display user\'s selected answer in reading comprehension feedback', async ({page}) => {
        // Navigate to reading comprehension page
        await page.getByRole('link', {name: 'Reading Comprehension'}).click();
        await expect(page).toHaveURL('/reading-comprehension');

        // Wait for loading to complete
        await expect(page.getByText('Loading your next question...')).toBeHidden();
        await expect(page.getByText('Generating your personalized question...')).toBeHidden({timeout: 15000});

        const questionContent = page.locator('[data-testid="question-content"]');

        // If no question is immediately visible, try getting a new one
        if (!(await questionContent.isVisible())) {
            const newQuestionButton = page.getByRole('button', {name: 'New Question'});
            if (await newQuestionButton.isVisible()) {
                await newQuestionButton.click();
                await expect(page.getByText('Loading your next question...')).toBeHidden();
                await expect(page.getByText('Generating your personalized question...')).toBeHidden({timeout: 15000});
            }
        }

        // Now the question should be visible - this test requires a question to be meaningful
        await expect(questionContent).toBeVisible({timeout: 2000});

        const questionText = (await questionContent.textContent())?.trim() || '';
        const radioOptions = page.locator('input[type="radio"]');
        const optionCount = await radioOptions.count();
        const submitButton = page.getByRole('button', {name: 'Submit'});

        expect(optionCount).toBeGreaterThan(0); // Ensure we have options to test with

        // Get all option labels
        const optionLabels: string[] = [];
        for (let i = 0; i < optionCount; i++) {
            const label = await page.locator('label').nth(i).textContent();
            if (label) optionLabels.push(label.trim());
        }

        // Select the first option
        let selectedOption = optionLabels[0]; // Default to first option
        let selectedIndex = 0;

        // Select the option
        await radioOptions.nth(selectedIndex).click();
        await expect(submitButton).toBeEnabled({timeout: 5000});
        await submitButton.click();

        // Wait for feedback to appear
        const questionCard = page.locator('[data-testid="question-card"]');
        await expect(questionCard.locator('.mantine-Alert-title')).toBeVisible({timeout: 5000});

        // Check what kind of feedback we got
        const feedbackText = await questionCard.locator('.mantine-Alert-title').textContent();
        const isCorrect = feedbackText?.includes('Correct');
        const isIncorrect = feedbackText?.includes('Incorrect');

        expect(isCorrect || isIncorrect).toBe(true); // Should show either correct or incorrect

        // The key test: verify that feedback is displayed properly
        // Look for visual indicators that feedback is working:

        // 1. Should show the Next Question button after submission
        await expect(page.getByRole('button', {name: 'Next Question'})).toBeVisible({timeout: 5000});

        // 2. The user's selected answer should be visible somewhere in the feedback
        // The answer should appear either in "Your Answer" section or in feedback content
        const hasUserAnswer = await page.getByText('Your Answer').isVisible();
        const hasAnswerInFeedback = await page.getByText(selectedOption, {exact: true}).isVisible();

        expect(hasUserAnswer || hasAnswerInFeedback).toBe(true);

        // 3. Should show explanation button or feedback content
        // Use a more specific selector to avoid strict mode violations
        const hasExplanationButton = await page.getByRole('button', {name: /explanation/i}).isVisible();
        const hasFeedbackContent = await questionCard.locator('.mantine-Alert-title').isVisible();

        expect(hasExplanationButton || hasFeedbackContent).toBe(true);

        // 4. Verify the feedback shows appropriate styling (correct vs incorrect)
        if (isCorrect) {
            // For correct answers, should have green styling somewhere
            const hasGreenStyling = await page.locator('[class*="green"], [style*="green"], .mantine-Text-root[color="green"]').count() > 0;
            expect(hasGreenStyling).toBe(true);
        } else if (isIncorrect) {
            // For incorrect answers, should have red styling somewhere
            const hasRedStyling = await page.locator('[class*="red"], [style*="red"], .mantine-Text-root[color="red"]').count() > 0;
            expect(hasRedStyling).toBe(true);
        }
    });
});
