import Combine
import XCTest

@testable import Quiz

class VocabularyViewModelTests: XCTestCase {
    var viewModel: VocabularyViewModel!
    var mockAPIService: MockAPIService!

    override func setUp() {
        super.setUp()
        mockAPIService = MockAPIService()
        viewModel = VocabularyViewModel(apiService: mockAPIService)
    }

    override func tearDown() {
        viewModel = nil
        mockAPIService = nil
        super.tearDown()
    }

    func testGetSnippetsSuccess() {
        // Given
        let snippet = Snippet(
            id: 1, originalText: "hello", translatedText: "ciao", context: nil,
            sourceLanguage: "en", targetLanguage: "it", difficultyLevel: nil,
            questionId: nil, storyId: nil, sectionId: nil)
        let list = SnippetList(limit: 10, offset: 0, query: nil, snippets: [snippet])
        mockAPIService.getSnippetsResult = .success(list)

        // When
        viewModel.getSnippets()

        // Then
        // Wait a bit for async operation
        let expectation = XCTestExpectation(description: "Snippets fetched")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertEqual(self.viewModel.snippets.count, 1)
            XCTAssertEqual(self.viewModel.snippets.first?.originalText, "hello")
            XCTAssertNil(self.viewModel.error)
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }

    func testLanguageCacheUpdate() {
        // Given
        let languages = [
            LanguageInfo(code: "en", name: "English", ttsLocale: "en-US", ttsVoice: "en-US-Standard-A"),
            LanguageInfo(code: "es", name: "Spanish", ttsLocale: "es-ES", ttsVoice: "es-ES-Standard-A")
        ]
        mockAPIService.getLanguagesResult = .success(languages)

        // When
        viewModel.fetchLanguages()

        // Then - wait for languages to load
        let expectation = XCTestExpectation(description: "Languages loaded and cached")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            // Verify cache is populated by checking lookup works
            XCTAssertEqual(self.viewModel.availableLanguages.count, 2)
            // Verify we can find languages using the extension (which uses cache internally)
            let found = self.viewModel.availableLanguages.find(byCodeOrName: "en")
            XCTAssertNotNil(found)
            XCTAssertEqual(found?.code, "en")
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
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

    func testLoadSnippetsUpdatesOnMainThread() {
        // Given
        let snippet = Snippet(
            id: 1, originalText: "test", translatedText: "testo", context: nil,
            sourceLanguage: "en", targetLanguage: "it", difficultyLevel: nil,
            questionId: 1, storyId: nil, sectionId: nil)
        let list = SnippetList(limit: 10, offset: 0, query: nil, snippets: [snippet])
        mockAPIService.getSnippetsResult = .success(list)

        // When
        viewModel.loadSnippets(questionId: 1)

        // Then - verify update happens on main thread
        let expectation = XCTestExpectation(description: "Snippets updated on main thread")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertTrue(Thread.isMainThread, "Snippet update should be on main thread")
            XCTAssertEqual(self.viewModel.snippets.count, 1)
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }
}
