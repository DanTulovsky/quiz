import Foundation
import Combine

class SettingsViewModel: ObservableObject {
    @Published var aiProviders: [AIProviderInfo] = []
    @Published var availableVoices: [EdgeTTSVoiceInfo] = []
    @Published var availableLevels: [String] = []

    @Published var testResult: String?

    @Published var user: User?
    @Published var learningPrefs: UserLearningPreferences?
    @Published var error: APIService.APIError?
    @Published var isLoading = false
    @Published var isSuccess = false

    var apiService: APIService
    private var cancellables = Set<AnyCancellable>()

    init(apiService: APIService = APIService.shared) {
        self.apiService = apiService
    }

    func fetchSettings() {
        isLoading = true
        isSuccess = false

        // Fetch learning preferences
        apiService.getLearningPreferences()
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                if case .failure(let error) = completion {
                    self?.error = error
                    self?.isLoading = false
                }
            }, receiveValue: { [weak self] prefs in
                self?.learningPrefs = prefs
                self?.isLoading = false
            })
            .store(in: &cancellables)
    }

    func saveChanges(userUpdate: UserUpdateRequest, prefs: UserLearningPreferences?) {
        isLoading = true
        isSuccess = false
        error = nil

        let userPublisher = apiService.updateUser(request: userUpdate)

        if let prefs = prefs {
            let prefsPublisher = apiService.updateLearningPreferences(prefs: prefs)

            Publishers.Zip(userPublisher, prefsPublisher)
                .receive(on: DispatchQueue.main)
                .sink(receiveCompletion: { [weak self] completion in
                    self?.isLoading = false
                    if case .failure(let error) = completion {
                        self?.error = error
                    }
                }, receiveValue: { [weak self] user, prefs in
                    self?.user = user
                    self?.learningPrefs = prefs
                    self?.isSuccess = true
                })
                .store(in: &cancellables)
        } else {
            userPublisher
                .receive(on: DispatchQueue.main)
                .sink(receiveCompletion: { [weak self] completion in
                    self?.isLoading = false
                    if case .failure(let error) = completion {
                        self?.error = error
                    }
                }, receiveValue: { [weak self] user in
                    self?.user = user
                    self?.isSuccess = true
                })
                .store(in: &cancellables)
        }
    }

    func fetchAIProviders() {
        apiService.getAIProviders()
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] resp in
                self?.aiProviders = resp.providers
                self?.availableLevels = resp.levels
            })
            .store(in: &cancellables)
    }

    func testAI(provider: String, model: String, apiKey: String?) {
        testResult = nil
        apiService.testAIConnection(provider: provider, model: model, apiKey: apiKey)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                if case .failure(let error) = completion {
                    self?.testResult = "Error: \(error.localizedDescription)"
                }
            }, receiveValue: { [weak self] resp in
                self?.testResult = "Success: \(resp.message)"
            })
            .store(in: &cancellables)
    }

    func sendTestEmail() {
        apiService.sendTestEmail()
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                if case .failure(let error) = completion {
                    self?.error = error
                }
            }, receiveValue: { [weak self] _ in
                self?.isSuccess = true
            })
            .store(in: &cancellables)
    }

    func clearStories() {
        apiService.clearStories()
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] _ in self?.isSuccess = true })
            .store(in: &cancellables)
    }

    func clearAIChats() {
        apiService.clearAIChats()
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] _ in self?.isSuccess = true })
            .store(in: &cancellables)
    }

    func clearTranslationHistory() {
        apiService.clearTranslationHistory()
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] _ in self?.isSuccess = true })
            .store(in: &cancellables)
    }

    func resetAccount() {
        apiService.resetAccount()
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] _ in self?.isSuccess = true })
            .store(in: &cancellables)
    }



        private func mapLanguageToLocale(_ lang: String) -> String {
        switch lang.lowercased() {
        case "italian", "it": return "it-IT"
        case "spanish", "es": return "es-ES"
        case "french", "fr": return "fr-FR"
        case "german", "de": return "de-DE"
        case "english", "en": return "en-US"
        case "hindi": return "hi-IN"
        case "russian": return "ru-RU"
        case "japanese": return "ja-JP"
        case "chinese": return "zh-CN"
        case "portuguese": return "pt-PT"
        case "korean": return "ko-KR"
        default: return lang
        }
    }

    func fetchVoices(language: String) {
        let locale = mapLanguageToLocale(language)
        apiService.getVoices(language: locale)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] voices in
                self?.availableVoices = voices
            })
            .store(in: &cancellables)
    }

}
