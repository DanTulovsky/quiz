import SwiftUI

struct DailyView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject var viewModel = DailyViewModel()
    @EnvironmentObject var authViewModel: AuthenticationViewModel
    @State private var reportReason = ""
    @State private var selectedConfidence: Int?
    @State private var selectedText: String?
    @State private var showTranslationPopup = false
    @State private var translationSentence: String?
    @State private var showingSnippet: Snippet?
    @State var snippetRefreshTrigger: Int = 0

    var body: some View {
        ScrollView {
            ScrollViewReader { proxy in
                VStack(spacing: 20) {
                    if viewModel.isLoading && viewModel.dailyQuestions.isEmpty {
                        ProgressView("Loading Daily Challenge...")
                            .padding(.top, 50)
                    } else if viewModel.error != nil {
                        ErrorDisplay(
                            error: viewModel.error,
                            onDismiss: { viewModel.clearError() },
                            showRetryButton: true,
                            onRetry: { viewModel.fetchDaily() }
                        )
                    } else if !viewModel.isPositioned && !viewModel.dailyQuestions.isEmpty {
                        // Show loading while positioning to prevent showing question 0
                        ProgressView("Loading Daily Challenge...")
                            .padding(.top, 50)
                    } else if let question = viewModel.currentQuestion {
                        headerSection

                        QuestionCardView(
                            question: question.question,
                            snippets: viewModel.snippets,
                            onTextSelected: { text, fullText in
                                selectedText = text
                                translationSentence = TextUtils.extractSentence(
                                    from: fullText, containing: text)
                                showTranslationPopup = true
                            },
                            onSnippetTapped: { snippet in
                                showingSnippet = snippet
                            },
                            showLanguageBadge: false
                        )
                        .id(questionCardId)

                        QuestionOptionsView(
                            question: question.question,
                            selectedAnswerIndex: viewModel.selectedAnswerIndex,
                            correctAnswerIndex: question.isCompleted
                                ? question.question.correctAnswerIndex : nil,
                            userAnswerIndex: question.isCompleted ? question.userAnswerIndex : nil,
                            showResults: viewModel.answerResponse != nil || question.isCompleted,
                            onOptionSelected: { index in
                                viewModel.selectedAnswerIndex = index
                            }
                        )

                        if let response = viewModel.answerResponse {
                            AnswerFeedbackView(
                                isCorrect: response.isCorrect,
                                explanation: response.explanation,
                                language: question.question.language,
                                snippets: viewModel.snippets,
                                onTextSelected: { text, fullText in
                                    selectedText = text
                                    translationSentence = TextUtils.extractSentence(
                                        from: fullText, containing: text)
                                    showTranslationPopup = true
                                },
                                onSnippetTapped: { snippet in
                                    showingSnippet = snippet
                                },
                                showOverlay: true
                            )
                        } else if question.isCompleted {
                            // Show feedback for completed questions even if answerResponse is nil
                            let isCorrect =
                                question.userAnswerIndex == question.question.correctAnswerIndex
                            let explanation =
                                stringValue(question.question.content["explanation"])
                                ?? "Review your answer above."
                            AnswerFeedbackView(
                                isCorrect: isCorrect,
                                explanation: explanation,
                                language: question.question.language,
                                snippets: viewModel.snippets,
                                onTextSelected: { text, fullText in
                                    selectedText = text
                                    translationSentence = TextUtils.extractSentence(
                                        from: fullText, containing: text)
                                    showTranslationPopup = true
                                },
                                onSnippetTapped: { snippet in
                                    showingSnippet = snippet
                                },
                                showOverlay: true
                            )
                        }

                        actionButtons()

                        QuestionActionButtons(
                            isReported: viewModel.isReported,
                            onReport: { viewModel.showReportModal = true },
                            onMarkKnown: { viewModel.showMarkKnownModal = true }
                        )
                    } else if !viewModel.dailyQuestions.isEmpty {
                        completionView
                    }
                }
                .id("top")
                .padding()
                Color.clear.frame(height: 1).id("bottom")

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
                    .onChange(of: viewModel.currentQuestionIndex) { old, new in
                        // Scroll to top when switching questions (but not on initial load)
                        if old != new {
                            withAnimation {
                                proxy.scrollTo("top", anchor: .top)
                            }
                            // Clear snippets when question changes to avoid showing old snippets
                            viewModel.snippets = []
                            snippetRefreshTrigger += 1
                        }
                        // When navigating to a completed question, set the selected answer
                        if let question = viewModel.currentQuestion, question.isCompleted {
                            viewModel.selectedAnswerIndex = question.userAnswerIndex
                        }
                        // Fetch snippets for the new question
                        if let question = viewModel.currentQuestion {
                            viewModel.loadSnippets(questionId: question.question.id)
                        }
                    }
            }
        }
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
            if let text = selectedText, let question = viewModel.currentQuestion {
                TranslationPopupView(
                    selectedText: text,
                    sourceLanguage: question.question.language,
                    questionId: question.question.id,
                    sectionId: nil,
                    storyId: nil,
                    sentence: translationSentence,
                    onClose: {
                        showTranslationPopup = false
                        selectedText = nil
                        translationSentence = nil
                    },
                    onSnippetSaved: { snippet in
                        // Optimistically add the snippet immediately for instant UI update
                        if !viewModel.snippets.contains(where: { $0.id == snippet.id }) {
                            viewModel.snippets += [snippet]
                            snippetRefreshTrigger += 1
                        }
                        // Reload snippets from server to ensure we have the latest data
                        // This handles the case where the snippet already existed
                        if let questionId = viewModel.currentQuestion?.question.id {
                            // Reset cache to force reload
                            viewModel.resetSnippetCache()
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
        .onAppear {
            viewModel.fetchDaily()
            // Also check positioning after a delay to catch any edge cases
            Task { @MainActor in
                try? await Task.sleep(nanoseconds: 200_000_000)  // 0.2 seconds
                if !viewModel.dailyQuestions.isEmpty {
                    viewModel.validateCurrentQuestionPosition()
                    // Only load snippets if we don't already have them for this question
                    if let question = viewModel.currentQuestion {
                        if viewModel.snippets.isEmpty
                            || viewModel.snippets.first?.questionId != question.question.id {
                            viewModel.loadSnippets(questionId: question.question.id)
                        }
                    }
                }
            }
        }
        .onChange(of: viewModel.dailyQuestions.count) { _, new in
            // When questions count changes (questions loaded), ensure positioning
            if new > 0 {
                DispatchQueue.main.async {
                    viewModel.validateCurrentQuestionPosition()
                    // Load snippets for the current question after positioning
                    if let question = viewModel.currentQuestion {
                        viewModel.loadSnippets(questionId: question.question.id)
                    }
                }
            } else {
                // Reset positioning state when questions are cleared
                viewModel.isPositioned = false
            }
        }
        .onChange(of: viewModel.currentQuestion?.question.id) { oldQuestionId, newQuestionId in
            // When question changes, reload snippets
            // Only clear and reload if the question ID actually changed
            if let newQuestionId = newQuestionId, oldQuestionId != newQuestionId {
                viewModel.snippets = []
                snippetRefreshTrigger += 1
                viewModel.loadSnippets(questionId: newQuestionId)
            } else if newQuestionId == nil && oldQuestionId != nil {
                // Question became nil, clear snippets
                viewModel.snippets = []
                snippetRefreshTrigger += 1
            }
        }
        .onChange(of: viewModel.snippets.count) { oldCount, newCount in
            // Force view update when snippets change
            if oldCount != newCount {
                snippetRefreshTrigger += 1
            }
        }
    }

}
