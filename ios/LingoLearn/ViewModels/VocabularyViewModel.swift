import Foundation
import Combine

class VocabularyViewModel: ObservableObject {
    @Published var snippets = [Snippet]()
    @Published var isLoading = false
    @Published var error: APIService.APIError?
    @Published var searchQuery = ""

    private var apiService: APIService
    private var cancellables = Set<AnyCancellable>()

    init(apiService: APIService = APIService.shared) {
        self.apiService = apiService

        // Handle search with debounce
        $searchQuery
            .debounce(for: .milliseconds(500), scheduler: RunLoop.main)
            .removeDuplicates()
            .sink { [weak self] query in
                self?.getSnippets()
            }
            .store(in: &cancellables)
    }

    func getSnippets() {
        isLoading = true
        apiService.getSnippets(sourceLang: nil, targetLang: nil)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isLoading = false
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] snippetList in
                self?.snippets = snippetList.snippets
            })
            .store(in: &cancellables)
    }

    var filteredSnippets: [Snippet] {
        if searchQuery.isEmpty {
            return snippets
        } else {
            return snippets.filter {
                $0.originalText.localizedCaseInsensitiveContains(searchQuery) ||
                $0.translatedText.localizedCaseInsensitiveContains(searchQuery)
            }
        }
    }
}
