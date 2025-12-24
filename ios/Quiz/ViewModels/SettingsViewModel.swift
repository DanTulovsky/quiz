import Combine
import Foundation

class SettingsViewModel: BaseViewModel, LanguageCaching, ListFetchingWithName, LanguageFetching,
                         SuccessStateManaging, LevelFetching {
    typealias Item = AIProviderInfo

    @Published var aiProviders: [AIProviderInfo] = []

    var items: [AIProviderInfo] {
        get { aiProviders }
        set { aiProviders = newValue }
    }
    @Published var availableVoices: [EdgeTTSVoiceInfo] = []
    @Published var availableLevels: [String] = []
    @Published var levelDescriptions: [String: String] = [:]
    @Published var availableLanguages: [LanguageInfo] = [] {
        didSet {
            DispatchQueue.main.async { [weak self] in
                self?.updateLanguageCache()
            }
        }
    }

    var languageCacheByCode: [String: LanguageInfo] = [:]
    var languageCacheByName: [String: LanguageInfo] = [:]

    @Published var testResult: String?
    @Published var testNotificationResults: [String: String] = [:]

    @Published var user: User?
    @Published var learningPrefs: UserLearningPreferences?
    @Published var isSuccess = false

    override init(apiService: APIServiceProtocol = APIService.shared) {
        super.init(apiService: apiService)
    }

    func fetchSettings() {
        resetSuccessState()

        // Fetch learning preferences
        apiService.getLearningPreferences()
            .handleLoadingAndError(on: self)
            .sinkValue(on: self) { [weak self] prefs in
                self?.learningPrefs = prefs
            }
            .store(in: &cancellables)
    }

    func saveChanges(userUpdate: UserUpdateRequest, prefs: UserLearningPreferences?) {
        resetSuccessState()

        let userPublisher = apiService.updateUser(request: userUpdate)

        if let prefs = prefs {
            let prefsPublisher = apiService.updateLearningPreferences(prefs: prefs)

            Publishers.Zip(userPublisher, prefsPublisher)
                .handleLoadingAndError(on: self)
                .sinkValue(on: self) { [weak self] user, prefs in
                    self?.user = user
                    self?.learningPrefs = prefs
                    self?.setSuccessState()
                }
                .store(in: &cancellables)
        } else {
            userPublisher
                .handleLoadingAndError(on: self)
                .sinkValue(on: self) { [weak self] user in
                    self?.user = user
                    self?.setSuccessState()
                }
                .store(in: &cancellables)
        }
    }

    func fetchItemsPublisher() -> AnyPublisher<[AIProviderInfo], APIService.APIError> {
        return apiService.getAIProviders()
            .map { $0.providers }
            .eraseToAnyPublisher()
    }

    func updateItems(_ items: [AIProviderInfo]) {
        aiProviders = items
    }

    func fetchAIProviders() {
        fetchItems()
    }

    func testAI(provider: String, model: String, apiKey: String?) {
        testResult = nil
        apiService.testAIConnection(provider: provider, model: model, apiKey: apiKey)
            .handleErrorOnly(on: self)
            .sink(
                receiveCompletion: { [weak self] completion in
                    if case .failure(let error) = completion {
                        self?.testResult = "Error: \(error.localizedDescription)"
                    }
                },
                receiveValue: { [weak self] resp in
                    if let message = resp.message {
                        self?.testResult = "Success: \(message)"
                    } else {
                        self?.testResult = "Success"
                    }
                }
            )
            .store(in: &cancellables)
    }

    func sendTestEmail() {
        executeVoidWithSuccessState(publisher: apiService.sendTestEmail())
            .store(in: &cancellables)
    }

    func sendTestIOSNotification(notificationType: String) {
        testNotificationResults[notificationType] = nil
        apiService.sendTestIOSNotification(notificationType: notificationType)
            .handleErrorOnly(on: self)
            .sink(
                receiveCompletion: { [weak self] completion in
                    if case .failure(let error) = completion {
                        self?.testNotificationResults[notificationType] =
                            "Error: \(error.localizedDescription)"
                    }
                },
                receiveValue: { [weak self] resp in
                    if let message = resp.message {
                        self?.testNotificationResults[notificationType] = "Success: \(message)"
                    } else {
                        self?.testNotificationResults[notificationType] =
                            "Success: Test notification sent"
                    }
                }
            )
            .store(in: &cancellables)
    }

    func clearStories() {
        executeVoidWithSuccessState(publisher: apiService.clearStories())
            .store(in: &cancellables)
    }

    func clearAIChats() {
        executeVoidWithSuccessState(publisher: apiService.clearAIChats())
            .store(in: &cancellables)
    }

    func clearTranslationHistory() {
        executeVoidWithSuccessState(publisher: apiService.clearTranslationHistory())
            .store(in: &cancellables)
    }

    func resetAccount() {
        executeVoidWithSuccessState(publisher: apiService.resetAccount())
            .store(in: &cancellables)
    }

    private func mapLanguageToLocale(_ lang: String) -> String? {
        let lowercased = lang.lowercased()
        return languageCacheByCode[lowercased]?.ttsLocale
            ?? languageCacheByName[lowercased]?.ttsLocale
    }

    func fetchVoices(language: String) {
        guard let locale = mapLanguageToLocale(language) else {
            // Languages not loaded yet, skip fetching voices
            return
        }
        apiService.getVoices(language: locale)
            .handleErrorOnly(on: self)
            .sinkValue(on: self) { [weak self] voices in
                self?.availableVoices = voices
            }
            .store(in: &cancellables)
    }

    func getDefaultVoiceIdentifier(for language: String) -> String? {
        // First try to get the default voice from LanguageInfo
        if let defaultVoice = getDefaultVoice(for: language) {
            return defaultVoice
        }
        // If no default in LanguageInfo, return the first voice in the list
        return availableVoices.first?.shortName ?? availableVoices.first?.name
    }

    func getDefaultVoice(for language: String) -> String? {
        let lowercased = language.lowercased()
        return languageCacheByCode[lowercased]?.ttsVoice
            ?? languageCacheByName[lowercased]?.ttsVoice
    }
}
