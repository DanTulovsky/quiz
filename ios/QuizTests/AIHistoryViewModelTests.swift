import Combine
import XCTest

@testable import Quiz

class AIHistoryViewModelTests: XCTestCase {
    var viewModel: AIHistoryViewModel!
    var mockAPIService: MockAPIService!

    override func setUp() {
        super.setUp()
        mockAPIService = MockAPIService()
        viewModel = AIHistoryViewModel(apiService: mockAPIService)
    }

    override func tearDown() {
        viewModel = nil
        mockAPIService = nil
        super.tearDown()
    }

    func testFetchConversationsSuccess() {
        // Given
        let conv = Conversation(
            id: "1", userId: 1, title: "test", createdAt: Date(), updatedAt: Date(),
            messageCount: 1, messages: nil)
        let response = ConversationListResponse(conversations: [conv], total: 1)
        mockAPIService.getAIConversationsResult = .success(response)
        let expectation = XCTestExpectation(description: "Conversations fetched")

        // When
        viewModel.fetchConversations()

        // Then - wait for async operation
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertEqual(self.viewModel.conversations.count, 1)
            XCTAssertEqual(self.viewModel.conversations.first?.title, "test")
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }
}
