import Combine
import XCTest

@testable import Quiz

class SnippetLoadingTests: XCTestCase {
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

    func testLoadSnippetsWithErrorSilentlyHandled() {
        // Given - snippet loading should fail but not show error
        mockAPIService.getSnippetsResult = .failure(.decodingFailed(
            NSError(domain: "Test", code: -1, userInfo: [NSLocalizedDescriptionKey: "Decoding failed"])
        ))

        // When
        viewModel.loadSnippets(questionId: 1)

        // Then - error should be silently handled, snippets should be empty, no error set
        let expectation = XCTestExpectation(description: "Snippets error handled silently")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertEqual(self.viewModel.snippets.count, 0, "Snippets should be empty on error")
            XCTAssertNil(self.viewModel.error, "Error should not be set for snippet loading failures")
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }

    func testLoadSnippetsWithEmptyResponse() {
        // Given - empty snippet list
        let emptyList = SnippetList(limit: 0, offset: 0, query: nil, snippets: [])
        mockAPIService.getSnippetsResult = .success(emptyList)

        // When
        viewModel.loadSnippets(questionId: 1)

        // Then
        let expectation = XCTestExpectation(description: "Empty snippets loaded")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertEqual(self.viewModel.snippets.count, 0)
            XCTAssertNil(self.viewModel.error)
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }

    func testLoadSnippetsWithSuccess() {
        // Given
        let snippet = Snippet(
            id: 1, originalText: "hello", translatedText: "ciao", context: "test context",
            sourceLanguage: "en", targetLanguage: "it", difficultyLevel: "A1",
            questionId: 1, storyId: nil, sectionId: nil)
        let list = SnippetList(limit: 10, offset: 0, query: nil, snippets: [snippet])
        mockAPIService.getSnippetsResult = .success(list)

        // When
        viewModel.loadSnippets(questionId: 1)

        // Then
        let expectation = XCTestExpectation(description: "Snippets loaded successfully")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertEqual(self.viewModel.snippets.count, 1)
            XCTAssertEqual(self.viewModel.snippets.first?.id, 1)
            XCTAssertEqual(self.viewModel.snippets.first?.originalText, "hello")
            XCTAssertNil(self.viewModel.error)
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }

    func testLoadSnippetsUpdatesOnMainThread() {
        // Given
        let snippet = Snippet(
            id: 1, originalText: "test", translatedText: "testo", context: nil,
            sourceLanguage: "en", targetLanguage: "it", difficultyLevel: nil,
            questionId: 1, storyId: nil, sectionId: nil)
        let list = SnippetList(limit: 10, offset: 0, query: nil, snippets: [snippet])
        mockAPIService.getSnippetsResult = .success(list)
        var wasOnMainThread = false

        // When
        viewModel.loadSnippets(questionId: 1)

        // Then - verify update happens on main thread
        let expectation = XCTestExpectation(description: "Snippets updated on main thread")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            wasOnMainThread = Thread.isMainThread
            XCTAssertTrue(wasOnMainThread, "Snippet update should be on main thread")
            XCTAssertEqual(self.viewModel.snippets.count, 1)
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }

    func testLoadSnippetsByQuestionId() {
        // Given
        let snippet = Snippet(
            id: 1, originalText: "question text", translatedText: "testo domanda", context: nil,
            sourceLanguage: "en", targetLanguage: "it", difficultyLevel: nil,
            questionId: 123, storyId: nil, sectionId: nil)
        let list = SnippetList(limit: 10, offset: 0, query: nil, snippets: [snippet])
        mockAPIService.getSnippetsResult = .success(list)

        // When
        viewModel.loadSnippets(questionId: 123)

        // Then
        let expectation = XCTestExpectation(description: "Snippets loaded by question ID")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertEqual(self.viewModel.snippets.count, 1)
            XCTAssertEqual(self.viewModel.snippets.first?.questionId, 123)
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }

    func testLoadSnippetsByStoryId() {
        // Given
        let snippet = Snippet(
            id: 1, originalText: "story text", translatedText: "testo storia", context: nil,
            sourceLanguage: "en", targetLanguage: "it", difficultyLevel: nil,
            questionId: nil, storyId: 456, sectionId: nil)
        let list = SnippetList(limit: 10, offset: 0, query: nil, snippets: [snippet])
        mockAPIService.getSnippetsResult = .success(list)

        // When
        viewModel.loadSnippets(storyId: 456)

        // Then
        let expectation = XCTestExpectation(description: "Snippets loaded by story ID")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertEqual(self.viewModel.snippets.count, 1)
            XCTAssertEqual(self.viewModel.snippets.first?.storyId, 456)
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }

    func testLoadSnippetsWithNetworkError() {
        // Given - network error
        mockAPIService.getSnippetsResult = .failure(.requestFailed(
            NSError(domain: NSURLErrorDomain, code: NSURLErrorNotConnectedToInternet, userInfo: nil)
        ))

        // When
        viewModel.loadSnippets(questionId: 1)

        // Then - should silently handle network errors
        let expectation = XCTestExpectation(description: "Network error handled silently")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertEqual(self.viewModel.snippets.count, 0, "Snippets should be empty on network error")
            XCTAssertNil(self.viewModel.error, "Error should not be set for snippet loading failures")
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }
}

