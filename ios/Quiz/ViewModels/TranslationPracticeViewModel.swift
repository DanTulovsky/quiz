import Combine
import Foundation

class TranslationPracticeViewModel: BaseViewModel {
    @Published var currentSentence: TranslationPracticeSentenceResponse?
    @Published var feedback: TranslationPracticeSessionResponse?
    @Published var history: [TranslationPracticeSessionResponse] = []
    @Published var totalHistoryCount = 0

    @Published var userTranslation = ""
    @Published var optionalTopic = ""
    @Published var selectedDirection = "learning_to_en"

    override init(apiService: APIService = .shared) {
        super.init(apiService: apiService)
    }

    func fetchHistory() {
        apiService.getTranslationPracticeHistory()
            .handleErrorOnly(on: self)
            .sinkValue(on: self) { [weak self] response in
                self?.history = response.sessions
                self?.totalHistoryCount = response.total
            }
            .store(in: &cancellables)
    }

    func fetchExistingSentence(language: String, level: String) {
        clearError()
        feedback = nil
        userTranslation = ""

        apiService.getExistingTranslationSentence(
            language: language, level: level, direction: selectedDirection
        )
        .handleLoadingAndError(on: self)
        .sinkValue(on: self) { [weak self] response in
            self?.currentSentence = response
        }
        .store(in: &cancellables)
    }

    func generateSentence(language: String, level: String) {
        clearError()
        feedback = nil
        userTranslation = ""
        currentSentence = nil

        let topic = optionalTopic.isEmpty ? nil : optionalTopic
        let request = TranslationPracticeGenerateRequest(
            language: language, level: level, direction: selectedDirection, topic: topic)
        apiService.generateTranslationSentence(request: request)
            .handleLoadingAndError(on: self)
            .sinkValue(on: self) { [weak self] response in
                self?.currentSentence = response
            }
            .store(in: &cancellables)
    }

    func submitTranslation() {
        guard let sentence = currentSentence else { return }
        clearError()

        let request = TranslationPracticeSubmitRequest(
            sentenceId: sentence.id,
            originalSentence: sentence.sentenceText,
            userTranslation: userTranslation,
            translationDirection: selectedDirection
        )

        apiService.submitTranslation(request: request)
            .handleLoadingAndError(on: self)
            .sinkValue(on: self) { [weak self] response in
                self?.feedback = response
                self?.fetchHistory()  // Refresh history after submission
            }
            .store(in: &cancellables)
    }
}
