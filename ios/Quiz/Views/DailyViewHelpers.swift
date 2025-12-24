import Foundation
import SwiftUI

extension DailyView {
    func stringValue(_ value: JSONValue?) -> String? {
        guard let value else { return nil }
        if case .string(let stringValue) = value { return stringValue }
        return nil
    }

    func stringArrayValue(_ value: JSONValue?) -> [String]? {
        guard let value else { return nil }
        guard case .array(let arr) = value else { return nil }
        return arr.compactMap { item -> String? in
            if case .string(let stringValue) = item { return stringValue }
            return nil
        }
    }

    func extractSentence(from text: String, containing selectedText: String) -> String? {
        return TextUtils.extractSentence(from: text, containing: selectedText)
    }

    @ViewBuilder
    func actionButtons() -> some View {
        if viewModel.isAllCompleted {
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
            Button("Next Question") {
                viewModel.nextQuestion()
            }
            .buttonStyle(PrimaryButtonStyle())
        } else if let question = viewModel.currentQuestion, question.isCompleted {
            Button("Next Question") {
                viewModel.nextQuestion()
            }
            .buttonStyle(PrimaryButtonStyle())
        } else {
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

    var questionCardId: String {
        guard let question = viewModel.currentQuestion else { return "" }
        let snippetIds = viewModel.snippets.map { "\($0.id)" }.joined(separator: ",")
        return
            "question-\(question.question.id)-snippets-\(viewModel.snippets.count)-"
            + "\(snippetIds)-\(snippetRefreshTrigger)"
    }

    var headerSection: some View {
        VStack(spacing: AppTheme.Spacing.itemSpacing) {
            HStack {
                BadgeView(text: "DAILY CHALLENGE", color: AppTheme.Colors.accentIndigo)
                Spacer()
                BadgeView(
                    text:
                        "\(viewModel.currentQuestion?.question.language.uppercased() ?? "") - "
                        + "\(viewModel.currentQuestion?.question.level ?? "")",
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
                        viewModel.currentQuestionIndex >= 0
                        ? "\(viewModel.currentQuestionIndex + 1) OF \(viewModel.dailyQuestions.count)"
                        : "0 OF \(viewModel.dailyQuestions.count)",
                    color: .blue)
            }

            ProgressView(value: viewModel.progress)
                .accentColor(AppTheme.Colors.primaryBlue)
                .scaleEffect(x: 1, y: 2, anchor: .center)
                .clipShape(RoundedRectangle(cornerRadius: 4))
        }
        .appCard()
    }

    var completionView: some View {
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
}

struct DailyQuestionContentView: View {
    let question: DailyQuestionWithDetails
    let viewModel: DailyViewModel
    let questionCardId: String
    let onTextSelected: (String, String) -> Void
    let onSnippetTapped: (Snippet) -> Void
    let stringValue: (JSONValue?) -> String?
    let actionButtons: () -> AnyView

    var body: some View {
        VStack(spacing: 20) {
            QuestionCardView(
                question: question.question,
                snippets: viewModel.snippets,
                onTextSelected: onTextSelected,
                onSnippetTapped: onSnippetTapped,
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
                    onTextSelected: onTextSelected,
                    onSnippetTapped: onSnippetTapped,
                    showOverlay: true
                )
            } else if question.isCompleted {
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
                    onTextSelected: onTextSelected,
                    onSnippetTapped: onSnippetTapped,
                    showOverlay: true
                )
            }

            actionButtons()

            QuestionActionButtons(
                isReported: viewModel.isReported,
                onReport: { viewModel.showReportModal = true },
                onMarkKnown: { viewModel.showMarkKnownModal = true }
            )
        }
    }
}
