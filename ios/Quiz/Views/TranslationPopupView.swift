import SwiftUI
import Combine

struct TranslationPopupView: View {
    let selectedText: String
    let sourceLanguage: String
    let questionId: Int?
    let sectionId: Int?
    let storyId: Int?
    let sentence: String?
    let onClose: () -> Void
    let onSnippetSaved: (() -> Void)?

    @StateObject private var viewModel = TranslationPopupViewModel()
    @StateObject private var ttsManager = TTSSynthesizerManager.shared
    @State private var targetLanguage: String = "en"
    @State private var isSaving = false
    @State private var saveError: String?
    @State private var isSaved = false
    @State private var copySuccess: String?

    private let maxTranslationLength = 5000
    private let maxSnippetLength = 2000

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text("Translation")
                    .font(AppTheme.Typography.headingFont)
                Spacer()
                Button(action: onClose) {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundColor(.secondary)
                        .font(.title2)
                }
            }
            .padding()

            Divider()

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    // Selected text section
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            Text(selectedText)
                                .font(AppTheme.Typography.bodyFont)
                                .textSelection(.enabled)
                            Spacer()
                            HStack(spacing: 12) {
                                TTSButton(text: selectedText, language: sourceLanguage)
                                Button(action: {
                                    UIPasteboard.general.string = selectedText
                                    copySuccess = "Original text copied"
                                    DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
                                        copySuccess = nil
                                    }
                                }) {
                                    Image(systemName: "doc.on.doc")
                                        .foregroundColor(.blue)
                                }
                            }
                        }
                    }
                    .padding()
                    .background(AppTheme.Colors.secondaryBackground)
                    .cornerRadius(AppTheme.CornerRadius.innerCard)

                    // Language selector
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Translate to")
                            .font(AppTheme.Typography.subheadlineFont)
                            .foregroundColor(AppTheme.Colors.secondaryText)
                        Picker("Target Language", selection: $targetLanguage) {
                            ForEach(viewModel.availableLanguages.filter { $0.code.lowercased() != sourceLanguage.lowercased() }, id: \.code) { language in
                                Text(language.name).tag(language.code)
                            }
                        }
                        .pickerStyle(.menu)
                        .onChange(of: targetLanguage) { _, newValue in
                            // Ensure target is different from source
                            if newValue.lowercased() == sourceLanguage.lowercased() {
                                // Reset to a valid language
                                if let alternative = viewModel.availableLanguages.first(where: { $0.code.lowercased() != sourceLanguage.lowercased() }) {
                                    targetLanguage = alternative.code
                                }
                            } else {
                                translateText()
                            }
                        }
                    }
                    .padding()
                    .background(AppTheme.Colors.secondaryBackground)
                    .cornerRadius(AppTheme.CornerRadius.innerCard)

                    // Translation section
                    if viewModel.isLoading {
                        HStack {
                            ProgressView()
                            Text("Translating...")
                                .font(AppTheme.Typography.subheadlineFont)
                                .foregroundColor(AppTheme.Colors.secondaryText)
                        }
                        .frame(maxWidth: .infinity)
                        .padding()
                    } else if let error = viewModel.error {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Translation Error")
                                .font(AppTheme.Typography.subheadlineFont)
                                .foregroundColor(AppTheme.Colors.errorRed)
                            Text(error)
                                .font(AppTheme.Typography.captionFont)
                                .foregroundColor(AppTheme.Colors.secondaryText)
                        }
                        .padding()
                        .background(AppTheme.Colors.errorRed.opacity(0.1))
                        .cornerRadius(AppTheme.CornerRadius.innerCard)
                    } else if let translation = viewModel.translation {
                        VStack(alignment: .leading, spacing: 8) {
                            HStack {
                                Text(translation.translatedText)
                                    .font(AppTheme.Typography.bodyFont)
                                    .textSelection(.enabled)
                                Spacer()
                                HStack(spacing: 12) {
                                    TTSButton(
                                        text: translation.translatedText,
                                        language: targetLanguage,
                                        voiceIdentifier: defaultVoiceForTargetLanguage()
                                    )
                                    Button(action: {
                                        UIPasteboard.general.string = translation.translatedText
                                        copySuccess = "Translation copied"
                                        DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
                                            copySuccess = nil
                                        }
                                    }) {
                                        Image(systemName: "doc.on.doc")
                                            .foregroundColor(.blue)
                                    }
                                }
                            }
                        }
                        .padding()
                        .background(AppTheme.Colors.secondaryBackground)
                        .cornerRadius(AppTheme.CornerRadius.innerCard)
                    }

                    // Copy success message
                    if let message = copySuccess {
                        Text(message)
                            .font(AppTheme.Typography.captionFont)
                            .foregroundColor(AppTheme.Colors.successGreen)
                            .padding(.horizontal)
                    }

                    // Save snippet button
                    if let translation = viewModel.translation {
                        Button(action: saveSnippet) {
                            HStack {
                                if isSaving {
                                    ProgressView()
                                        .progressViewStyle(CircularProgressViewStyle(tint: .white))
                                } else if isSaved {
                                    Image(systemName: "checkmark.circle.fill")
                                } else {
                                    Image(systemName: "bookmark")
                                }
                                Text(isSaved ? "Saved" : "Save as Snippet")
                                    .font(AppTheme.Typography.buttonFont)
                            }
                        }
                        .buttonStyle(PrimaryButtonStyle(isDisabled: isSaving || isSaved || !canSaveSnippet(translation: translation)))
                        .padding(.top, 8)
                    }

                    // Save error
                    if let error = saveError {
                        Text(error)
                            .font(AppTheme.Typography.captionFont)
                            .foregroundColor(AppTheme.Colors.errorRed)
                            .padding(.horizontal)
                    }

                    // Text length warnings
                    if selectedText.count > maxTranslationLength {
                        Text("Text exceeds translation limit (\(maxTranslationLength) characters)")
                            .font(AppTheme.Typography.captionFont)
                            .foregroundColor(AppTheme.Colors.errorRed)
                            .padding(.horizontal)
                    }
                    if selectedText.count > maxSnippetLength {
                        Text("Text exceeds snippet limit (\(maxSnippetLength) characters). Translation will be saved, but snippet saving may be limited.")
                            .font(AppTheme.Typography.captionFont)
                            .foregroundColor(AppTheme.Colors.errorRed)
                            .padding(.horizontal)
                    }
                }
                .padding()
            }
        }
        .frame(maxWidth: .infinity, maxHeight: 600)
        .background(AppTheme.Colors.cardBackground)
        .cornerRadius(AppTheme.CornerRadius.card)
        .shadow(radius: 20)
        .padding()
        .onAppear {
            viewModel.loadLanguages()
            // Set default target language - ensure it's different from source
            if let userLang = viewModel.user?.preferredLanguage {
                // If source language is the user's preferred language, default to English
                // Otherwise, use the user's preferred language as target
                if sourceLanguage.lowercased() == userLang.lowercased() {
                    targetLanguage = "en"
                } else {
                    targetLanguage = userLang
                }
            } else {
                // Default to English if no user preference, or if source is already English
                if sourceLanguage.lowercased() == "en" {
                    // If source is English, pick first available language that's not English
                    // This will be set after languages load
                    targetLanguage = "en" // Temporary, will be updated
                } else {
                    targetLanguage = "en"
                }
            }
            translateText()
        }
        .onChange(of: viewModel.availableLanguages) { _, languages in
            // Once languages are loaded, ensure target is different from source
            if !languages.isEmpty && targetLanguage.lowercased() == sourceLanguage.lowercased() {
                // Find first language that's not the source
                if let alternative = languages.first(where: { $0.code.lowercased() != sourceLanguage.lowercased() }) {
                    targetLanguage = alternative.code
                } else {
                    // Fallback to 'en' if source is not English, otherwise use first available
                    targetLanguage = sourceLanguage.lowercased() == "en" ? (languages.first?.code ?? "en") : "en"
                }
            }
        }
        .onChange(of: targetLanguage) { _, _ in
            translateText()
        }
    }

    private func translateText() {
        // Validate text length
        guard selectedText.count <= maxTranslationLength else {
            viewModel.error = "Text exceeds maximum translation length of \(maxTranslationLength) characters"
            return
        }

        guard selectedText.count > 1 else {
            viewModel.error = "Text must be more than 1 character"
            return
        }

        // Ensure source and target are different
        guard sourceLanguage.lowercased() != targetLanguage.lowercased() else {
            viewModel.error = "Source and target languages must be different"
            return
        }

        viewModel.translate(
            text: selectedText,
            sourceLanguage: sourceLanguage,
            targetLanguage: targetLanguage
        )
    }

    private func defaultVoiceForTargetLanguage() -> String? {
        // Find the default voice for the target language from available languages
        if let languageInfo = viewModel.availableLanguages.find(byCodeOrName: targetLanguage),
           let defaultVoice = languageInfo.ttsVoice {
            return defaultVoice
        }
        // Fallback to using TTSSynthesizerManager's default voice method
        return TTSSynthesizerManager.shared.defaultVoiceForLanguage(targetLanguage)
    }

    private func canSaveSnippet(translation: TranslateResponse) -> Bool {
        return selectedText.count <= maxSnippetLength && translation.translatedText.count <= maxSnippetLength
    }

    private func saveSnippet() {
        guard let translation = viewModel.translation else { return }

        // Validate lengths
        guard selectedText.count <= maxSnippetLength else {
            saveError = "Original text exceeds snippet limit of \(maxSnippetLength) characters"
            return
        }

        guard translation.translatedText.count <= maxSnippetLength else {
            saveError = "Translated text exceeds snippet limit of \(maxSnippetLength) characters"
            return
        }

        isSaving = true
        saveError = nil

        let request = CreateSnippetRequest(
            originalText: selectedText,
            translatedText: translation.translatedText,
            sourceLanguage: translation.sourceLanguage,
            targetLanguage: translation.targetLanguage,
            context: sentence,
            questionId: questionId,
            sectionId: sectionId,
            storyId: storyId
        )

        APIService.shared.createSnippet(request: request)
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { completion in
                    isSaving = false
                    if case .failure(let error) = completion {
                        saveError = error.localizedDescription
                    }
                },
                receiveValue: { _ in
                    isSaving = false
                    isSaved = true
                    saveError = nil

                    onSnippetSaved?()

                    DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
                        onClose()
                    }
                }
            )
            .store(in: &viewModel.cancellables)
    }
}

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

