# iOS App Build Report

## Implemented Features

An iOS application was built using Swift and SwiftUI, mirroring the functionality of the mobile web UI. The app includes the following features:

*   **Authentication:** Users can sign up, log in, and log out. Authentication tokens are securely stored in the Keychain.
*   **Core UI:** A tab-based interface provides navigation to the main sections of the app: Home (Quiz), Learn, Vocabulary, Phrasebook, and Profile (Settings).
*   **Daily Quiz:** Users can take quizzes, answer questions, and receive feedback.
*   **Learning Modules:**
    *   **Stories:** Users can browse a list of stories and read their content.
    *   **Vocabulary:** Users can view their saved vocabulary snippets.
    *   **Phrasebook:** Users can browse a phrasebook with categories and phrases.
    *   **Translation Practice:** A dedicated quiz type for translation practice.
    *   **Verb Conjugation:** A dedicated quiz type for verb conjugation practice.
*   **Settings:** Users can view and update their profile information (username, email, language, and level).

## Testing

The application was tested by writing unit tests for the view models. The `APIService` was mocked to allow for isolated testing of the view models. The following view models were tested:

*   `AuthenticationViewModel`
*   `QuizViewModel`
*   `StoryViewModel`
*   `VocabularyViewModel`
*   `PhrasebookViewModel`
*   `SettingsViewModel`

## Challenges

The main challenge was the rapid implementation of a relatively large number of features. The use of a clear and structured plan, along with a consistent MVVM architecture, was crucial for managing the complexity. The `replace` tool's limitations with single-character replacements required some workarounds. A significant amount of time was spent on creating the models and API service functions, as well as the corresponding view models and views. The testing phase was also time-consuming, but essential for ensuring the quality of the application.
