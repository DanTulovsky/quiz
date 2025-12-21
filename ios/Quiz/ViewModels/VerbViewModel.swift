import Foundation
import Combine

class VerbViewModel: BaseViewModel {
    @Published var verbs: [VerbConjugationSummary] = []
    @Published var selectedVerb: String = ""
    @Published var selectedVerbDetail: VerbConjugationDetail?
    @Published var expandedTenses: Set<String> = []

    private var currentLanguage: String = "it"

    init(apiService: APIService = .shared) {
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

    func fetchVerbs(language: String) {
        self.currentLanguage = language
        apiService.getVerbConjugations(language: language)
            .handleLoadingAndError(on: self)
            .sink(receiveValue: { [weak self] data in
                self?.verbs = data.verbs
                if self?.selectedVerb.isEmpty == true, let first = data.verbs.first {
                    self?.selectedVerb = first.infinitive
                }
            })
            .store(in: &cancellables)
    }

    func fetchVerbDetail(language: String, verb: String) {
        apiService.getVerbConjugation(language: language, verb: verb)
            .handleLoadingAndError(on: self)
            .sink(receiveValue: { [weak self] detail in
                self?.selectedVerbDetail = detail
            })
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
