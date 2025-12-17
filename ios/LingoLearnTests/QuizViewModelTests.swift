import XCTest
import Combine
@testable import LingoLearn

class QuizViewModelTests: XCTestCase {
    var viewModel: QuizViewModel!
    var mockAPIService: MockAPIService!

    override func setUp() {
        super.setUp()
        mockAPIService = MockAPIService()
        viewModel = QuizViewModel(apiService: mockAPIService)
    }

    override func tearDown() {
        viewModel = nil
        mockAPIService = nil
        super.tearDown()
    }

    func testGetQuestionSuccess() {
        // Given
        let question = Question(id: 1, type: "vocabulary", language: "it", level: "A1", content: [:], correctAnswerIndex: 1)
        mockAPIService.getQuestionResult = .success(.question(question))

        // When
        viewModel.getQuestion()

        // Then
        XCTAssertNotNil(viewModel.question)
        XCTAssertEqual(viewModel.question?.id, 1)
        XCTAssertNil(viewModel.error)
        XCTAssertNil(viewModel.generatingMessage)
    }

    func testGetQuestionGenerating() {
        // Given
        mockAPIService.getQuestionResult = .success(.generating(GeneratingStatusResponse(message: "Wait", status: "generating")))

        // When
        viewModel.getQuestion()

        // Then
        XCTAssertNil(viewModel.question)
        XCTAssertEqual(viewModel.generatingMessage, "Wait")
    }
}
