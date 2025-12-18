import Foundation
import Combine

class AIHistoryViewModel: ObservableObject {
    @Published var conversations: [Conversation] = []
    @Published var bookmarks: [ChatMessage] = []
    @Published var selectedConversation: Conversation?
    @Published var isLoading = false
    @Published var isDeleting = false
    @Published var error: APIService.APIError?

    private var apiService: APIService
    private var cancellables = Set<AnyCancellable>()

    init(apiService: APIService = .shared) {
        self.apiService = apiService
    }

    func fetchConversations() {
        isLoading = true
        error = nil

        apiService.getAIConversations()
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isLoading = false
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] response in
                self?.conversations = response.conversations
            })
            .store(in: &cancellables)
    }

    func fetchConversation(id: String) {
        isLoading = true
        error = nil

        apiService.getAIConversation(id: id)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isLoading = false
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] conversation in
                self?.selectedConversation = conversation
            })
            .store(in: &cancellables)
    }

    func updateTitle(id: String, newTitle: String) {
        apiService.updateAIConversationTitle(id: id, title: newTitle)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] _ in
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
                self?.conversations.removeAll(where: { $0.id == id })
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
}
