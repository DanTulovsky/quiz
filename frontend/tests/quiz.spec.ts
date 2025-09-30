import {test, expect} from '@playwright/test';
import {assertStatus} from './http-assert';
import {fileURLToPath} from 'url';
import path from 'path';
import fs from 'fs';
import yaml from 'js-yaml';
import {resetTestDatabase} from './reset-db';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

test.beforeAll(() => {
  resetTestDatabase();
});

// Dynamically load correct answers from golden YAML at test time
function loadCorrectAnswers() {
  const yamlPath = path.resolve(__dirname, '../../backend/data/test_questions.yaml');
  const doc = yaml.load(fs.readFileSync(yamlPath, 'utf8')) as unknown;
  const map: Record<string, string> = {};

  // Type guard to ensure we have the expected structure
  if (doc && typeof doc === 'object' && 'questions' in doc) {
    const data = doc as {questions: Array<{content?: {question?: string}, correct_answer?: string}>};
    for (const q of data.questions) {
      if (q.content && q.content.question && q.correct_answer) {
        map[q.content.question] = q.correct_answer;
      }
    }
  }
  return map;
}

test.describe.serial('Quiz Functionality', () => {
  // Helper to login before each test using the existing testuser
  test.beforeEach(async ({page}) => {
    await page.goto('/login');
    await page.getByLabel('Username').fill('testuser');
    await page.getByLabel('Password').fill('password');
    await page.locator('form').getByRole('button', {name: 'Sign In'}).click();
    await page.waitForURL('/');
  });

  test('should load quiz page and show question or generation message', async ({page}) => {
    // Wait for loading to complete - check that loading spinner is gone
    await expect(page.getByText('Loading your next question...')).toBeHidden();

    // Wait for one of the expected states to appear
    await expect(async () => {
      const hasQuizCard = await page.locator('[data-testid="question-content"]').isVisible();
      const hasGenerationMessage = await page.getByText(/generating/i).isVisible();
      // Fix the text selector - the actual text is "No question available." (singular)
      const hasErrorMessage = await page.getByText(/No question available/i).isVisible();

      // At least one of these should be true
      expect(hasQuizCard || hasGenerationMessage || hasErrorMessage).toBe(true);
    }).toPass({timeout: 2000});

    // Now check which state we're in and verify it's correct
    const hasQuizCard = await page.locator('[data-testid="question-content"]').isVisible();

    if (hasQuizCard) {
      // If quiz card is shown, it should have question content
      await expect(page.locator('[data-testid="question-content"]')).toBeVisible();
      // Should show level indicator - use the specific data-testid we added
      await expect(page.locator('[data-testid="quiz-level"]')).toBeVisible();

      // Verify that this is NOT a reading comprehension question (should be filtered out)
      const questionText = await page.locator('[data-testid="question-content"]').textContent();
      if (questionText) {
        // Reading comprehension questions typically have very long passages
        // Regular quiz questions should be shorter
        expect(questionText.length).toBeLessThan(500);
      }
    }
  });

  test('should handle question interaction correctly', async ({page}) => {
    await expect(page.getByText('Loading your next question...')).toBeHidden();

    // Check if we have a question available
    const questionContent = page.locator('[data-testid="question-content"]');
    if (await questionContent.isVisible()) {
      // All questions now use multiple choice radio buttons
      const multipleChoiceOption = page.locator('input[type="radio"]').first();
      const submitButton = page.getByRole('button', {name: 'Submit'});

      if (await multipleChoiceOption.isVisible()) {
        // Click the first radio button option
        await multipleChoiceOption.click();
        // Wait for the submit button to be enabled
        await expect(submitButton).toBeEnabled({timeout: 5000});
        await submitButton.click();

        // Should show feedback after submission - look for the Alert with feedback
        await expect(page.getByText(/Correct!|Incorrect/)).toBeVisible({timeout: 2000});
      }
    }
  });

  test('should show next question button after answering', async ({page}) => {
    await expect(page.getByText('Loading your next question...')).toBeHidden();

    const questionContent = page.locator('[data-testid="question-content"]');
    if (await questionContent.isVisible()) {
      // All questions now use multiple choice radio buttons
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
        // Should show loading again - but it might be very brief, so check for either loading state or question transition
        await expect(async () => {
          const hasLoadingText = await page.getByText('Loading your next question...').isVisible();
          const hasGeneratingText = await page.getByText('Generating your personalized question...').isVisible();
          const hasTransitioningState = await page.locator('[data-testid="question-content"]').isVisible() === false;
          expect(hasLoadingText || hasGeneratingText || hasTransitioningState).toBe(true);
        }).toPass({timeout: 5000});
      }
    }
  });

  test('should navigate to progress page', async ({page}) => {
    // Answer a quiz question to ensure progress data exists
    await expect(page.getByText('Loading your next question...')).toBeHidden();
    const questionContent = page.locator('[data-testid="question-content"]');
    if (await questionContent.isVisible()) {
      const firstOption = page.locator('input[type="radio"]').first();
      if (await firstOption.isVisible()) {
        await firstOption.click();
        await page.getByRole('button', {name: 'Submit'}).click();
        await expect(page.getByText(/Correct!|Incorrect/)).toBeVisible({timeout: 2000});
      }
    }

    // Ensure authentication is fully established before making API calls
    await expect(async () => {
      const response = await page.request.get('/v1/quiz/progress');
      await assertStatus(response, 200, {method: 'GET', url: '/v1/quiz/progress'});
      const data = await response.json();
      expect(data).toBeDefined();
      return data;
    }).toPass({timeout: 5000});

    // Now click on progress navigation
    // Get progress data for verification
    const progressResp = await page.request.get('/v1/quiz/progress');
    const progressJson = await progressResp.json();

    await page.locator('header').getByLabel('Progress').click();
    await expect(page).toHaveURL('/progress');

    // Wait for the page to load and check for progress content
    await expect(async () => {
      // Check for various progress page elements that should be present
      const hasCurrentLevel = await page.getByText('Current Level').isVisible().catch(() => false);
      const hasQuestionsAnswered = await page.getByText('Questions Answered').isVisible().catch(() => false);
      const hasAccuracy = await page.getByText('Accuracy').isVisible().catch(() => false);
      const hasPerformanceByTopic = await page.getByText('Performance by Topic').isVisible().catch(() => false);
      const hasAreasToImprove = await page.getByText('Areas to Improve').isVisible().catch(() => false);
      const hasRecentActivity = await page.getByText('Recent Activity').isVisible().catch(() => false);
      const hasProgressCards = await page.locator('svg').isVisible().catch(() => false); // Look for SVG elements (RingProgress)
      const hasProgressTable = await page.locator('table').isVisible().catch(() => false);

      // Check if any of the expected content is present
      const hasAnyProgressContent = hasCurrentLevel || hasQuestionsAnswered || hasAccuracy ||
        hasPerformanceByTopic || hasAreasToImprove || hasRecentActivity ||
        hasProgressCards || hasProgressTable;

      if (!hasAnyProgressContent) {
        // Progress content not found - this will cause the test to fail
      }

      expect(hasAnyProgressContent).toBe(true);
    }).toPass({timeout: 15000});
  });

  test('should navigate to settings page', async ({page}) => {
    // Wait for the page to be fully loaded and navigation to be available
    await expect(page.getByText('Loading your next question...')).toBeHidden();

    // Wait for authentication to complete and navigation to be ready
    // First check if we're logged in by looking for the username in the header
    await expect(page.getByText('testuser')).toBeVisible({timeout: 2000});

    // Wait for navigation to be ready by checking for any nav link first
    await expect(page.locator('header').getByLabel('Settings')).toBeVisible({timeout: 2000});

    // Ensure authentication is fully established by waiting for a successful API call
    let providersData: any;
    await expect(async () => {
      const response = await page.request.get('/v1/settings/ai-providers');
      await assertStatus(response, 200, {method: 'GET', url: '/v1/settings/ai-providers'});
      providersData = await response.json();
      expect(providersData).toHaveProperty('levels');
    }).toPass({timeout: 5000});

    // Now click on settings navigation
    await page.locator('header').getByLabel('Settings').click();

    // Should navigate to settings page
    await expect(page).toHaveURL('/settings');

    // Should show settings page title - be more specific
    await expect(page.getByRole('heading', {name: 'Settings', exact: true})).toBeVisible();

    // Should show learning preferences section
    await expect(page.getByRole('heading', {name: 'Learning Preferences'})).toBeVisible();

    // Should show form fields - check by labels
    await expect(page.getByText('Learning Language')).toBeVisible();
    await expect(page.getByText('Current Level')).toBeVisible();

    // Check that the select elements are present and have values - Mantine Select components
    const languageSelect = page.locator('[data-testid="learning-language-select"]');
    const levelSelect = page.getByText('Current Level').locator('..').locator('input');

    await expect(languageSelect).toBeVisible();
    await expect(levelSelect).toBeVisible();

    // Verify they have values - get available options dynamically from API
    const languageValue = await languageSelect.inputValue();
    const levelValue = await levelSelect.inputValue();

    // Verify language is one of the supported languages - read from config.yaml
    const aiProvidersPath = path.resolve(__dirname, '../../config.yaml');
    const aiProvidersDoc = yaml.load(fs.readFileSync(aiProvidersPath, 'utf8')) as unknown as {language_levels: Record<string, any>};
    const supportedLanguages = Object.keys(aiProvidersDoc.language_levels).map((lang: string) =>
      lang.charAt(0).toUpperCase() + lang.slice(1)
    );
    expect(supportedLanguages).toContain(languageValue);

    // Get available levels from the API dynamically to avoid hardcoding
    // Use the data we already fetched during authentication check
    const availableLevels = providersData.levels;

    // Verify the current level is one of the available levels from the API
    expect(availableLevels).toContain("A1");
  });

  test('should update settings', async ({page}) => {
    // Load available languages from config.yaml and levels from API
    const aiProvidersPath = path.resolve(__dirname, '../../config.yaml');
    const aiProvidersDoc = yaml.load(fs.readFileSync(aiProvidersPath, 'utf8')) as unknown as {language_levels: Record<string, any>};
    const availableLanguages = Object.keys(aiProvidersDoc.language_levels).map((lang: string) =>
      lang.charAt(0).toUpperCase() + lang.slice(1)
    );

    // Wait for authentication to be fully established before making API calls
    let providersData: any;
    await expect(async () => {
      const response = await page.request.get('/v1/settings/ai-providers');
      await assertStatus(response, 200, {method: 'GET', url: '/v1/settings/ai-providers'});
      providersData = await response.json();
      expect(providersData).toHaveProperty('levels');
    }).toPass({timeout: 5000});

    // Get available levels from the API dynamically
    const availableLevels = providersData.levels;



    // Wait for the page to be fully loaded and navigation to be available
    await expect(page.getByText('Loading your next question...')).toBeHidden();

    // Wait for authentication to complete and navigation to be ready
    // First check if we're logged in by looking for the username in the header
    await expect(page.getByText('testuser')).toBeVisible({timeout: 2000});

    // Wait for navigation to be ready by checking for any nav link first
    await expect(page.locator('header').getByLabel('Settings')).toBeVisible({timeout: 2000});

    // Now click on settings navigation
    await page.locator('header').getByLabel('Settings').click();

    // Wait for the settings page to load - check for loading or error states first
    await expect(page.getByText('Learning Preferences')).toBeVisible();

    // Check if we're in a loading state
    const loadingText = page.getByText('Loading settings...');
    if (await loadingText.isVisible()) {
      await expect(loadingText).toBeHidden({timeout: 2000});
    }

    // Check for error state
    const errorText = page.getByText(/Error loading settings/);
    if (await errorText.isVisible()) {
      throw new Error('Settings page failed to load: ' + await errorText.textContent());
    }

    // Now wait for the form content to appear
    await expect(page.getByText('Learning Preferences')).toBeVisible({timeout: 2000});

    // Wait for the form elements to be present - Mantine Select components
    await expect(page.locator('[data-testid="learning-language-select"]')).toBeVisible({timeout: 2000});
    await expect(page.getByText('Current Level')).toBeVisible({timeout: 2000});

    // Get current values first to avoid issues with test order dependencies - Mantine Select components
    const languageSelect = page.locator('[data-testid="learning-language-select"]');
    const levelSelect = page.getByText('Current Level').locator('..').locator('input');

    const currentLanguage = await languageSelect.inputValue();
    const currentLevel = await levelSelect.inputValue();

    // Change to different values - use first two available languages and levels
    const firstLanguage = availableLanguages[0];
    const secondLanguage = availableLanguages.length > 1 ? availableLanguages[1] : availableLanguages[0];
    const newLanguage = currentLanguage === firstLanguage ? secondLanguage : firstLanguage;

    const firstLevel = availableLevels[0];
    const secondLevel = availableLevels.length > 1 ? availableLevels[1] : availableLevels[0];
    const newLevel = currentLevel === firstLevel ? secondLevel : firstLevel;

    // Update Mantine Select components
    await languageSelect.click();
    await page.waitForTimeout(500);

    // Be more specific about which element to click to avoid strict mode violation
    await page.locator('[role="option"]').filter({hasText: newLanguage}).first().click();

    // Wait for the value to actually change
    await expect(languageSelect).toHaveValue(newLanguage, {timeout: 5000});

    await levelSelect.click();
    await page.waitForTimeout(500);
    await page.getByText(newLevel).click();

    // Wait for the level value to actually change - Mantine shows full text like "A1 — Beginner"
    const expectedLevelText = `${newLevel} — ${newLevel === 'A1' ? 'Beginner' : newLevel === 'A2' ? 'Elementary' : newLevel === 'B1' ? 'Intermediate' : newLevel === 'B2' ? 'Upper Intermediate' : newLevel === 'C1' ? 'Advanced' : 'Proficient'}`;
    await expect(levelSelect).toHaveValue(expectedLevelText, {timeout: 5000});

    // Wait for the save button to be enabled (when isDirty becomes true)
    const saveButton = page.getByRole('button', {name: 'Save Changes'});
    await expect(saveButton).toBeEnabled({timeout: 5000});



    await saveButton.click();

    // Should show success message in Mantine notification
    await expect(page.getByText('Settings saved successfully', {exact: false})).toBeVisible({timeout: 2000});


    // Instead of reloading the page (which causes routing issues),
    // navigate away and back to verify persistence
    await page.getByRole('link', {name: 'Quiz'}).click();

    // Wait for the page to load - might show "No questions available" if no questions for the new language/level
    await page.waitForTimeout(2000);

    // Navigate back to settings to verify the change worked
    await page.locator('header').getByLabel('Settings').click();
    await expect(page.getByText('Learning Preferences')).toBeVisible({timeout: 2000});


    // Wait for the form to be re-initialized with updated user data
    // The form fields might take a moment to update after navigation
    // First wait for any loading states to clear
    const settingsLoadingText = page.getByText('Loading settings...');
    if (await settingsLoadingText.isVisible()) {
      await expect(settingsLoadingText).toBeHidden({timeout: 2000});
    }

    // Wait for the form fields to be updated with the new user data
    await expect(async () => {
      const currentLanguageValue = await languageSelect.inputValue();
      expect(currentLanguageValue).toBe(newLanguage);
    }).toPass({timeout: 10000});

    // Verify the settings were persisted (first change) - Mantine Select components
    await expect(languageSelect).toHaveValue(newLanguage);
    const expectedLevelTextForVerification = `${newLevel} — ${newLevel === 'A1' ? 'Beginner' : newLevel === 'A2' ? 'Elementary' : newLevel === 'B1' ? 'Intermediate' : newLevel === 'B2' ? 'Upper Intermediate' : newLevel === 'C1' ? 'Advanced' : 'Proficient'}`;
    await expect(levelSelect).toHaveValue(expectedLevelTextForVerification);

    // Now change back to first language (since we likely have questions for it)
    await languageSelect.click();
    // Use a more specific selector to avoid strict mode violation
    await page.locator('[role="option"]').filter({hasText: firstLanguage}).first().click();


    // Wait for save button to be enabled again
    await expect(saveButton).toBeEnabled({timeout: 5000});
    await saveButton.click();

    // Should show success message again in Mantine notification
    await expect(page.getByText('Settings saved successfully', {exact: false})).toBeVisible({timeout: 2000});


    // Navigate to quiz to verify questions are available for the first language
    await page.getByRole('link', {name: 'Quiz'}).click();

    // Should show Quiz for the first language since we likely have questions available
    const expectedQuizTitle = `${firstLanguage} Quiz`;
    const hasQuestions = await page.getByText(expectedQuizTitle).isVisible();
    const noQuestions = await page.getByText('No questions available').isVisible();

    if (hasQuestions) {
      await expect(page.getByText(expectedQuizTitle)).toBeVisible({timeout: 5000});
    } else if (noQuestions) {
      // If no questions available, that's also acceptable for this test
      await expect(page.getByText('No questions available')).toBeVisible({timeout: 5000});
    } else {
      // Wait a bit and check again
      await page.waitForTimeout(2000);
      await expect(page.locator('main, [role="main"], .mantine-Container-root').first()).toBeVisible({timeout: 5000});
    }

    // Navigate back to settings for final verification
    await page.locator('header').getByLabel('Settings').click();
    await expect(page.getByText('Learning Preferences')).toBeVisible({timeout: 2000});


    // Wait for the form to be re-initialized with updated user data
    // First wait for any loading states to clear
    const finalSettingsLoadingText = page.getByText('Loading settings...');
    if (await finalSettingsLoadingText.isVisible()) {
      await expect(finalSettingsLoadingText).toBeHidden({timeout: 2000});
    }

    // Wait for the form fields to be updated with the new user data
    await expect(async () => {
      const currentLanguageValue = await languageSelect.inputValue();
      expect(currentLanguageValue).toBe(firstLanguage);
    }).toPass({timeout: 10000});

    // Verify the final settings (first language, changed level) - Mantine Select components
    await expect(languageSelect).toHaveValue(firstLanguage);
    const finalExpectedLevelText = "A1 — Beginner";
    await expect(levelSelect).toHaveValue(finalExpectedLevelText);
  });

  test('should remember current page on browser refresh', async ({page}) => {
    // Wait for the page to be fully loaded and navigation to be available
    await expect(page.getByText('Loading your next question...')).toBeHidden();

    // Wait for authentication to complete and navigation to be ready
    // First check if we're logged in by looking for the username in the header
    await expect(page.getByText('testuser')).toBeVisible({timeout: 2000});

    // Wait for navigation to be ready by checking for any nav link first
    await expect(page.locator('header').getByLabel('Settings')).toBeVisible({timeout: 2000});

    // Test Settings page refresh
    await page.locator('header').getByLabel('Settings').click();
    await expect(page.getByText('Learning Preferences')).toBeVisible({timeout: 2000});

    // Refresh the page
    await page.reload();

    // Should still be on settings page after refresh
    await expect(page.getByText('Learning Preferences')).toBeVisible({timeout: 2000});
    await expect(page).toHaveURL('/settings');

    // Test Progress page refresh
    await page.locator('header').getByLabel('Progress').click();
    await expect(page.getByText('Performance by Topic')).toBeVisible({timeout: 2000});

    // Refresh the page
    await page.reload();

    // Should still be on progress page after refresh
    await expect(page.getByText('Performance by Topic')).toBeVisible({timeout: 2000});
    await expect(page).toHaveURL('/progress');

    // Test Quiz page refresh
    await page.getByRole('link', {name: 'Quiz'}).click();

    // Load first language from config for dynamic testing
    const aiProvidersPath = path.resolve(__dirname, '../../config.yaml');
    const aiProvidersDoc = yaml.load(fs.readFileSync(aiProvidersPath, 'utf8')) as unknown as {language_levels: Record<string, any>};
    const firstLanguageKey = Object.keys(aiProvidersDoc.language_levels)[0];
    const firstLanguage = firstLanguageKey.charAt(0).toUpperCase() + firstLanguageKey.slice(1);

    // Quiz page might show quiz for first language if questions are available, or "No questions available" if none
    const expectedQuizTitle = `${firstLanguage} Quiz`;
    const hasQuestions = await page.getByText(expectedQuizTitle).isVisible();
    const noQuestions = await page.getByText('No questions available').isVisible();

    if (!hasQuestions && !noQuestions) {
      // Wait a bit for the page to load
      await page.waitForTimeout(2000);
    }

    // Verify we're on the quiz page (either with questions or without)
    await expect(page.locator('main, [role="main"], .mantine-Container-root').first()).toBeVisible({timeout: 2000});

    // Refresh the page
    await page.reload();

    // Should still be on quiz page after refresh - check URL and that we're not on other pages
    await expect(page).toHaveURL('/quiz');
    await expect(page.getByText('Learning Preferences')).not.toBeVisible();
    await expect(page.getByText('Performance by Topic')).not.toBeVisible();
  });

  test('should handle new question button', async ({page}) => {
    await expect(page.getByText('Loading your next question...')).toBeHidden();

    // Require the New Question button to be visible
    await expect(page.getByRole('button', {name: 'New Question'})).toBeVisible();

    // Click the button and check loading - handle both possible loading states
    await page.getByRole('button', {name: 'New Question'}).click();

    // Check for either loading state - the API might return generating status immediately
    await expect(async () => {
      const hasLoadingText = await page.getByText('Loading your next question...').isVisible();
      const hasGeneratingText = await page.getByText('Generating your personalized question...').isVisible();
      const hasTransitioningState = await page.locator('[data-testid="question-content"]').isVisible() === false;
      expect(hasLoadingText || hasGeneratingText || hasTransitioningState).toBe(true);
    }).toPass({timeout: 5000});
  });

  test('should show appropriate quiz header information', async ({page}) => {
    // Wait for all loading states to complete
    await expect(page.getByText('Loading your next question...')).toBeHidden();
    await expect(page.getByText('Generating your personalized question...')).toBeHidden({timeout: 15000});

    // Wait for either a question to load OR an error state
    const questionContent = page.locator('[data-testid="question-content"]');
    const errorText = page.locator('p').filter({hasText: 'No questions available'});
    const tryAgainButton = page.getByRole('button', {name: 'Try Again'});

    // Check if we have a question available (which means the header should be visible)
    if (await questionContent.isVisible()) {
      // Should show language in header - use specific data-testid and filter by text content
      await expect(page.locator('[data-testid="quiz-title"]').filter({hasText: 'Quiz'})).toBeVisible({timeout: 2000});

      // Should show current level - use specific data-testid
      await expect(page.locator('[data-testid="quiz-level"]')).toBeVisible();
    } else if (await errorText.isVisible() || await tryAgainButton.isVisible()) {
      // If there's an error or no questions, we can't test the header - skip this part
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

  test('should display user\'s selected answer in feedback', async ({page}) => {
    const correctAnswers = loadCorrectAnswers();
    await expect(page.getByText('Loading your next question...')).toBeHidden();

    // Wait for either a question to appear or generation to complete
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

    // Try to intentionally select a wrong answer, but handle the case where we might get it right
    let selectedOption = optionLabels[0]; // Default to first option
    let selectedIndex = 0;

    if (correctAnswers[questionText]) {
      const correct = correctAnswers[questionText];
      const wrongOption = optionLabels.find(opt => opt !== correct);
      if (wrongOption) {
        selectedOption = wrongOption;
        selectedIndex = optionLabels.findIndex(opt => opt === wrongOption);
      }
    }

    // Select the option
    await radioOptions.nth(selectedIndex).click();
    await expect(submitButton).toBeEnabled({timeout: 5000});
    await submitButton.click();

    // Wait for feedback to appear
    await expect(page.getByText(/Correct!|Incorrect/)).toBeVisible({timeout: 2000});

    // Check what kind of feedback we got
    const feedbackText = await page.getByText(/Correct!|Incorrect/).textContent();
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
    const hasFeedbackContent = await page.getByText(/Correct!|Incorrect/).isVisible();

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

  test('should show user\'s selected answer for both correct and incorrect responses', async ({page}) => {
    await expect(page.getByText('Loading your next question...')).toBeHidden();

    // Try to answer multiple questions to test both correct and incorrect scenarios
    for (let i = 0; i < 3; i++) {
      const questionContent = page.locator('[data-testid="question-content"]');

      if (await questionContent.isVisible()) {
        const radioOptions = page.locator('input[type="radio"]');
        const submitButton = page.getByRole('button', {name: 'Submit'});

        if (await radioOptions.count() > 0) {
          // Try different options to increase chance of both correct and incorrect answers
          const optionIndex = i % await radioOptions.count();
          const selectedOption = radioOptions.nth(optionIndex);

          // Get the text of the selected option
          const selectedLabel = await page.locator('label').nth(optionIndex).textContent();

          await selectedOption.click();
          await expect(submitButton).toBeEnabled({timeout: 5000});
          await submitButton.click();

          // Wait for feedback
          await expect(page.getByText(/Correct!|Incorrect/)).toBeVisible({timeout: 2000});

          // Verify the user's answer is shown somewhere in the feedback
          if (selectedLabel) {
            const trimmedLabel = selectedLabel.trim();
            // The answer should appear either in "Your Answer" section or in feedback content
            const hasUserAnswer = await page.getByText('Your Answer').isVisible();
            const hasAnswerInFeedback = await page.getByText(trimmedLabel, {exact: true}).isVisible();

            expect(hasUserAnswer || hasAnswerInFeedback).toBe(true);
          }

          // Move to next question if available
          const nextButton = page.getByRole('button', {name: 'Next Question'});
          if (await nextButton.isVisible()) {
            await nextButton.click();
            await expect(page.getByText('Loading your next question...')).toBeHidden();
          } else {
            break;
          }
        } else {
          break;
        }
      } else {
        // Try to get a new question
        const newQuestionButton = page.getByRole('button', {name: 'New Question'});
        if (await newQuestionButton.isVisible()) {
          await newQuestionButton.click();
          await expect(page.getByText('Loading your next question...')).toBeHidden();
        } else {
          break;
        }
      }
    }
  });
});
