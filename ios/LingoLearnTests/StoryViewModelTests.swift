import XCTest
import Combine
@testable import LingoLearn

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

        // When
        viewModel.getStories()

        // Then
        XCTAssertEqual(viewModel.stories.count, 1)
        XCTAssertEqual(viewModel.stories.first?.title, "test")
        XCTAssertNil(viewModel.error)
    }
}
