import XCTest
import Combine
@testable import LingoLearn

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
        let conv = Conversation(id: "1", userId: 1, title: "test", createdAt: Date(), updatedAt: Date(), messageCount: 1, messages: nil)
        let response = ConversationListResponse(conversations: [conv], total: 1)
        mockAPIService.getAIConversationsResult = .success(response)

        // When
        viewModel.fetchConversations()

        // Then
        XCTAssertEqual(viewModel.conversations.count, 1)
        XCTAssertEqual(viewModel.conversations.first?.title, "test")
    }
}


        set { Self._convResult = newValue }
    }
    
    override func getAIConversations() -> AnyPublisher<ConversationListResponse, APIError> {
        return getAIConversationsResult!.publisher.eraseToAnyPublisher()
    }
}
