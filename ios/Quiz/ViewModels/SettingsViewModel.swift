import Combine
import Foundation

class SettingsViewModel: BaseViewModel {
    @Published var aiProviders: [AIProviderInfo] = []
    @Published var availableVoices: [EdgeTTSVoiceInfo] = []
    @Published var availableLevels: [String] = []
    @Published var levelDescriptions: [String: String] = [:]
    @Published var availableLanguages: [LanguageInfo] = [] {
        didSet {
            updateLanguageCache()
        }
    }

    private var languageCacheByCode: [String: LanguageInfo] = [:]
    private var languageCacheByName: [String: LanguageInfo] = [:]

    @Published var testResult: String?

    @Published var user: User?
    @Published var learningPrefs: UserLearningPreferences?
    @Published var isSuccess = false

    private func updateLanguageCache() {
        languageCacheByCode.removeAll()
        languageCacheByName.removeAll()
        for lang in availableLanguages {
            languageCacheByCode[lang.code.lowercased()] = lang
            languageCacheByName[lang.name.lowercased()] = lang
        }
    }

    init(apiService: APIService = APIService.shared) {
        super.init(apiService: apiService)
    }

    func fetchSettings() {
        isLoading = true
        isSuccess = false

        // Fetch learning preferences
        apiService.getLearningPreferences()
            .handleLoadingAndError(on: self)
            .sink(receiveValue: { [weak self] prefs in
                self?.learningPrefs = prefs
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
                .sink(
                    receiveCompletion: { [weak self] completion in
                        self?.isLoading = false
                        if case .failure(let error) = completion {
                            self?.error = error
                        }
                    },
                    receiveValue: { [weak self] user, prefs in
                        self?.user = user
                        self?.learningPrefs = prefs
                        self?.isSuccess = true
                    }
                )
                .store(in: &cancellables)
        } else {
            userPublisher
                .receive(on: DispatchQueue.main)
                .sink(
                    receiveCompletion: { [weak self] completion in
                        self?.isLoading = false
                        if case .failure(let error) = completion {
                            self?.error = error
                        }
                    },
                    receiveValue: { [weak self] user in
                        self?.user = user
                        self?.isSuccess = true
                    }
                )
                .store(in: &cancellables)
        }
    }

    func fetchAIProviders() {
        apiService.getAIProviders()
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { _ in },
                receiveValue: { [weak self] resp in
                    self?.aiProviders = resp.providers
                }
            )
            .store(in: &cancellables)
    }

    func fetchLanguages() {
        apiService.getLanguages()
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { _ in },
                receiveValue: { [weak self] languages in
                    self?.availableLanguages = languages
                }
            )
            .store(in: &cancellables)
    }

    func fetchLevels(language: String?) {
        apiService.getLevels(language: language)
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { _ in },
                receiveValue: { [weak self] response in
                    self?.availableLevels = response.levels
                    self?.levelDescriptions = response.levelDescriptions
                }
            )
            .store(in: &cancellables)
    }

    func testAI(provider: String, model: String, apiKey: String?) {
        testResult = nil
        apiService.testAIConnection(provider: provider, model: model, apiKey: apiKey)
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { [weak self] completion in
                    if case .failure(let error) = completion {
                        self?.testResult = "Error: \(error.localizedDescription)"
                    }
                },
                receiveValue: { [weak self] resp in
                    self?.testResult = "Success: \(resp.message)"
                }
            )
            .store(in: &cancellables)
    }

    func sendTestEmail() {
        apiService.sendTestEmail()
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { [weak self] completion in
                    if case .failure(let error) = completion {
                        self?.error = error
                    }
                },
                receiveValue: { [weak self] _ in
                    self?.isSuccess = true
                }
            )
            .store(in: &cancellables)
    }

    func clearStories() {
        apiService.clearStories()
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { [weak self] completion in
                    if case .failure(let error) = completion { self?.error = error }
                }, receiveValue: { [weak self] _ in self?.isSuccess = true }
            )
            .store(in: &cancellables)
    }

    func clearAIChats() {
        apiService.clearAIChats()
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { [weak self] completion in
                    if case .failure(let error) = completion { self?.error = error }
                }, receiveValue: { [weak self] _ in self?.isSuccess = true }
            )
            .store(in: &cancellables)
    }

    func clearTranslationHistory() {
        apiService.clearTranslationHistory()
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { [weak self] completion in
                    if case .failure(let error) = completion { self?.error = error }
                }, receiveValue: { [weak self] _ in self?.isSuccess = true }
            )
            .store(in: &cancellables)
    }

    func resetAccount() {
        apiService.resetAccount()
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { [weak self] completion in
                    if case .failure(let error) = completion { self?.error = error }
                }, receiveValue: { [weak self] _ in self?.isSuccess = true }
            )
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
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { _ in },
                receiveValue: { [weak self] voices in
                    self?.availableVoices = voices
                }
            )
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
