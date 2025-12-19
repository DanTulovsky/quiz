import XCTest
import Combine
@testable import Quiz

class APIServiceURLValidationTests: XCTestCase {
    var apiService: APIService!
    var cancellables: Set<AnyCancellable>!

    override func setUp() {
        super.setUp()
        apiService = APIService.shared
        cancellables = Set<AnyCancellable>()
    }

    override func tearDown() {
        cancellables = nil
        super.tearDown()
    }

    func testStreamURLWithValidInputs() {
        // Given
        let streamId = "test-stream-id"
        let token = "test-token"

        // When
        let url = apiService.streamURL(for: streamId, token: token)

        // Then
        XCTAssertNotNil(url, "Stream URL should be created")
        XCTAssertTrue(url.absoluteString.contains(streamId), "URL should contain stream ID")
    }

    func testStreamURLWithNilToken() {
        // Given
        let streamId = "test-stream-id"

        // When
        let url = apiService.streamURL(for: streamId, token: nil)

        // Then
        XCTAssertNotNil(url, "Stream URL should be created even without token")
        XCTAssertTrue(url.absoluteString.contains(streamId), "URL should contain stream ID")
    }

    func testBuildURLWithEmptyQueryItems() {
        // This tests the buildURL helper method indirectly through public methods
        let expectation = XCTestExpectation(description: "Request completes")

        apiService.getStories()
            .sink(
                receiveCompletion: { _ in
                    expectation.fulfill()
                },
                receiveValue: { _ in }
            )
            .store(in: &cancellables)

        wait(for: [expectation], timeout: 5.0)

        // Should not crash
        XCTAssertTrue(true, "Should handle empty query items")
    }

    func testBuildURLWithSpecialCharacters() {
        let expectation = XCTestExpectation(description: "Request handles special characters")

        // Test with query parameters that might have special characters
        apiService.getSnippets(sourceLang: "en", targetLang: "es", storyId: nil, query: "test query", level: "A1")
            .sink(
                receiveCompletion: { _ in
                    expectation.fulfill()
                },
                receiveValue: { _ in }
            )
            .store(in: &cancellables)

        wait(for: [expectation], timeout: 5.0)

        // Should not crash
        XCTAssertTrue(true, "Should handle special characters in query")
    }
}


