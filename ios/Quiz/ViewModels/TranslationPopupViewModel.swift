import Foundation
import Combine
import SwiftUI

class TranslationPopupViewModel: ObservableObject {
    @Published var translation: TranslateResponse?
    @Published var isLoading = false
    @Published var error: String?
    @Published var availableLanguages: [LanguageInfo] = [] {
        didSet {
            updateLanguageCache()
        }
    }
    @Published var user: User?

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

    private var apiService = APIService.shared
    var cancellables = Set<AnyCancellable>()

    init() {
        loadUser()
    }

    func loadUser() {
        apiService.authStatus()
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { _ in },
                receiveValue: { [weak self] response in
                    if response.authenticated {
                        self?.user = response.user
                    }
                }
            )
            .store(in: &cancellables)
    }

    func loadLanguages() {
        apiService.getLanguages()
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { completion in
                    if case .failure(let error) = completion {
                        print("Failed to load languages: \(error)")
                    }
                },
                receiveValue: { [weak self] languages in
                    self?.availableLanguages = languages
                }
            )
            .store(in: &cancellables)
    }

    func translate(text: String, sourceLanguage: String, targetLanguage: String) {
        isLoading = true
        error = nil

        let request = TranslateRequest(
            text: text,
            targetLanguage: targetLanguage,
            sourceLanguage: sourceLanguage
        )

        apiService.translateText(request: request)
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { [weak self] completion in
                    self?.isLoading = false
                    if case .failure(let error) = completion {
                        self?.error = error.localizedDescription
                    }
                },
                receiveValue: { [weak self] response in
                    self?.isLoading = false
                    self?.translation = response
                    self?.error = nil
                }
            )
            .store(in: &cancellables)
    }
}





