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
        let expectation = XCTestExpectation(description: "Word of the day fetched")

        // When
        viewModel.fetchWordOfTheDay()

        // Then - wait for async operation
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertEqual(self.viewModel.wordOfTheDay?.word, "test")
            XCTAssertNil(self.viewModel.error)
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }
}
