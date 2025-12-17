import Foundation
import Combine

class AIHistoryViewModel: ObservableObject {
    @Published var conversations: [Conversation] = []
    @Published var bookmarks: [ChatMessage] = []
    @Published var isLoading = false
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
