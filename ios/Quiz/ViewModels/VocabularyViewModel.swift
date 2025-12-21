import Foundation
import Combine

class VocabularyViewModel: BaseViewModel, SnippetLoading {
    @Published var snippets = [Snippet]()
    @Published var stories = [StorySummary]()
    @Published var searchQuery = ""

    @Published var selectedStoryId: Int? = nil
    @Published var selectedLevel: String? = nil
    @Published var selectedSourceLang: String? = nil
    @Published var availableLanguages: [LanguageInfo] = [] {
        didSet {
            updateLanguageCache()
        }
    }

    private var languageCacheByCode: [String: LanguageInfo] = [:]
    private var languageCacheByName: [String: LanguageInfo] = [:]

    private func updateLanguageCache() {
        languageCacheByCode.removeAll()
        languageCacheByName.removeAll()
        for lang in availableLanguages {
            languageCacheByCode[lang.code.lowercased()] = lang
            languageCacheByName[lang.name.lowercased()] = lang
        }
    }

    override init(apiService: APIService = APIService.shared) {
        super.init(apiService: apiService)

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
            .handleErrorOnly(on: self)
            .sinkValue(on: self) { [weak self] stories in
                self?.stories = stories
            }
            .store(in: &cancellables)
    }

    func getSnippets() {
        let searchQueryTrimmed = searchQuery.trimmingCharacters(in: .whitespacesAndNewlines)
        let query = searchQueryTrimmed.isEmpty ? nil : searchQueryTrimmed
        apiService.getSnippets(sourceLang: selectedSourceLang, targetLang: nil, storyId: selectedStoryId, query: query, level: selectedLevel)
            .handleLoadingAndError(on: self)
            .sinkValue(on: self) { [weak self] snippetList in
                self?.snippets = snippetList.snippets
            }
            .store(in: &cancellables)
    }

    func fetchLanguages() {
        apiService.getLanguages()
            .handleErrorOnly(on: self)
            .sinkValue(on: self) { [weak self] languages in
                self?.availableLanguages = languages
            }
            .store(in: &cancellables)
    }

    func createSnippet(request: CreateSnippetRequest, completion: @escaping (Result<Snippet, APIService.APIError>) -> Void) {
        apiService.createSnippet(request: request)
            .handleErrorOnly(on: self)
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

    func updateSnippet(id: Int, request: UpdateSnippetRequest, completion: @escaping (Result<Snippet, APIService.APIError>) -> Void) {
        apiService.updateSnippet(id: id, request: request)
            .handleErrorOnly(on: self)
            .sink(receiveCompletion: { result in
                if case .failure(let error) = result {
                    completion(.failure(error))
                }
            }, receiveValue: { [weak self] updatedSnippet in
                if let index = self?.snippets.firstIndex(where: { $0.id == id }) {
                    self?.snippets[index] = updatedSnippet
                }
                completion(.success(updatedSnippet))
            })
            .store(in: &cancellables)
    }

    func deleteSnippet(id: Int, completion: @escaping (Result<Void, APIService.APIError>) -> Void) {
        apiService.deleteSnippet(id: id)
            .handleErrorOnly(on: self)
            .sink(receiveCompletion: { result in
                if case .failure(let error) = result {
                    completion(.failure(error))
                }
            }, receiveValue: { [weak self] _ in
                self?.snippets.removeAll { $0.id == id }
                completion(.success(()))
            })
            .store(in: &cancellables)
    }

    var filteredSnippets: [Snippet] {
        // Server-side filtering is now handled by the API
        // This just returns the snippets from the server
        return snippets
    }
}
