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
}
