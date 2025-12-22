import Combine
import Foundation

class AIHistoryViewModel: BaseViewModel, Refreshable, ListFetchingWithName, DetailFetching,
    OptimisticUpdating
{
    typealias Item = Conversation
    typealias DetailID = String
    typealias DetailItem = Conversation

    @Published var conversations: [Conversation] = []
    @Published var bookmarks: [ChatMessage] = []
    @Published var selectedConversation: Conversation?
    @Published var isDeleting = false

    var items: [Conversation] {
        get { conversations }
        set { conversations = newValue }
    }

    var selectedDetail: Conversation? {
        get { selectedConversation }
        set { selectedConversation = newValue }
    }

    override init(apiService: APIService = .shared) {
        super.init(apiService: apiService)
    }

    func fetchItemsPublisher() -> AnyPublisher<[Conversation], APIService.APIError> {
        return apiService.getAIConversations()
            .map { $0.conversations }
            .eraseToAnyPublisher()
    }

    func updateItems(_ items: [Conversation]) {
        conversations = items
    }

    func fetchDetailPublisher(id: String) -> AnyPublisher<Conversation, APIService.APIError> {
        return apiService.getAIConversation(id: id)
    }

    func fetchConversation(id: String) {
        fetchDetail(id: id)
    }

    func updateTitle(id: String, newTitle: String) {
        applyOptimisticUpdate(id: id) { oldConversation in
            Conversation(
                id: oldConversation.id,
                userId: oldConversation.userId,
                title: newTitle,
                createdAt: oldConversation.createdAt,
                updatedAt: oldConversation.updatedAt,
                messageCount: oldConversation.messageCount,
                messages: oldConversation.messages
            )
        }

        apiService.updateAIConversationTitle(id: id, title: newTitle)
            .handleErrorOnly(on: self)
            .handleEvents(receiveCompletion: { [weak self] completion in
                if case .failure = completion {
                    // On error, refetch to restore correct state
                    self?.fetchItems()
                }
            })
            .sinkValue(on: self) { [weak self] _ in
                // Refetch to ensure we have the latest data from server
                self?.fetchItems()
                if self?.selectedConversation?.id == id {
                    self?.fetchConversation(id: id)
                }
            }
            .store(in: &cancellables)
    }

    func deleteConversation(id: String) {
        isDeleting = true
        apiService.deleteAIConversation(id: id)
            .handleErrorOnly(on: self)
            .handleEvents(receiveCompletion: { [weak self] _ in
                self?.isDeleting = false
            })
            .sinkVoid(on: self) { [weak self] in
                // Refetch conversations to ensure instant update
                self?.fetchItems()
            }
            .store(in: &cancellables)
    }

    func fetchBookmarks() {
        apiService.getBookmarkedMessages()
            .handleLoadingAndError(on: self)
            .sinkValue(on: self) { [weak self] response in
                self?.bookmarks = response.messages
            }
            .store(in: &cancellables)
    }

    func toggleBookmark(conversationId: String, messageId: String) {
        apiService.toggleBookmark(conversationId: conversationId, messageId: messageId)
            .handleErrorOnly(on: self)
            .sinkValue(on: self) { [weak self] response in
                if response.bookmarked {
                    // Message was bookmarked - refresh bookmarks list
                    self?.fetchBookmarks()
                } else {
                    // Message was unbookmarked - remove from local list
                    self?.bookmarks.removeAll(where: { $0.id == messageId })
                }
            }
            .store(in: &cancellables)
    }

    func refreshData() {
        fetchItems()
    }
}
