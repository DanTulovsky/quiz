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
        let question = Question(id: 1, text: "test question", type: "test", choices: nil)
        mockAPIService.getQuestionResult = .success(question)

        // When
        viewModel.getQuestion()

        // Then
        XCTAssertNotNil(viewModel.question)
        XCTAssertEqual(viewModel.question?.id, 1)
        XCTAssertNil(viewModel.error)
    }

    func testGetQuestionFailure() {
        // Given
        mockAPIService.getQuestionResult = .failure(.invalidResponse)

        // When
        viewModel.getQuestion()

        // Then
        XCTAssertNil(viewModel.question)
        XCTAssertNotNil(viewModel.error)
    }
}

extension MockAPIService {
    var getQuestionResult: Result<Question, APIError>?
    
    override func getQuestion(language: Language?, level: Level?, type: String?, excludeType: String?) -> AnyPublisher<Question, APIError> {
        return getQuestionResult!.publisher.eraseToAnyPublisher()
    }
}
