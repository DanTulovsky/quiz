import SwiftUI

struct QuizView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel: QuizViewModel
    @State private var reportReason = ""
    @State private var selectedConfidence: Int? = nil
    @State private var selectedText: String?
    @State private var showTranslationPopup = false
    @State private var translationSentence: String?
    @State private var showingSnippet: Snippet? = nil

    @StateObject private var ttsManager = TTSSynthesizerManager.shared

    init(question: Question? = nil, questionType: String? = nil, isDaily: Bool = false) {
        _viewModel = StateObject(
            wrappedValue: QuizViewModel(
                question: question, questionType: questionType, isDaily: isDaily))
    }

    private func stringValue(_ v: JSONValue?) -> String? {
        guard let v else { return nil }
        if case .string(let s) = v { return s }
        return nil
    }

    private func stringArrayValue(_ v: JSONValue?) -> [String]? {
        guard let v else { return nil }
        guard case .array(let arr) = v else { return nil }
        let strings = arr.compactMap { item -> String? in
            guard case .string(let s) = item else { return nil }
            return s
        }
        return strings.isEmpty ? nil : strings
    }

    var body: some View {
        ScrollView {
            ScrollViewReader { proxy in
                VStack(spacing: 20) {
                    if let error = ttsManager.errorMessage {
                        Text(error)
                            .font(.caption)
                            .foregroundColor(.red)
                            .padding()
                            .background(Color.red.opacity(0.1))
                            .cornerRadius(8)
                    }

                    if viewModel.isLoading && viewModel.question == nil {
                        LoadingView()
                            .padding(.top, 50)
                    }

                    if viewModel.error != nil {
                        ErrorDisplay(
                            error: viewModel.error,
                            onDismiss: {
                                viewModel.clearError()
                            })
                    }

                    if let message = viewModel.generatingMessage {
                        VStack {
                            Text(message)
                                .padding()
                            Button("Try Again") {
                                viewModel.getQuestion()
                            }
                            .buttonStyle(.bordered)
                        }
                    } else if let question = viewModel.question {
                        QuestionCardView(
                            question: question,
                            snippets: viewModel.snippets,
                            onTextSelected: { text, fullText in
                                selectedText = text
                                translationSentence = extractSentence(
                                    from: fullText, containing: text)
                                showTranslationPopup = true
                            },
                            onSnippetTapped: { snippet in
                                showingSnippet = snippet
                            }
                        )

                        QuestionOptionsView(
                            question: question,
                            selectedAnswerIndex: viewModel.selectedAnswerIndex,
                            answerResponse: viewModel.answerResponse,
                            showResults: viewModel.answerResponse != nil,
                            onOptionSelected: { index in
                                viewModel.selectedAnswerIndex = index
                            }
                        )

                        if let response = viewModel.answerResponse {
                            AnswerFeedbackView(
                                isCorrect: response.isCorrect,
                                explanation: response.explanation,
                                language: question.language,
                                snippets: viewModel.snippets,
                                onTextSelected: { text, fullText in
                                    selectedText = text
                                    translationSentence = extractSentence(
                                        from: fullText, containing: text)
                                    showTranslationPopup = true
                                },
                                onSnippetTapped: { snippet in
                                    showingSnippet = snippet
                                }
                            )
                        }

                        actionButtons()

                        QuestionActionButtons(
                            isReported: viewModel.isReported,
                            onReport: { viewModel.showReportModal = true },
                            onMarkKnown: { viewModel.showMarkKnownModal = true }
                        )
                    } else {
                        VStack(spacing: 20) {
                            Image(systemName: "questionmark.circle")
                                .scaledFont(size: 60)
                                .foregroundColor(.blue)
                            Text("Ready to test your knowledge?")
                                .font(.title2)
                            Button("Start Quiz") {
                                viewModel.getQuestion()
                            }
                            .buttonStyle(.borderedProminent)
                            .controlSize(.large)
                        }
                        .padding(.top, 100)
                    }
                }
                .padding()
                Color.clear
                    .frame(height: 1)
                    .id("bottom")
                    .onChange(of: viewModel.selectedAnswerIndex) { old, val in
                        if val != nil {
                            withAnimation {
                                proxy.scrollTo("bottom", anchor: .bottom)
                            }
                        }
                    }
                    .onChange(of: viewModel.answerResponse) { old, response in
                        if response != nil {
                            withAnimation {
                                proxy.scrollTo("bottom", anchor: .bottom)
                            }
                        }
                    }
            }
        }
        .navigationBarTitleDisplayMode(.inline)
        .navigationBarBackButtonHidden(true)
        .toolbar {
            ToolbarItem(placement: .navigationBarLeading) {
                Button(action: { dismiss() }) {
                    HStack(spacing: 4) {
                        Image(systemName: "chevron.left")
                            .scaledFont(size: 17, weight: .semibold)
                        Text("Back")
                            .scaledFont(size: 17)
                    }
                    .foregroundColor(.blue)
                }
            }
        }
        .sheet(isPresented: $viewModel.showReportModal) {
            ReportQuestionSheet(
                reportReason: $reportReason,
                isPresented: $viewModel.showReportModal,
                isSubmitting: viewModel.isSubmittingAction
            ) { reason in
                viewModel.reportQuestion(reason: reason.isEmpty ? nil : reason)
            }
        }
        .sheet(isPresented: $viewModel.showMarkKnownModal) {
            MarkKnownSheet(
                selectedConfidence: $selectedConfidence,
                isPresented: $viewModel.showMarkKnownModal,
                isSubmitting: viewModel.isSubmittingAction
            ) { confidence in
                viewModel.markQuestionKnown(confidence: confidence)
            }
        }
        .sheet(isPresented: $showTranslationPopup) {
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
                    onSnippetSaved: {
                        if let questionId = viewModel.question?.id {
                            viewModel.loadSnippets(questionId: questionId)
                        }
                    }
                )
            }
        }
        .snippetDetailPopup(
            showingSnippet: $showingSnippet,
            onSnippetDeleted: { snippet in
                viewModel.snippets.removeAll { $0.id == snippet.id }
            }
        )
        .onChange(of: viewModel.question?.id) { _, questionId in
            if let questionId = questionId {
                viewModel.loadSnippets(questionId: questionId)
            } else {
                viewModel.snippets = []
            }
        }
        .onChange(of: viewModel.question) { _, _ in
            if let questionId = viewModel.question?.id {
                viewModel.loadSnippets(questionId: questionId)
            } else {
                viewModel.snippets = []
            }
        }
        .onAppear {
            if viewModel.question == nil {
                viewModel.getQuestion()
            } else if let questionId = viewModel.question?.id {
                viewModel.loadSnippets(questionId: questionId)
            }
        }
    }

    private func extractSentence(from text: String, containing selectedText: String) -> String? {
        guard let range = text.range(of: selectedText, options: .caseInsensitive) else {
            return nil
        }

        // Find sentence boundaries
        let startIndex = text.startIndex
        let endIndex = text.endIndex

        // Find the start of the sentence (look backwards for sentence-ending punctuation)
        var sentenceStart = range.lowerBound
        let sentenceEnders = CharacterSet(charactersIn: ".!?\n")

        while sentenceStart > startIndex {
            let char = text[sentenceStart]
            if sentenceEnders.contains(char.unicodeScalars.first!) {
                sentenceStart = text.index(after: sentenceStart)
                break
            }
            sentenceStart = text.index(before: sentenceStart)
        }

        // Find the end of the sentence
        var sentenceEnd = range.upperBound
        while sentenceEnd < endIndex {
            let char = text[sentenceEnd]
            if sentenceEnders.contains(char.unicodeScalars.first!) {
                sentenceEnd = text.index(after: sentenceEnd)
                break
            }
            sentenceEnd = text.index(after: sentenceEnd)
        }

        let sentence = String(text[sentenceStart..<sentenceEnd]).trimmingCharacters(
            in: .whitespacesAndNewlines)
        return sentence.isEmpty ? nil : sentence
    }

    @ViewBuilder
    private func actionButtons() -> some View {
        if viewModel.answerResponse != nil {
            Button("Next Question") {
                viewModel.selectedAnswerIndex = nil
                viewModel.getQuestion()
            }
            .buttonStyle(PrimaryButtonStyle())
        } else {
            Button("Submit Answer") {
                if let idx = viewModel.selectedAnswerIndex {
                    viewModel.submitAnswer(userAnswerIndex: idx)
                }
            }
            .buttonStyle(
                PrimaryButtonStyle(
                    isDisabled: viewModel.selectedAnswerIndex == nil || viewModel.isLoading))
        }
    }

}
