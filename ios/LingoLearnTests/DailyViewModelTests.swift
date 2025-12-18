import XCTest
import Combine
@testable import LingoLearn

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
        let question = Question(id: 1, type: "vocabulary", language: "it", level: "A1", content: [:], correctAnswerIndex: 1)
        let daily = DailyQuestionWithDetails(id: 100, questionId: 1, question: question, isCompleted: false)
        let response = DailyQuestionsResponse(date: "2025-12-17", questions: [daily])
        mockAPIService.getDailyQuestionsResult = .success(response)

        // When
        viewModel.fetchDaily()

        // Then
        XCTAssertEqual(viewModel.dailyQuestions.count, 1)
        XCTAssertEqual(viewModel.dailyQuestions.first?.id, 100)
        XCTAssertNil(viewModel.error)
    }
}
