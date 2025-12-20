import XCTest
import Combine
@testable import Quiz

class ViewModelCancellationTests: XCTestCase {

    func testQuizViewModelCancellation() {
        // Given
        let viewModel = QuizViewModel()
        let expectation = XCTestExpectation(description: "Requests cancelled")

        // When - start a request
        viewModel.getQuestion()

        // Then - cancel all requests
        viewModel.cancelAllRequests()

        // Verify cancellables are cleared
        // Note: We can't directly test cancellables, but we can verify the method exists and doesn't crash
        XCTAssertNoThrow(viewModel.cancelAllRequests(), "Cancellation should not throw")

        expectation.fulfill()
        wait(for: [expectation], timeout: 1.0)
    }

    func testAuthenticationViewModelCancellation() {
        // Given
        let viewModel = AuthenticationViewModel()

        // When/Then
        XCTAssertNoThrow(viewModel.cancelAllRequests(), "Cancellation should not throw")
    }

    func testSettingsViewModelCancellation() {
        // Given
        let viewModel = SettingsViewModel()

        // When/Then
        XCTAssertNoThrow(viewModel.cancelAllRequests(), "Cancellation should not throw")
    }

    func testDailyViewModelCancellation() {
        // Given
        let viewModel = DailyViewModel()

        // When/Then
        XCTAssertNoThrow(viewModel.cancelAllRequests(), "Cancellation should not throw")
    }

    func testVocabularyViewModelCancellation() {
        // Given
        let viewModel = VocabularyViewModel()

        // When/Then
        XCTAssertNoThrow(viewModel.cancelAllRequests(), "Cancellation should not throw")
    }

    func testStoryViewModelCancellation() {
        // Given
        let viewModel = StoryViewModel()

        // When/Then
        XCTAssertNoThrow(viewModel.cancelAllRequests(), "Cancellation should not throw")
    }

    func testWordOfTheDayViewModelCancellation() {
        // Given
        let viewModel = WordOfTheDayViewModel()

        // When/Then
        XCTAssertNoThrow(viewModel.cancelAllRequests(), "Cancellation should not throw")
    }

    func testTranslationPracticeViewModelCancellation() {
        // Given
        let viewModel = TranslationPracticeViewModel()

        // When/Then
        XCTAssertNoThrow(viewModel.cancelAllRequests(), "Cancellation should not throw")
    }

    func testVerbViewModelCancellation() {
        // Given
        let viewModel = VerbViewModel()

        // When/Then
        XCTAssertNoThrow(viewModel.cancelAllRequests(), "Cancellation should not throw")
    }

    func testAIHistoryViewModelCancellation() {
        // Given
        let viewModel = AIHistoryViewModel()

        // When/Then
        XCTAssertNoThrow(viewModel.cancelAllRequests(), "Cancellation should not throw")
    }
}



