import Combine
import XCTest

@testable import Quiz

class StoryViewModelTests: XCTestCase {
    var viewModel: StoryViewModel!
    var mockAPIService: MockAPIService!

    override func setUp() {
        super.setUp()
        mockAPIService = MockAPIService()
        viewModel = StoryViewModel(apiService: mockAPIService)
    }

    override func tearDown() {
        viewModel = nil
        mockAPIService = nil
        super.tearDown()
    }

    func testGetStoriesSuccess() {
        // Given
        let stories = [StorySummary(id: 1, title: "test", language: "it", status: "active")]
        mockAPIService.getStoriesResult = .success(stories)
        let expectation = XCTestExpectation(description: "Stories fetched")

        // When
        viewModel.getStories()

        // Then - wait for async operation
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertEqual(self.viewModel.stories.count, 1)
            XCTAssertEqual(self.viewModel.stories.first?.title, "test")
            XCTAssertNil(self.viewModel.error)
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }
}
