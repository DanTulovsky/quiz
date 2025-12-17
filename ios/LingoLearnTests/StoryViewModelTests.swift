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
        let stories = [Story(id: 1, title: "Test Story", language: .en, level: .a1, sections: [])]
        let storyList = StoryList(stories: stories)
        mockAPIService.getStoriesResult = .success(storyList)

        // When
        viewModel.getStories()

        // Then
        XCTAssertEqual(viewModel.stories.count, 1)
        XCTAssertEqual(viewModel.stories.first?.title, "Test Story")
        XCTAssertNil(viewModel.error)
    }

    func testGetStoriesFailure() {
        // Given
        mockAPIService.getStoriesResult = .failure(.invalidResponse)

        // When
        viewModel.getStories()

        // Then
        XCTAssertEqual(viewModel.stories.count, 0)
        XCTAssertNotNil(viewModel.error)
    }
}

extension MockAPIService {
    var getStoriesResult: Result<StoryList, APIError>?
    
    override func getStories(language: Language?, level: Level?) -> AnyPublisher<StoryList, APIError> {
        return getStoriesResult!.publisher.eraseToAnyPublisher()
    }
}
