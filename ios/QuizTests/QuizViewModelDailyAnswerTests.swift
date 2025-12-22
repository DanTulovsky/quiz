import XCTest
import Combine
@testable import Quiz

class QuizViewModelDailyAnswerTests: XCTestCase {
    var viewModel: QuizViewModel!
    var mockAPIService: MockAPIService!
    var cancellables: Set<AnyCancellable>!

    override func setUp() {
        super.setUp()
        mockAPIService = MockAPIService()
        viewModel = QuizViewModel(questionType: nil, isDaily: true, apiService: mockAPIService)
        cancellables = Set<AnyCancellable>()
    }

    override func tearDown() {
        cancellables = nil
        viewModel = nil
        mockAPIService = nil
        super.tearDown()
    }

    func testSubmitDailyAnswerUsesCorrectAnswerIndex() {
        // Given
        let expectation = XCTestExpectation(description: "Answer submitted")
        let testQuestion = Question(
            id: 1,
            type: "vocabulary",
            language: "en",
            level: "A1",
            content: ["question": .string("Test")],
            correctAnswerIndex: 2
        )
        viewModel.question = testQuestion

        let expectedResponse = DailyAnswerResponse(
            isCorrect: true,
            explanation: "Correct!",
            isCompleted: true,
            correctAnswerIndex: 2,
            userAnswer: "Answer 2",
            userAnswerIndex: 2
        )

        mockAPIService.dailyAnswerResponse = expectedResponse

        // When
        viewModel.submitDailyAnswer(userAnswerIndex: 2)

        // Wait for response
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            expectation.fulfill()
        }

        wait(for: [expectation], timeout: 1.0)

        // Then - verify the answerResponse uses the correct answer index from response
        XCTAssertNotNil(viewModel.answerResponse, "Answer response should be set")
        if let response = viewModel.answerResponse {
            XCTAssertEqual(
                response.correctAnswerIndex,
                expectedResponse.correctAnswerIndex,
                "Should use correct answer index from response, not placeholder"
            )
            XCTAssertEqual(response.userAnswer, expectedResponse.userAnswer, "Should use user answer from response")
            XCTAssertEqual(
                response.userAnswerIndex,
                expectedResponse.userAnswerIndex,
                "Should use user answer index from response"
            )
        }
    }
}
