import XCTest
import Combine
@testable import LingoLearn

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
        let snippet = Snippet(id: 1, originalText: "hello", translatedText: "ciao", context: nil, sourceLanguage: "en", targetLanguage: "it")
        let list = SnippetList(limit: 10, offset: 0, query: nil, snippets: [snippet])
        mockAPIService.getSnippetsResult = .success(list)

        // When
        viewModel.getSnippets()

        // Then
        XCTAssertEqual(viewModel.snippets.count, 1)
        XCTAssertEqual(viewModel.snippets.first?.originalText, "hello")
        XCTAssertNil(viewModel.error)
    }
}
