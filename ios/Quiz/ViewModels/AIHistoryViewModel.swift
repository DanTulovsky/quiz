import Foundation
import Combine

class AIHistoryViewModel: BaseViewModel {
    @Published var conversations: [Conversation] = []
    @Published var bookmarks: [ChatMessage] = []
    @Published var selectedConversation: Conversation?
    @Published var isDeleting = false

    override init(apiService: APIService = .shared) {
        super.init(apiService: apiService)
    }

    func fetchConversations() {
        apiService.getAIConversations()
            .handleLoadingAndError(on: self)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] response in
                self?.conversations = response.conversations
            })
            .store(in: &cancellables)
    }

    func fetchConversation(id: String) {
        apiService.getAIConversation(id: id)
            .handleLoadingAndError(on: self)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] conversation in
                self?.selectedConversation = conversation
            })
            .store(in: &cancellables)
    }

    func updateTitle(id: String, newTitle: String) {
        // Optimistically update the local array immediately for instant UI feedback
        if let index = conversations.firstIndex(where: { $0.id == id }) {
            let oldConversation = conversations[index]
            // Create a new Conversation with updated title
            let updatedConversation = Conversation(
                id: oldConversation.id,
                userId: oldConversation.userId,
                title: newTitle,
                createdAt: oldConversation.createdAt,
                updatedAt: oldConversation.updatedAt,
                messageCount: oldConversation.messageCount,
                messages: oldConversation.messages
            )
            // Create a new array to ensure SwiftUI detects the change
            var updatedConversations = conversations
            updatedConversations[index] = updatedConversation
            conversations = updatedConversations
        }

        apiService.updateAIConversationTitle(id: id, title: newTitle)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                if case .failure(let error) = completion {
                    self?.error = error
                    // On error, refetch to restore correct state
                    self?.fetchConversations()
                }
            }, receiveValue: { [weak self] _ in
                // Refetch to ensure we have the latest data from server
                self?.fetchConversations()
                if self?.selectedConversation?.id == id {
                    self?.fetchConversation(id: id)
                }
            })
            .store(in: &cancellables)
    }

    func deleteConversation(id: String) {
        isDeleting = true
        apiService.deleteAIConversation(id: id)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isDeleting = false
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] _ in
                // Refetch conversations to ensure instant update
                self?.fetchConversations()
            })
            .store(in: &cancellables)
    }

    func fetchBookmarks() {
        isLoading = true
        error = nil

        apiService.getBookmarkedMessages()
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isLoading = false
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] response in
                self?.bookmarks = response.messages
            })
            .store(in: &cancellables)
    }

    func toggleBookmark(conversationId: String, messageId: String) {
        apiService.toggleBookmark(conversationId: conversationId, messageId: messageId)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] response in
                if response.bookmarked {
                    // Message was bookmarked - refresh bookmarks list
                    self?.fetchBookmarks()
                } else {
                    // Message was unbookmarked - remove from local list
                    self?.bookmarks.removeAll(where: { $0.id == messageId })
                }
            })
            .store(in: &cancellables)
    }

    override func cancelAllRequests() {
        cancellables.removeAll()
    }
}
