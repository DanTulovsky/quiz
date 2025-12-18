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
}
