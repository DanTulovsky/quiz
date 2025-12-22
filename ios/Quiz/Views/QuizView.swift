import SwiftUI

struct QuizView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject var viewModel: QuizViewModel
    @State private var reportReason = ""
    @State private var selectedConfidence: Int?
    @State var selectedText: String?
    @State var showTranslationPopup = false
    @State var translationSentence: String?
    @State private var showingSnippet: Snippet?
    @State var snippetRefreshTrigger: Int = 0

    @StateObject private var ttsManager = TTSSynthesizerManager.shared

    init(question: Question? = nil, questionType: String? = nil, isDaily: Bool = false) {
        _viewModel = StateObject(
            wrappedValue: QuizViewModel(
                question: question, questionType: questionType, isDaily: isDaily))
    }

    private var questionCardId: String {
        guard let question = viewModel.question else { return "" }
        let snippetIds = viewModel.snippets.map { "\($0.id)" }.joined(separator: ",")
        return
            "question-\(question.id)-snippets-\(viewModel.snippets.count)-\(snippetIds)-\(snippetRefreshTrigger)"
    }

    var body: some View {
        mainContent
            .navigationBarTitleDisplayMode(.inline)
            .navigationBarBackButtonHidden(true)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button(
                        action: { dismiss() },
                        label: {
                            HStack(spacing: 4) {
                                Image(systemName: "chevron.left")
                                    .scaledFont(size: 17, weight: .semibold)
                                Text("Back")
                                    .scaledFont(size: 17)
                            }
                            .foregroundColor(.blue)
                        })
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
                translationSheetContent
            }
            .snippetDetailPopup(
                showingSnippet: $showingSnippet,
                onSnippetDeleted: { snippet in
                    viewModel.snippets.removeAll { $0.id == snippet.id }
                }
            )
            .onChange(of: viewModel.question?.id) { _, questionId in
                if questionId == nil {
                    viewModel.snippets = []
                } else if let questionId = questionId {
                    viewModel.loadSnippets(questionId: questionId)
                }
            }
            .onAppear {
                handleOnAppear()
            }
            .onChange(of: viewModel.snippets.count) { _, _ in
                // Force view update when snippets change
                snippetRefreshTrigger += 1
            }
    }

    @ViewBuilder
    private var mainContent: some View {
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
                                translationSentence = TextUtils.extractSentence(
                                    from: fullText, containing: text)
                                showTranslationPopup = true
                            },
                            onSnippetTapped: { snippet in
                                showingSnippet = snippet
                            }
                        )
                        .id(questionCardId)

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
                                    translationSentence = TextUtils.extractSentence(
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
                    .onChange(of: viewModel.selectedAnswerIndex) { _, val in
                        if val != nil {
                            withAnimation {
                                proxy.scrollTo("bottom", anchor: .bottom)
                            }
                        }
                    }
                    .onChange(of: viewModel.answerResponse) { _, response in
                        if response != nil {
                            withAnimation {
                                proxy.scrollTo("bottom", anchor: .bottom)
                            }
                        }
                    }
            }
        }
    }

    private func handleOnAppear() {
        if viewModel.question == nil {
            viewModel.getQuestion()
        } else if let questionId = viewModel.question?.id {
            // Always reload snippets when view appears to ensure fresh data
            // This handles the case when navigating back to the quiz
            viewModel.loadSnippets(questionId: questionId)
        }
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
