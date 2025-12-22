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
    let onSnippetSaved: ((Snippet) -> Void)?

    @StateObject var viewModel = TranslationPopupViewModel()
    @StateObject private var ttsManager = TTSSynthesizerManager.shared
    @State var targetLanguage: String = "en"
    @State var isSaving = false
    @State var saveError: String?
    @State var isSaved = false
    @State private var copySuccess: String?

    let maxTranslationLength = 5000
    let maxSnippetLength = 2000

    var body: some View {
        VStack(spacing: 0) {
            ModalHeader(title: "Translation", onClose: onClose)

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
                                }, label: {
                                    Image(systemName: "doc.on.doc")
                                        .foregroundColor(.blue)
                                })
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
                            ForEach(
                                viewModel.availableLanguages.filter {
                                    $0.code.lowercased() != sourceLanguage.lowercased()
                                },
                                id: \.code
                            ) { language in
                                Text(language.name).tag(language.code)
                            }
                        }
                        .pickerStyle(.menu)
                        .onChange(of: targetLanguage) { _, newValue in
                            // Ensure target is different from source
                            if newValue.lowercased() == sourceLanguage.lowercased() {
                                // Reset to a valid language
                                if let alternative = viewModel.availableLanguages.first(
                                    where: { $0.code.lowercased() != sourceLanguage.lowercased() }
                                ) {
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
                                    }, label: {
                                        Image(systemName: "doc.on.doc")
                                            .foregroundColor(.blue)
                                    })
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
                        .buttonStyle(
                            PrimaryButtonStyle(
                                isDisabled: isSaving || isSaved || !canSaveSnippet(translation: translation)
                            )
                        )
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
                        Text(
                            "Text exceeds snippet limit (\(maxSnippetLength) characters). "
                                + "Translation will be saved, but snippet saving may be limited."
                        )
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

}
