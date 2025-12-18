import Foundation
import Combine

class VocabularyViewModel: ObservableObject {
    @Published var snippets = [Snippet]()
    @Published var stories = [StorySummary]()
    @Published var isLoading = false
    @Published var error: APIService.APIError?
    @Published var searchQuery = ""

    @Published var selectedStoryId: Int? = nil
    @Published var selectedLevel: String? = nil
    @Published var selectedSourceLang: Language? = nil

    private var apiService: APIService
    private var cancellables = Set<AnyCancellable>()

    init(apiService: APIService = APIService.shared) {
        self.apiService = apiService

        // Handle search with debounce
        $searchQuery
            .debounce(for: .milliseconds(500), scheduler: RunLoop.main)
            .removeDuplicates()
            .sink { [weak self] _ in
                self?.getSnippets()
            }
            .store(in: &cancellables)

        // Handle filter changes
        Publishers.CombineLatest3($selectedStoryId, $selectedLevel, $selectedSourceLang)
            .dropFirst()
            .debounce(for: .milliseconds(100), scheduler: RunLoop.main)
            .sink { [weak self] _, _, _ in
                self?.getSnippets()
            }
            .store(in: &cancellables)
    }

    func fetchStories() {
        apiService.getStories()
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] stories in
                self?.stories = stories
            })
            .store(in: &cancellables)
    }

    func getSnippets() {
        isLoading = true
        apiService.getSnippets(sourceLang: selectedSourceLang, targetLang: nil, storyId: selectedStoryId)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isLoading = false
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] snippetList in
                self?.snippets = snippetList.snippets
            })
            .store(in: &cancellables)
    }

    func createSnippet(request: CreateSnippetRequest, completion: @escaping (Result<Snippet, APIService.APIError>) -> Void) {
        apiService.createSnippet(request: request)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { result in
                if case .failure(let error) = result {
                    completion(.failure(error))
                }
            }, receiveValue: { [weak self] snippet in
                self?.snippets.insert(snippet, at: 0)
                completion(.success(snippet))
            })
            .store(in: &cancellables)
    }

    var filteredSnippets: [Snippet] {
        var results = snippets

        // Filter by search query
        if !searchQuery.isEmpty {
            results = results.filter {
                $0.originalText.localizedCaseInsensitiveContains(searchQuery) ||
                $0.translatedText.localizedCaseInsensitiveContains(searchQuery)
            }
        }

        // Filter by level (client-side)
        if let level = selectedLevel, !level.isEmpty {
            results = results.filter { $0.difficultyLevel?.uppercased() == level.uppercased() }
        }

        return results
    }
}
