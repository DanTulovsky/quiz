import XCTest
import Combine
@testable import LingoLearn

class PhrasebookViewModelTests: XCTestCase {
    var viewModel: PhrasebookViewModel!
    var mockAPIService: MockAPIService!

    override func setUp() {
        super.setUp()
        mockAPIService = MockAPIService()
        viewModel = PhrasebookViewModel(apiService: mockAPIService)
    }

    override func tearDown() {
        viewModel = nil
        mockAPIService = nil
        super.tearDown()
    }

    func testGetPhrasebookSuccess() {
        // Given
        let categories = [PhrasebookCategory(name: "Greetings", phrases: [PhrasebookPhrase(phrase: "Hello", translation: "Hola")])]
        let phrasebook = PhrasebookResponse(categories: categories)
        mockAPIService.getPhrasebookResult = .success(phrasebook)

        // When
        viewModel.getPhrasebook(language: .en)

        // Then
        XCTAssertNotNil(viewModel.phrasebook)
        XCTAssertEqual(viewModel.phrasebook?.categories.count, 1)
        XCTAssertNil(viewModel.error)
    }

    func testGetPhrasebookFailure() {
        // Given
        mockAPIService.getPhrasebookResult = .failure(.invalidResponse)

        // When
        viewModel.getPhrasebook(language: .en)

        // Then
        XCTAssertNil(viewModel.phrasebook)
        XCTAssertNotNil(viewModel.error)
    }
}

extension MockAPIService {
    var getPhrasebookResult: Result<PhrasebookResponse, APIError>?
    
    override func getPhrasebook(language: Language) -> AnyPublisher<PhrasebookResponse, APIError> {
        return getPhrasebookResult!.publisher.eraseToAnyPublisher()
    }
}
