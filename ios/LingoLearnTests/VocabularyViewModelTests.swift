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
        let snippets = [Snippet(id: 1, text: "test", translation: "test", sourceLanguage: .en, targetLanguage: .es)]
        let snippetList = SnippetList(snippets: snippets)
        mockAPIService.getSnippetsResult = .success(snippetList)

        // When
        viewModel.getSnippets()

        // Then
        XCTAssertEqual(viewModel.snippets.count, 1)
        XCTAssertEqual(viewModel.snippets.first?.text, "test")
        XCTAssertNil(viewModel.error)
    }

    func testGetSnippetsFailure() {
        // Given
        mockAPIService.getSnippetsResult = .failure(.invalidResponse)

        // When
        viewModel.getSnippets()

        // Then
        XCTAssertEqual(viewModel.snippets.count, 0)
        XCTAssertNotNil(viewModel.error)
    }
}

extension MockAPIService {
    var getSnippetsResult: Result<SnippetList, APIError>?
    
    override func getSnippets(sourceLang: Language?, targetLang: Language?) -> AnyPublisher<SnippetList, APIError> {
        return getSnippetsResult!.publisher.eraseToAnyPublisher()
    }
}
