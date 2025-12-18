import Combine
import XCTest

@testable import Quiz

class WordOfTheDayViewModelTests: XCTestCase {
    var viewModel: WordOfTheDayViewModel!
    var mockAPIService: MockAPIService!

    override func setUp() {
        super.setUp()
        mockAPIService = MockAPIService()
        viewModel = WordOfTheDayViewModel(apiService: mockAPIService)
    }

    override func tearDown() {
        viewModel = nil
        mockAPIService = nil
        super.tearDown()
    }

    func testFetchWordOfTheDaySuccess() {
        // Given
        let wotd = WordOfTheDayDisplay(
            date: "2025-12-17", word: "test", translation: "test", sentence: "test",
            sourceType: "test", sourceId: 1, language: "it", level: "A1", context: nil,
            explanation: nil, topicCategory: nil)
        mockAPIService.getWordOfTheDayResult = .success(wotd)

        // When
        viewModel.fetchWordOfTheDay()

        // Then
        XCTAssertEqual(viewModel.wordOfTheDay?.word, "test")
        XCTAssertNil(viewModel.error)
    }
}
