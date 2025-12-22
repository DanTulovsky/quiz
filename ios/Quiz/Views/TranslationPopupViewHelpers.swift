import SwiftUI
import Combine

extension TranslationPopupView {
    func translateText() {
        guard selectedText.count <= maxTranslationLength else {
            viewModel.error = "Text exceeds maximum translation length of \(maxTranslationLength) characters"
            return
        }

        guard selectedText.count > 1 else {
            viewModel.error = "Text must be more than 1 character"
            return
        }

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

    func defaultVoiceForTargetLanguage() -> String? {
        if let languageInfo = viewModel.availableLanguages.find(byCodeOrName: targetLanguage),
           let defaultVoice = languageInfo.ttsVoice {
            return defaultVoice
        }
        return TTSSynthesizerManager.shared.defaultVoiceForLanguage(targetLanguage)
    }

    func canSaveSnippet(translation: TranslateResponse) -> Bool {
        return selectedText.count <= maxSnippetLength && translation.translatedText.count <= maxSnippetLength
    }

    func saveSnippet() {
        guard let translation = viewModel.translation else { return }

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
                receiveValue: { snippet in
                    isSaving = false
                    isSaved = true
                    saveError = nil

                    onSnippetSaved?(snippet)

                    DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
                        onClose()
                    }
                }
            )
            .store(in: &viewModel.cancellables)
    }
}
