import Combine
import XCTest

@testable import Quiz

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
        let question = Question(
            id: 1, type: "vocabulary", language: "it", level: "A1", content: [:],
            correctAnswerIndex: 1)
        mockAPIService.getQuestionResult = .success(.question(question))
        // Set snippets result to avoid error from loadSnippets call
        mockAPIService.getSnippetsResult = .success(SnippetList(limit: 0, offset: 0, query: nil, snippets: []))
        let expectation = XCTestExpectation(description: "Question fetched")

        // When
        viewModel.getQuestion()

        // Then - wait for async operation
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertNotNil(self.viewModel.question)
            XCTAssertEqual(self.viewModel.question?.id, 1)
            XCTAssertNil(self.viewModel.error)
            XCTAssertNil(self.viewModel.generatingMessage)
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }

    func testLoadSnippetsWithErrorDoesNotShowError() {
        // Given - question loads successfully but snippets fail
        let question = Question(
            id: 1, type: "vocabulary", language: "it", level: "A1", content: [:],
            correctAnswerIndex: 1)
        mockAPIService.getQuestionResult = .success(.question(question))
        // Snippets should fail but not show error
        mockAPIService.getSnippetsResult = .failure(.decodingFailed(
            NSError(domain: "Test", code: -1, userInfo: [NSLocalizedDescriptionKey: "Decoding failed"])
        ))

        // When
        viewModel.getQuestion()

        // Then - question should load, snippets should be empty, but no error shown
        let expectation = XCTestExpectation(description: "Question loads despite snippet error")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.2) {
            XCTAssertNotNil(self.viewModel.question, "Question should load successfully")
            XCTAssertEqual(self.viewModel.question?.id, 1)
            XCTAssertEqual(self.viewModel.snippets.count, 0, "Snippets should be empty on error")
            XCTAssertNil(self.viewModel.error, "Error should not be set - snippet errors are non-fatal")
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }

    func testGetQuestionGenerating() {
        // Given
        mockAPIService.getQuestionResult = .success(
            .generating(GeneratingStatusResponse(message: "Wait", status: "generating")))
        let expectation = XCTestExpectation(description: "Generating status received")

        // When
        viewModel.getQuestion()

        // Then - wait for async operation
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertNil(self.viewModel.question)
            XCTAssertEqual(self.viewModel.generatingMessage, "Wait")
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }
}
