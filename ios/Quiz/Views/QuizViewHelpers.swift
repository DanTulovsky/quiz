import SwiftUI

extension QuizView {
    func stringValue(_ value: JSONValue?) -> String? {
        guard let value else { return nil }
        if case .string(let stringValue) = value { return stringValue }
        return nil
    }

    func stringArrayValue(_ value: JSONValue?) -> [String]? {
        guard let value else { return nil }
        guard case .array(let arr) = value else { return nil }
        let strings = arr.compactMap { item -> String? in
            guard case .string(let stringValue) = item else { return nil }
            return stringValue
        }
        return strings.isEmpty ? nil : strings
    }

    func extractSentence(from text: String, containing selectedText: String) -> String? {
        return TextUtils.extractSentence(from: text, containing: selectedText)
    }

    @ViewBuilder
    var translationSheetContent: some View {
        if let text = selectedText, let question = viewModel.question {
            TranslationPopupView(
                selectedText: text,
                sourceLanguage: question.language,
                questionId: question.id,
                sectionId: nil,
                storyId: nil,
                sentence: translationSentence,
                onClose: {
                    showTranslationPopup = false
                    selectedText = nil
                    translationSentence = nil
                },
                onSnippetSaved: { snippet in
                    if !viewModel.snippets.contains(where: { $0.id == snippet.id }) {
                        viewModel.snippets += [snippet]
                        snippetRefreshTrigger += 1
                    }
                    if let questionId = viewModel.question?.id {
                        viewModel.resetSnippetCache()
                        viewModel.loadSnippets(questionId: questionId)
                    }
                }
            )
        }
    }
}
