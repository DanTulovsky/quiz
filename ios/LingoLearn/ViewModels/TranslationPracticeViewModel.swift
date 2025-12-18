import Foundation
import Combine

class TranslationPracticeViewModel: ObservableObject {
    @Published var currentSentence: TranslationPracticeSentenceResponse?
    @Published var feedback: TranslationPracticeSessionResponse?
    @Published var history: [TranslationPracticeSessionResponse] = []
    @Published var totalHistoryCount = 0
    @Published var isLoading = false
    @Published var error: APIService.APIError?
    
    @Published var userTranslation = ""
    @Published var optionalTopic = ""
    @Published var selectedDirection = "learning_to_en"
    
    private var apiService: APIService
    private var cancellables = Set<AnyCancellable>()
    
    init(apiService: APIService = .shared) {
        self.apiService = apiService
    }
    
    func fetchHistory() {
        apiService.getTranslationPracticeHistory()
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] response in
                self?.history = response.sessions
                self?.totalHistoryCount = response.total
            })
            .store(in: &cancellables)
    }

    func fetchExistingSentence(language: String, level: String) {
        isLoading = true
        error = nil
        feedback = nil
        userTranslation = ""
        
        apiService.getExistingTranslationSentence(language: language, level: level, direction: selectedDirection)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isLoading = false
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] response in
                self?.currentSentence = response
            })
            .store(in: &cancellables)
    }

    func generateSentence(language: String, level: String) {
        isLoading = true
        error = nil
        feedback = nil
        userTranslation = ""
        
        let topic = optionalTopic.isEmpty ? nil : optionalTopic
        let request = TranslationPracticeGenerateRequest(language: language, level: level, direction: selectedDirection, topic: topic)
        apiService.generateTranslationSentence(request: request)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isLoading = false
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] response in
                self?.currentSentence = response
            })
            .store(in: &cancellables)
    }
    
    func submitTranslation() {

        guard let sentence = currentSentence else { return }
        isLoading = true
        error = nil
        
        let request = TranslationPracticeSubmitRequest(
            sentenceId: sentence.id,
            originalSentence: sentence.sentenceText,
            userTranslation: userTranslation,
            translationDirection: selectedDirection
        )
        
        apiService.submitTranslation(request: request)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isLoading = false
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] response in
                self?.feedback = response
                self?.userTranslation = "" // Clear after success
                self?.fetchHistory() // Refresh history after submission
            })
            .store(in: &cancellables)
    }
}
