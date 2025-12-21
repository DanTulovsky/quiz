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
            reportSheet
        }
        .sheet(isPresented: $viewModel.showMarkKnownModal) {
            markKnownSheet
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

    private func questionCard(_ question: Question) -> some View {
        VStack(alignment: .leading, spacing: 15) {
            HStack {
                BadgeView(
                    text: question.type.replacingOccurrences(of: "_", with: " ").uppercased(),
                    color: AppTheme.Colors.accentIndigo)

                Spacer()
            }

            if let passage = stringValue(question.content["passage"]) {
                VStack(alignment: .trailing) {
                    TTSButton(text: passage, language: question.language)
                    SelectableTextView(
                        text: passage,
                        language: question.language,
                        onTextSelected: { text in
                            selectedText = text
                            translationSentence = extractSentence(from: passage, containing: text)
                            showTranslationPopup = true
                        },
                        highlightedSnippets: viewModel.snippets,
                        onSnippetTapped: { snippet in
                            showingSnippet = snippet
                        }
                    )
                    .id("passage-\(viewModel.snippets.count)")
                    .frame(minHeight: 100)
                }
                .appInnerCard()
            }

            let sentence = stringValue(question.content["sentence"])
            let questionText =
                stringValue(question.content["question"]) ?? stringValue(question.content["prompt"])

            if let sentence = sentence {
                SelectableTextView(
                    text: sentence,
                    language: question.language,
                    onTextSelected: { text in
                        selectedText = text
                        translationSentence = extractSentence(from: sentence, containing: text)
                        showTranslationPopup = true
                    },
                    highlightedSnippets: viewModel.snippets,
                    onSnippetTapped: { snippet in
                        showingSnippet = snippet
                    }
                )
                .id("sentence-\(viewModel.snippets.count)")
                .frame(minHeight: 44)
            } else if let questionText = questionText {
                SelectableTextView(
                    text: questionText,
                    language: question.language,
                    onTextSelected: { text in
                        selectedText = text
                        translationSentence = extractSentence(from: questionText, containing: text)
                        showTranslationPopup = true
                    },
                    highlightedSnippets: viewModel.snippets,
                    onSnippetTapped: { snippet in
                        showingSnippet = snippet
                    }
                )
                .id("question-\(viewModel.snippets.count)")
                .frame(minHeight: 44)
            }

            if question.type == "vocabulary",
                let targetWord = stringValue(question.content["question"])
            {
                SelectableTextView(
                    text: "What does \(targetWord) mean in this context?",
                    language: question.language,
                    onTextSelected: { text in
                        selectedText = text
                        translationSentence = extractSentence(
                            from: "What does \(targetWord) mean in this context?", containing: text)
                        showTranslationPopup = true
                    },
                    highlightedSnippets: viewModel.snippets,
                    onSnippetTapped: { snippet in
                        showingSnippet = snippet
                    }
                )
                .id("vocab-\(viewModel.snippets.count)")
                .frame(minHeight: 44)
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
        let showResults = viewModel.answerResponse != nil || isCompleted

        // Only get correct answer info when we should show results
        let correctAnswerIndex: Int? =
            showResults
            ? (viewModel.answerResponse?.correctAnswerIndex
                ?? currentQuestion?.question.correctAnswerIndex) : nil
        let isCorrect = correctAnswerIndex != nil && correctAnswerIndex == index
        let userAnswerIndex =
            viewModel.answerResponse?.userAnswerIndex
            ?? (isCompleted ? currentQuestion?.userAnswerIndex : nil)
        let isUserIncorrect = userAnswerIndex != nil && userAnswerIndex == index && !isCorrect

        return HStack {
            if showResults {
                if isCorrect {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundColor(AppTheme.Colors.successGreen)
                } else if isUserIncorrect {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundColor(AppTheme.Colors.errorRed)
                }
            }

            Text(option)
                .font(AppTheme.Typography.bodyFont)
                .foregroundColor(
                    showResults && isUserIncorrect
                        ? AppTheme.Colors.errorRed
                        : (showResults && isCorrect
                            ? AppTheme.Colors.successGreen
                            : (isSelected ? .white : AppTheme.Colors.primaryText))
                )
                .frame(maxWidth: .infinity, alignment: .leading)

            Spacer()
        }
        .padding(AppTheme.Spacing.innerPadding)
        .frame(maxWidth: .infinity)
        .background(
            showResults && isUserIncorrect
                ? AppTheme.Colors.errorRed.opacity(0.1)
                : (showResults && isCorrect
                    ? AppTheme.Colors.successGreen.opacity(0.1)
                    : (isSelected
                        ? AppTheme.Colors.primaryBlue : AppTheme.Colors.primaryBlue.opacity(0.05)))
        )
        .cornerRadius(AppTheme.CornerRadius.button)
        .overlay(
            RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button)
                .stroke(
                    showResults && isUserIncorrect
                        ? AppTheme.Colors.errorRed
                        : (showResults && isCorrect
                            ? AppTheme.Colors.successGreen : AppTheme.Colors.borderBlue),
                    lineWidth: 1)
        )
        .contentShape(Rectangle())
        .onTapGesture {
            if !showResults {
                viewModel.selectedAnswerIndex = index
            }
        }
        .disabled(showResults)
    }

    @ViewBuilder
    private func feedbackSection(_ response: DailyAnswerResponse) -> some View {
        let language = viewModel.currentQuestion?.question.language ?? "en"

        VStack(alignment: .leading, spacing: AppTheme.Spacing.itemSpacing) {
            HStack {
                Image(systemName: response.isCorrect ? "checkmark" : "xmark")
                    .foregroundColor(
                        response.isCorrect ? AppTheme.Colors.successGreen : AppTheme.Colors.errorRed
                    )
                Text(response.isCorrect ? "Correct!" : "Incorrect")
                    .font(AppTheme.Typography.headingFont)
                    .foregroundColor(
                        response.isCorrect ? AppTheme.Colors.successGreen : AppTheme.Colors.errorRed
                    )
            }

            SelectableTextView(
                text: response.explanation,
                language: language,
                onTextSelected: { text in
                    selectedText = text
                    translationSentence = extractSentence(
                        from: response.explanation, containing: text)
                    showTranslationPopup = true
                },
                highlightedSnippets: viewModel.snippets,
                onSnippetTapped: { snippet in
                    showingSnippet = snippet
                }
            )
            .id("explanation-\(viewModel.snippets.count)")
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(AppTheme.Spacing.innerPadding)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(
            response.isCorrect
                ? AppTheme.Colors.successGreen.opacity(0.05)
                : AppTheme.Colors.errorRed.opacity(0.05)
        )
        .cornerRadius(AppTheme.CornerRadius.button)
        .overlay(
            RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(
                response.isCorrect
                    ? AppTheme.Colors.successGreen.opacity(0.2)
                    : AppTheme.Colors.errorRed.opacity(0.2), lineWidth: 1))
    }

    private func completedQuestionFeedback(_ question: DailyQuestionWithDetails) -> some View {
        // For completed questions, determine if the answer was correct
        let correctAnswerIndex = question.question.correctAnswerIndex
        let userAnswerIndex = question.userAnswerIndex
        let isCorrect = userAnswerIndex == correctAnswerIndex

        // Try to get explanation from question content or use a default message
        let explanation =
            stringValue(question.question.content["explanation"]) ?? "Review your answer above."

        return VStack(alignment: .leading, spacing: AppTheme.Spacing.itemSpacing) {
            HStack {
                Image(systemName: isCorrect ? "checkmark" : "xmark")
                    .foregroundColor(
                        isCorrect ? AppTheme.Colors.successGreen : AppTheme.Colors.errorRed)
                Text(isCorrect ? "Correct!" : "Incorrect")
                    .font(AppTheme.Typography.headingFont)
                    .foregroundColor(
                        isCorrect ? AppTheme.Colors.successGreen : AppTheme.Colors.errorRed)
            }

            SelectableTextView(
                text: explanation,
                language: question.question.language,
                onTextSelected: { text in
                    selectedText = text
                    translationSentence = extractSentence(from: explanation, containing: text)
                    showTranslationPopup = true
                },
                highlightedSnippets: viewModel.snippets,
                onSnippetTapped: { snippet in
                    showingSnippet = snippet
                }
            )
            .id("completed-explanation-\(viewModel.snippets.count)")
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(AppTheme.Spacing.innerPadding)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(
            isCorrect
                ? AppTheme.Colors.successGreen.opacity(0.05)
                : AppTheme.Colors.errorRed.opacity(0.05)
        )
        .cornerRadius(AppTheme.CornerRadius.button)
        .overlay(
            RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(
                isCorrect
                    ? AppTheme.Colors.successGreen.opacity(0.2)
                    : AppTheme.Colors.errorRed.opacity(0.2), lineWidth: 1))
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
        if let targetWord = targetWord,
            let range = fullText.range(of: targetWord, options: .caseInsensitive)
        {
            let before = String(fullText[..<range.lowerBound])
            let word = String(fullText[range])
            let after = String(fullText[range.upperBound...])

            return Text(
                "\(Text(before))\(Text(word).foregroundColor(.blue).fontWeight(.bold))\(Text(after))"
            )
        } else {
            return Text(fullText)
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
                Text(
                    "Choose how often you want to see this question in future quizzes: 1–2 show it more, 3 no change, 4–5 show it less."
                )
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
                        .background(
                            selectedConfidence == level
                                ? AppTheme.Colors.primaryBlue
                                : AppTheme.Colors.primaryBlue.opacity(0.1)
                        )
                        .foregroundColor(
                            selectedConfidence == level ? .white : AppTheme.Colors.primaryBlue
                        )
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
                .buttonStyle(
                    PrimaryButtonStyle(
                        isDisabled: selectedConfidence == nil || viewModel.isSubmittingAction)
                )
                .padding()
            }
            .navigationTitle("Adjust Frequency")
            .navigationBarItems(trailing: Button("Cancel") { viewModel.showMarkKnownModal = false })
        }
    }
}
