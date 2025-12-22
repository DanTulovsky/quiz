import Combine
import Foundation

class VocabularyViewModel: BaseViewModel, SnippetLoading, LanguageCaching, ListFetching,
    LanguageFetching, Searchable, Filterable
{
    typealias Item = StorySummary

    @Published var snippets = [Snippet]()
    @Published var stories = [StorySummary]()

    var items: [StorySummary] {
        get { stories }
        set { stories = newValue }
    }
    @Published var searchQuery = ""

    var searchQueryPublisher: Published<String>.Publisher {
        $searchQuery
    }

    @Published var selectedStoryId: Int? = nil
    @Published var selectedLevel: String? = nil
    @Published var selectedSourceLang: String? = nil
    @Published var availableLanguages: [LanguageInfo] = [] {
        didSet {
            updateLanguageCache()
        }
    }

    var languageCacheByCode: [String: LanguageInfo] = [:]
    var languageCacheByName: [String: LanguageInfo] = [:]

    override init(apiService: APIService = APIService.shared) {
        super.init(apiService: apiService)

        setupSearchDebounce(delay: 0.5)
        setupFilterDebounce($selectedStoryId, $selectedLevel, $selectedSourceLang, delay: 0.1)
            .store(in: &cancellables)
    }

    func fetchItemsPublisher() -> AnyPublisher<[StorySummary], APIService.APIError> {
        return apiService.getStories()
    }

    func performSearch() {
        getSnippets()
    }

    func performFilter() {
        getSnippets()
    }

    func getSnippets() {
        let searchQueryTrimmed = searchQuery.trimmingCharacters(in: .whitespacesAndNewlines)
        let query = searchQueryTrimmed.isEmpty ? nil : searchQueryTrimmed
        apiService.getSnippets(
            sourceLang: selectedSourceLang, targetLang: nil, storyId: selectedStoryId, query: query,
            level: selectedLevel
        )
        .handleLoadingAndError(on: self)
        .sinkValue(on: self) { [weak self] snippetList in
            self?.snippets = snippetList.snippets
        }
        .store(in: &cancellables)
    }

    func createSnippet(
        request: CreateSnippetRequest,
        completion: @escaping (Result<Snippet, APIService.APIError>) -> Void
    ) {
        apiService.createSnippet(request: request)
            .executeWithCompletion(
                on: self,
                receiveValue: { [weak self] snippet in
                    self?.snippets.insert(snippet, at: 0)
                },
                completion: completion
            )
            .store(in: &cancellables)
    }

    func updateSnippet(
        id: Int, request: UpdateSnippetRequest,
        completion: @escaping (Result<Snippet, APIService.APIError>) -> Void
    ) {
        apiService.updateSnippet(id: id, request: request)
            .executeWithCompletion(
                on: self,
                receiveValue: { [weak self] updatedSnippet in
                    if let index = self?.snippets.firstIndex(where: { $0.id == id }) {
                        self?.snippets[index] = updatedSnippet
                    }
                },
                completion: completion
            )
            .store(in: &cancellables)
    }

    func deleteSnippet(id: Int, completion: @escaping (Result<Void, APIService.APIError>) -> Void) {
        apiService.deleteSnippet(id: id)
            .executeWithCompletion(
                on: self,
                receiveValue: { [weak self] _ in
                    self?.snippets.removeAll { $0.id == id }
                },
                completion: completion
            )
            .store(in: &cancellables)
    }

    var filteredSnippets: [Snippet] {
        // Server-side filtering is now handled by the API
        // This just returns the snippets from the server
        return snippets
    }
}
