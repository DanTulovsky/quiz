import Combine
import XCTest

@testable import Quiz

class DailyViewModelTests: XCTestCase {
    var viewModel: DailyViewModel!
    var mockAPIService: MockAPIService!

    override func setUp() {
        super.setUp()
        mockAPIService = MockAPIService()
        viewModel = DailyViewModel(apiService: mockAPIService)
    }

    override func tearDown() {
        viewModel = nil
        mockAPIService = nil
        super.tearDown()
    }

    func testFetchDailySuccess() {
        // Given
        let question = Question(
            id: 1, type: "vocabulary", language: "it", level: "A1", content: [:],
            correctAnswerIndex: 1)
        let daily = DailyQuestionWithDetails(
            id: 100, questionId: 1, question: question, isCompleted: false, userAnswerIndex: nil)
        let response = DailyQuestionsResponse(date: "2025-12-17", questions: [daily])
        mockAPIService.getDailyQuestionsResult = .success(response)
        // Set snippets result to avoid error from loadSnippets call
        mockAPIService.getSnippetsResult = .success(SnippetList(limit: 0, offset: 0, query: nil, snippets: []))
        let expectation = XCTestExpectation(description: "Daily questions fetched")

        // When
        viewModel.fetchDaily()

        // Then - wait for async operation
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertEqual(self.viewModel.dailyQuestions.count, 1)
            XCTAssertEqual(self.viewModel.dailyQuestions.first?.id, 100)
            XCTAssertNil(self.viewModel.error)
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }

    func testSubmitAnswerWithBoundsCheck() {
        // Given
        let question = Question(
            id: 1, type: "vocabulary", language: "it", level: "A1", content: [:],
            correctAnswerIndex: 1)
        let daily = DailyQuestionWithDetails(
            id: 100, questionId: 1, question: question, isCompleted: false, userAnswerIndex: nil)
        let response = DailyQuestionsResponse(date: "2025-12-17", questions: [daily])
        mockAPIService.getDailyQuestionsResult = .success(response)
        // Set snippets result to avoid error from loadSnippets call
        mockAPIService.getSnippetsResult = .success(SnippetList(limit: 0, offset: 0, query: nil, snippets: []))

        let answerResponse = DailyAnswerResponse(
            isCorrect: true,
            explanation: "Correct!",
            isCompleted: true,
            correctAnswerIndex: 1,
            userAnswer: "Answer",
            userAnswerIndex: 1
        )
        mockAPIService.postDailyAnswerResult = .success(answerResponse)

        let fetchExpectation = XCTestExpectation(description: "Daily questions fetched")
        let submitExpectation = XCTestExpectation(description: "Answer submitted")

        // When - fetch daily questions first
        viewModel.fetchDaily()
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.2) {
            fetchExpectation.fulfill()
        }
        wait(for: [fetchExpectation], timeout: 1.0)

        // Simulate index becoming invalid (e.g., array cleared or modified)
        viewModel.currentQuestionIndex = 999 // Invalid index

        // When - submit answer with invalid index
        // Since currentQuestion will be nil with invalid index, submitAnswer will return early
        // and not call the API, so the array should remain unchanged
        viewModel.submitAnswer(index: 1)

        // Then - should not crash, should handle gracefully
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.2) {
            // Should not have updated the array at invalid index
            XCTAssertEqual(self.viewModel.dailyQuestions.count, 1, "Array should still have one question")
            // Question should still be uncompleted since submitAnswer returned early
            XCTAssertFalse(self.viewModel.dailyQuestions[0].isCompleted, "Question should not be marked completed with invalid index")
            submitExpectation.fulfill()
        }
        wait(for: [submitExpectation], timeout: 1.0)
    }

    func testSubmitAnswerWithValidIndex() {
        // Given
        let question = Question(
            id: 1, type: "vocabulary", language: "it", level: "A1", content: [:],
            correctAnswerIndex: 1)
        let daily = DailyQuestionWithDetails(
            id: 100, questionId: 1, question: question, isCompleted: false, userAnswerIndex: nil)
        let response = DailyQuestionsResponse(date: "2025-12-17", questions: [daily])
        mockAPIService.getDailyQuestionsResult = .success(response)

        let answerResponse = DailyAnswerResponse(
            isCorrect: true,
            explanation: "Correct!",
            isCompleted: true,
            correctAnswerIndex: 1,
            userAnswer: "Answer",
            userAnswerIndex: 1
        )
        mockAPIService.postDailyAnswerResult = .success(answerResponse)

        let fetchExpectation = XCTestExpectation(description: "Daily questions fetched")
        let submitExpectation = XCTestExpectation(description: "Answer submitted")

        // When - fetch daily questions first
        viewModel.fetchDaily()
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            fetchExpectation.fulfill()
        }
        wait(for: [fetchExpectation], timeout: 1.0)

        // Ensure valid index
        viewModel.currentQuestionIndex = 0

        // When - submit answer with valid index
        viewModel.submitAnswer(index: 1)

        // Then - should update the question
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertTrue(self.viewModel.dailyQuestions[0].isCompleted, "Question should be marked as completed")
            XCTAssertEqual(self.viewModel.dailyQuestions[0].userAnswerIndex, 1, "Should have correct answer index")
            submitExpectation.fulfill()
        }
        wait(for: [submitExpectation], timeout: 1.0)
    }
}
