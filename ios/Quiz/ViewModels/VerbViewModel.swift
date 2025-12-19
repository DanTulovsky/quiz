import Foundation
import Combine

class VerbViewModel: ObservableObject {
    @Published var verbs: [VerbConjugationSummary] = []
    @Published var selectedVerb: String = ""
    @Published var selectedVerbDetail: VerbConjugationDetail?
    @Published var expandedTenses: Set<String> = []
    @Published var isLoading = false
    @Published var error: APIService.APIError?

    private var currentLanguage: String = "it"
    private var apiService: APIService
    private var cancellables = Set<AnyCancellable>()

    init(apiService: APIService = .shared) {
        self.apiService = apiService

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
        isLoading = true
        apiService.getVerbConjugations(language: language)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isLoading = false
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] data in
                self?.verbs = data.verbs
                if self?.selectedVerb.isEmpty == true, let first = data.verbs.first {
                    self?.selectedVerb = first.infinitive
                }
            })
            .store(in: &cancellables)
    }

    func fetchVerbDetail(language: String, verb: String) {
        isLoading = true
        apiService.getVerbConjugation(language: language, verb: verb)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isLoading = false
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] detail in
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

    func cancelAllRequests() {
        cancellables.removeAll()
    }

    deinit {
        cancelAllRequests()
    }
}
