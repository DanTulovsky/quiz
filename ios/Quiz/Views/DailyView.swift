import SwiftUI

struct DailyView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel = DailyViewModel()
    @EnvironmentObject var authViewModel: AuthenticationViewModel
    @State private var reportReason = ""
    @State private var selectedConfidence: Int? = nil

    var body: some View {
        ScrollView {
            ScrollViewReader { proxy in
                VStack(spacing: 20) {
                    if viewModel.isLoading && viewModel.dailyQuestions.isEmpty {
                        ProgressView("Loading Daily Challenge...")
                            .padding(.top, 50)
                    } else if let error = viewModel.error {
                        errorView(error)
                    } else if let question = viewModel.currentQuestion {
                        headerSection

                        questionCard(question.question)

                        optionsList(question.question)

                        if let response = viewModel.answerResponse {
                            feedbackSection(response)
                        } else if question.isCompleted {
                            // Show feedback for completed questions even if answerResponse is nil
                            completedQuestionFeedback(question)
                        }

                        actionButtons()

                        footerButtons()
                    } else if !viewModel.dailyQuestions.isEmpty {
                        completionView
                    }
                }
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
                        // When navigating to a completed question, set the selected answer
                        if let question = viewModel.currentQuestion, question.isCompleted {
                            viewModel.selectedAnswerIndex = question.userAnswerIndex
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
                            .font(.system(size: 17, weight: .semibold))
                        Text("Back")
                            .font(.system(size: 17))
                    }
                    .foregroundColor(.blue)
                }
            }
        }
        .sheet(isPresented: $viewModel.showReportModal) {
            reportSheet
        }
        .sheet(isPresented: $viewModel.showMarkKnownModal) {
            markKnownSheet
        }
        .onAppear {
            viewModel.fetchDaily()
        }
    }

    private var headerSection: some View {
        VStack(spacing: AppTheme.Spacing.itemSpacing) {
            HStack {
                BadgeView(text: "DAILY CHALLENGE", color: AppTheme.Colors.accentIndigo)
                Spacer()
                BadgeView(text: "\(viewModel.currentQuestion?.question.language.uppercased() ?? "") - \(viewModel.currentQuestion?.question.level ?? "")", color: AppTheme.Colors.primaryBlue)
            }

            HStack {
                Text(Date(), style: .date)
                    .font(.subheadline)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 8)
                    .background(Color(.systemBackground))
                    .cornerRadius(8)
                    .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.gray.opacity(0.2), lineWidth: 1))

                Spacer()

                BadgeView(text: "\(viewModel.currentQuestionIndex + 1) OF \(viewModel.dailyQuestions.count)", color: .blue)
            }

            ProgressView(value: viewModel.progress)
                .accentColor(AppTheme.Colors.primaryBlue)
                .scaleEffect(x: 1, y: 2, anchor: .center)
                .clipShape(RoundedRectangle(cornerRadius: 4))
        }
        .appCard()
    }

    private func questionCard(_ question: Question) -> some View {
        VStack(alignment: .leading, spacing: 15) {
            HStack {
                BadgeView(text: question.type.replacingOccurrences(of: "_", with: " ").uppercased(), color: AppTheme.Colors.accentIndigo)

                Spacer()
            }

            if let passage = stringValue(question.content["passage"]) {
                VStack(alignment: .trailing) {
                    TTSButton(text: passage, language: question.language)
                    Text(passage)
                        .font(.body)
                        .lineSpacing(4)
                }
                .appInnerCard()
            }

            let sentence = stringValue(question.content["sentence"])
            let questionText = stringValue(question.content["question"]) ?? stringValue(question.content["prompt"])

            if let sentence = sentence {
                let targetWord = stringValue(question.content["question"])
                highlightedText(sentence, targetWord: targetWord)
                    .font(.title2)
                    .fontWeight(.bold)
                    .lineSpacing(4)
            } else if let questionText = questionText {
                Text(questionText)
                    .font(.title2)
                    .fontWeight(.bold)
            }

            if question.type == "vocabulary", let targetWord = stringValue(question.content["question"]) {
                Text("What does **\(targetWord)** mean in this context?")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }
        }
        .appCard()
    }

    private func optionsList(_ question: Question) -> some View {
        VStack(spacing: 12) {
            if let options = stringArrayValue(question.content["options"]) {
                ForEach(Array(options.enumerated()), id: \.offset) { idx, option in
                    optionButton(option: option, index: idx)
                }
            }
        }
    }

    private func optionButton(option: String, index: Int) -> some View {
        let currentQuestion = viewModel.currentQuestion
        let isCompleted = currentQuestion?.isCompleted ?? false
        let isSelected = viewModel.selectedAnswerIndex == index
        let correctAnswerIndex = viewModel.answerResponse?.correctAnswerIndex ?? currentQuestion?.question.correctAnswerIndex
        let isCorrect = correctAnswerIndex == index
        let userAnswerIndex = viewModel.answerResponse?.userAnswerIndex ?? currentQuestion?.userAnswerIndex
        let isUserIncorrect = userAnswerIndex == index && !isCorrect
        let showResults = viewModel.answerResponse != nil || isCompleted

        return Button(action: {
            if !showResults {
                viewModel.selectedAnswerIndex = index
            }
        }) {
            HStack {
                if showResults {
                    if isCorrect {
                        Image(systemName: "check.circle.fill")
                            .foregroundColor(.green)
                    } else if isUserIncorrect {
                        Image(systemName: "x.circle.fill")
                            .foregroundColor(.red)
                    }
                }

                Text(option)
                    .font(.body)
                    .foregroundColor(isUserIncorrect ? .red : (isCorrect ? .green : (isSelected ? .white : .primary)))

                Spacer()
            }
            .padding()
            .frame(maxWidth: .infinity)
            .buttonStyle(OptionButtonStyle(
                isSelected: isSelected,
                isCorrect: isCorrect,
                isIncorrect: isUserIncorrect
            ))
        }
        .disabled(showResults)
    }

    private func feedbackSection(_ response: DailyAnswerResponse) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: response.isCorrect ? "checkmark" : "xmark")
                    .foregroundColor(response.isCorrect ? .green : .red)
                Text(response.isCorrect ? "Correct!" : "Incorrect")
                    .font(.headline)
                    .foregroundColor(response.isCorrect ? .green : .red)
            }

            Text(response.explanation)
                .font(.subheadline)
                .foregroundColor(.primary)
                .fixedSize(horizontal: false, vertical: true)
        }
        .padding(AppTheme.Spacing.innerPadding)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(response.isCorrect ? AppTheme.Colors.successGreen.opacity(0.05) : AppTheme.Colors.errorRed.opacity(0.05))
        .cornerRadius(AppTheme.CornerRadius.button)
        .overlay(RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(response.isCorrect ? AppTheme.Colors.successGreen.opacity(0.2) : AppTheme.Colors.errorRed.opacity(0.2), lineWidth: 1))
    }

    private func completedQuestionFeedback(_ question: DailyQuestionWithDetails) -> some View {
        // For completed questions, determine if the answer was correct
        let correctAnswerIndex = question.question.correctAnswerIndex
        let userAnswerIndex = question.userAnswerIndex
        let isCorrect = userAnswerIndex == correctAnswerIndex

        // Try to get explanation from question content or use a default message
        let explanation = stringValue(question.question.content["explanation"]) ?? "Review your answer above."

        return VStack(alignment: .leading, spacing: AppTheme.Spacing.itemSpacing) {
            HStack {
                Image(systemName: isCorrect ? "checkmark" : "xmark")
                    .foregroundColor(isCorrect ? AppTheme.Colors.successGreen : AppTheme.Colors.errorRed)
                Text(isCorrect ? "Correct!" : "Incorrect")
                    .font(AppTheme.Typography.headingFont)
                    .foregroundColor(isCorrect ? AppTheme.Colors.successGreen : AppTheme.Colors.errorRed)
            }

            Text(explanation)
                .font(AppTheme.Typography.subheadlineFont)
                .foregroundColor(AppTheme.Colors.primaryText)
                .fixedSize(horizontal: false, vertical: true)
        }
        .padding(AppTheme.Spacing.innerPadding)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(isCorrect ? AppTheme.Colors.successGreen.opacity(0.05) : AppTheme.Colors.errorRed.opacity(0.05))
        .cornerRadius(AppTheme.CornerRadius.button)
        .overlay(RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(isCorrect ? AppTheme.Colors.successGreen.opacity(0.2) : AppTheme.Colors.errorRed.opacity(0.2), lineWidth: 1))
    }

    private var completionView: some View {
        VStack(spacing: 20) {
            Image(systemName: "trophy.fill")
                .font(.system(size: 80))
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

    private func errorView(_ error: Error) -> some View {
        VStack(spacing: 15) {
            Image(systemName: "exclamationmark.triangle")
                .font(.largeTitle)
                .foregroundColor(.red)
            Text("Error: \(error.localizedDescription)")
                .multilineTextAlignment(.center)
            Button("Retry") {
                viewModel.fetchDaily()
            }
            .buttonStyle(.bordered)
        }
        .padding()
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

    private func highlightedText(_ fullText: String, targetWord: String?) -> some View {
        if let targetWord = targetWord, let range = fullText.range(of: targetWord, options: .caseInsensitive) {
            let before = String(fullText[..<range.lowerBound])
            let word = String(fullText[range])
            let after = String(fullText[range.upperBound...])

            return Text("\(Text(before))\(Text(word).foregroundColor(.blue).fontWeight(.bold))\(Text(after))")
        } else {
            return Text(fullText)
        }
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
        } else if let _ = viewModel.answerResponse {
            // After submission but not all completed
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
            .buttonStyle(PrimaryButtonStyle(isDisabled: viewModel.selectedAnswerIndex == nil || viewModel.isSubmitting))
        }
    }

    @ViewBuilder
    private func footerButtons() -> some View {
        HStack(spacing: 20) {
            Button(action: { viewModel.showReportModal = true }) {
                Label(viewModel.isReported ? "Reported" : "Report issue", systemImage: "flag")
                    .font(.caption)
            }
            .disabled(viewModel.isReported)
            .foregroundColor(.secondary)

            Spacer()

            Button(action: { viewModel.showMarkKnownModal = true }) {
                Label("Adjust frequency", systemImage: "slider.horizontal.3")
                    .font(.caption)
            }
            .foregroundColor(.secondary)
        }
        .padding(.top, 10)
    }

    private var reportSheet: some View {
        NavigationView {
            Form {
                Section(header: Text("Why are you reporting this question?")) {
                    TextEditor(text: $reportReason)
                        .frame(minHeight: 100)
                }

                Button("Submit Report") {
                    viewModel.reportQuestion(reason: reportReason)
                }
                .disabled(viewModel.isSubmittingAction)
            }
            .navigationTitle("Report Issue")
            .navigationBarItems(trailing: Button("Cancel") { viewModel.showReportModal = false })
        }
    }

    private var markKnownSheet: some View {
        NavigationView {
            VStack(spacing: 20) {
                Text("Choose how often you want to see this question in future quizzes: 1–2 show it more, 3 no change, 4–5 show it less.")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .padding()

                Text("How confident are you about this question?")
                    .font(.headline)

                HStack(spacing: 10) {
                    ForEach(1...5, id: \.self) { level in
                        Button("\(level)") {
                            selectedConfidence = level
                        }
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(selectedConfidence == level ? AppTheme.Colors.primaryBlue : AppTheme.Colors.primaryBlue.opacity(0.1))
                        .foregroundColor(selectedConfidence == level ? .white : AppTheme.Colors.primaryBlue)
                        .cornerRadius(AppTheme.CornerRadius.button)
                    }
                }
                .padding(.horizontal)

                Spacer()

                Button("Save Preference") {
                    if let confidence = selectedConfidence {
                        viewModel.markQuestionKnown(confidence: confidence)
                    }
                }
                .buttonStyle(.borderedProminent)
                .tint(.teal)
                .disabled(selectedConfidence == nil || viewModel.isSubmittingAction)
                .padding()
            }
            .navigationTitle("Adjust Frequency")
            .navigationBarItems(trailing: Button("Cancel") { viewModel.showMarkKnownModal = false })
        }
    }
}
