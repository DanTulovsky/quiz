import SwiftUI

struct DailyView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel = DailyViewModel()
    @EnvironmentObject var authViewModel: AuthenticationViewModel
    @State private var reportReason = ""
    @State private var selectedConfidence: Int? = nil
    @State private var selectedText: String?
    @State private var showTranslationPopup = false
    @State private var translationSentence: String?
    @State private var showingSnippet: Snippet? = nil

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
                    } else if let question = viewModel.currentQuestion {
                        headerSection

                        QuestionCardView(
                            question: question.question,
                            snippets: viewModel.snippets,
                            onTextSelected: { text, fullText in
                                selectedText = text
                                translationSentence = extractSentence(
                                    from: fullText, containing: text)
                                showTranslationPopup = true
                            },
                            onSnippetTapped: { snippet in
                                showingSnippet = snippet
                            },
                            showLanguageBadge: false
                        )

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
                                    translationSentence = extractSentence(
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
                                    translationSentence = extractSentence(
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
                    .onChange(of: viewModel.currentQuestionIndex) { old, new in
                        // Scroll to top when switching questions (but not on initial load)
                        if old != new {
                            withAnimation {
                                proxy.scrollTo("top", anchor: .top)
                            }
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
                viewModel.reportQuestion(reason: reason)
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
                    onSnippetSaved: {
                        if let questionId = viewModel.currentQuestion?.question.id {
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
                    viewModel.ensurePositionedOnFirstIncomplete()
                    if let question = viewModel.currentQuestion {
                        viewModel.loadSnippets(questionId: question.question.id)
                    }
                }
            }
        }
        .onChange(of: viewModel.dailyQuestions.count) { old, new in
            // When questions count changes (questions loaded), ensure positioning
            if new > 0 {
                DispatchQueue.main.async {
                    viewModel.ensurePositionedOnFirstIncomplete()
                }
            }
        }
    }

    private var headerSection: some View {
        VStack(spacing: AppTheme.Spacing.itemSpacing) {
            HStack {
                BadgeView(text: "DAILY CHALLENGE", color: AppTheme.Colors.accentIndigo)
                Spacer()
                BadgeView(
                    text:
                        "\(viewModel.currentQuestion?.question.language.uppercased() ?? "") - \(viewModel.currentQuestion?.question.level ?? "")",
                    color: AppTheme.Colors.primaryBlue)
            }

            HStack {
                Text(Date(), style: .date)
                    .font(.subheadline)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 8)
                    .background(Color(.systemBackground))
                    .cornerRadius(8)
                    .overlay(
                        RoundedRectangle(cornerRadius: 8).stroke(
                            Color.gray.opacity(0.2), lineWidth: 1))

                Spacer()

                BadgeView(
                    text:
                        "\(viewModel.currentQuestionIndex + 1) OF \(viewModel.dailyQuestions.count)",
                    color: .blue)
            }

            ProgressView(value: viewModel.progress)
                .accentColor(AppTheme.Colors.primaryBlue)
                .scaleEffect(x: 1, y: 2, anchor: .center)
                .clipShape(RoundedRectangle(cornerRadius: 4))
        }
        .appCard()
    }

    private var completionView: some View {
        VStack(spacing: 20) {
            Image(systemName: "trophy.fill")
                .scaledFont(size: 80)
                .foregroundColor(AppTheme.Colors.primaryBlue)

            Text("Daily Challenge Complete!")
                .font(.title)
                .fontWeight(.bold)

            Text("You've finished all your questions for today. Great job!")
                .multilineTextAlignment(.center)
                .foregroundColor(.secondary)

            Button("Back to Home") {
                // This would ideally pop back or switch tabs
            }
            .buttonStyle(.borderedProminent)
        }
        .padding(.top, 50)
    }

    // Helpers
    private func stringValue(_ v: JSONValue?) -> String? {
        guard let v else { return nil }
        if case .string(let s) = v { return s }
        return nil
    }

    private func stringArrayValue(_ v: JSONValue?) -> [String]? {
        guard let v else { return nil }
        guard case .array(let arr) = v else { return nil }
        return arr.compactMap { item -> String? in
            if case .string(let s) = item { return s }
            return nil
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
        if viewModel.isAllCompleted {
            // When all completed, show Previous/Next navigation
            HStack(spacing: 12) {
                Button("Previous") {
                    viewModel.previousQuestion()
                }
                .buttonStyle(.bordered)
                .controlSize(.large)
                .disabled(!viewModel.hasPreviousQuestion)
                .frame(maxWidth: .infinity)

                Button("Next") {
                    viewModel.nextQuestion()
                }
                .buttonStyle(.bordered)
                .controlSize(.large)
                .disabled(!viewModel.hasNextQuestion)
                .frame(maxWidth: .infinity)
            }
        } else if viewModel.answerResponse != nil {
            // After submission but not all completed
            Button("Next Question") {
                viewModel.nextQuestion()
            }
            .buttonStyle(PrimaryButtonStyle())
        } else if let question = viewModel.currentQuestion, question.isCompleted {
            // Viewing a completed question (answerResponse is nil but question is completed)
            Button("Next Question") {
                viewModel.nextQuestion()
            }
            .buttonStyle(PrimaryButtonStyle())
        } else {
            // Not submitted yet
            Button("Submit Answer") {
                if let idx = viewModel.selectedAnswerIndex {
                    viewModel.submitAnswer(index: idx)
                }
            }
            .buttonStyle(
                PrimaryButtonStyle(
                    isDisabled: viewModel.selectedAnswerIndex == nil || viewModel.isSubmitting))
        }
    }

}
