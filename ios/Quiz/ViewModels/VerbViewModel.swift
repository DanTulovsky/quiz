import Combine
import Foundation

class VerbViewModel: BaseViewModel, ListFetchingWithName, LanguageFetching {
    typealias Item = VerbConjugationSummary

    @Published var verbs: [VerbConjugationSummary] = []

    var items: [VerbConjugationSummary] {
        get { verbs }
        set { verbs = newValue }
    }
    @Published var selectedVerb: String = ""
    @Published var selectedVerbDetail: VerbConjugationDetail?
    @Published var expandedTenses: Set<String> = []
    @Published var availableLanguages: [LanguageInfo] = []

    private var currentLanguage: String = "it"

    override init(apiService: APIServiceProtocol = APIService.shared) {
        super.init(apiService: apiService)

        $selectedVerb
            .dropFirst()
            .removeDuplicates()
            .sink { [weak self] verb in
                guard let self = self, !verb.isEmpty else { return }
                self.fetchVerbDetail(language: self.currentLanguage, verb: verb)
            }
            .store(in: &cancellables)
    }

    func fetchItemsPublisher() -> AnyPublisher<[VerbConjugationSummary], APIService.APIError> {
        return apiService.getVerbConjugations(language: currentLanguage)
            .map { $0.verbs }
            .eraseToAnyPublisher()
    }

    func updateItems(_ items: [VerbConjugationSummary]) {
        verbs = items
        if selectedVerb.isEmpty, let first = items.first {
            selectedVerb = first.infinitive
        }
    }

    func fetchVerbs(language: String) {
        currentLanguage = language
        fetchItems()
    }

    func fetchVerbDetail(language: String, verb: String) {
        apiService.getVerbConjugation(language: language, verb: verb)
            .handleLoadingAndError(on: self)
            .sinkValue(on: self) { [weak self] detail in
                self?.selectedVerbDetail = detail
            }
            .store(in: &cancellables)
    }

    func toggleTense(_ tenseId: String) {
        if expandedTenses.contains(tenseId) {
            expandedTenses.remove(tenseId)
        } else {
            expandedTenses.insert(tenseId)
        }
    }

    func expandAll() {
        if let detail = selectedVerbDetail {
            expandedTenses = Set(detail.tenses.map { $0.tenseId })
        }
    }

    func collapseAll() {
        expandedTenses.removeAll()
    }
}
