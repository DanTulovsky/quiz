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
                        ProgressView()
                            .padding(.top, 50)
                    }

                    if let error = viewModel.error {
                        Text("Error: \(error.localizedDescription)")
                            .foregroundColor(.red)
                            .padding()
                            .background(Color.red.opacity(0.1))
                            .cornerRadius(8)
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
                        questionCard(question)

                        optionsList(question)

                        if let response = viewModel.answerResponse {
                            feedbackSection(response)
                        }

                        actionButtons()

                        footerButtons()
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
            reportSheet
        }
        .sheet(isPresented: $viewModel.showMarkKnownModal) {
            markKnownSheet
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
    private func questionCard(_ question: Question) -> some View {
        VStack(alignment: .leading, spacing: 15) {
            HStack {
                BadgeView(
                    text: question.type.replacingOccurrences(of: "_", with: " ").uppercased(),
                    color: AppTheme.Colors.accentIndigo)
                Spacer()
                BadgeView(
                    text: "\(question.language.uppercased()) - \(question.level)",
                    color: AppTheme.Colors.primaryBlue)
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
                    .id("\(passage)-\(viewModel.snippets.count)")
                    .frame(minHeight: 100)
                }
                .appInnerCard()
            }

            if let sentence = stringValue(question.content["sentence"]) {
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
                .id("\(sentence)-\(viewModel.snippets.count)")
                .frame(minHeight: 44)
            } else if let questionText = stringValue(question.content["question"])
                ?? stringValue(question.content["prompt"])
            {
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
                .id("\(questionText)-\(viewModel.snippets.count)")
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
                .id("vocab-\(targetWord)-\(viewModel.snippets.count)")
                .frame(minHeight: 44)
            }
        }
        .appCard()
    }

    @ViewBuilder
    private func highlightedText(_ fullText: String, targetWord: String?) -> some View {
        if let targetWord = targetWord,
            let range = fullText.range(of: targetWord, options: .caseInsensitive)
        {
            let before = String(fullText[..<range.lowerBound])
            let word = String(fullText[range])
            let after = String(fullText[range.upperBound...])

            Text(
                "\(Text(before))\(Text(word).foregroundColor(.blue).fontWeight(.bold))\(Text(after))"
            )
        } else {
            Text(fullText)
        }
    }

    @ViewBuilder
    private func optionsList(_ question: Question) -> some View {
        if let options = stringArrayValue(question.content["options"]) {
            VStack(spacing: 12) {
                ForEach(Array(options.enumerated()), id: \.offset) { idx, option in
                    optionButton(option: option, index: idx)
                }
            }
        }
    }

    @ViewBuilder
    private func optionButton(option: String, index: Int) -> some View {
        let isSelected = viewModel.selectedAnswerIndex == index
        let isCorrect = viewModel.answerResponse?.correctAnswerIndex == index
        let isUserIncorrect =
            viewModel.answerResponse != nil && viewModel.answerResponse?.userAnswerIndex == index
            && !isCorrect

        HStack {
            if viewModel.answerResponse != nil {
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
                    isUserIncorrect
                        ? AppTheme.Colors.errorRed
                        : (isCorrect
                            ? AppTheme.Colors.successGreen
                            : (isSelected ? .white : AppTheme.Colors.primaryText))
                )
                .frame(maxWidth: .infinity, alignment: .leading)

            Spacer()
        }
        .padding(AppTheme.Spacing.innerPadding)
        .frame(maxWidth: .infinity)
        .background(
            isUserIncorrect
                ? AppTheme.Colors.errorRed.opacity(0.1)
                : (isCorrect
                    ? AppTheme.Colors.successGreen.opacity(0.1)
                    : (isSelected
                        ? AppTheme.Colors.primaryBlue : AppTheme.Colors.primaryBlue.opacity(0.05)))
        )
        .cornerRadius(AppTheme.CornerRadius.button)
        .overlay(
            RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button)
                .stroke(
                    isUserIncorrect
                        ? AppTheme.Colors.errorRed
                        : (isCorrect ? AppTheme.Colors.successGreen : AppTheme.Colors.borderBlue),
                    lineWidth: 1)
        )
        .contentShape(Rectangle())
        .onTapGesture {
            if viewModel.answerResponse == nil {
                viewModel.selectedAnswerIndex = index
            }
        }
        .disabled(viewModel.answerResponse != nil)
    }

    @ViewBuilder
    private func feedbackSection(_ response: AnswerResponse) -> some View {
        let question = viewModel.question

        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Image(
                    systemName: response.isCorrect ? "checkmark.circle.fill" : "xmark.circle.fill"
                )
                .foregroundColor(
                    response.isCorrect ? AppTheme.Colors.successGreen : AppTheme.Colors.errorRed)
                Text(response.isCorrect ? "Correct!" : "Incorrect")
                    .font(AppTheme.Typography.headingFont)
                    .foregroundColor(
                        response.isCorrect ? AppTheme.Colors.successGreen : AppTheme.Colors.errorRed
                    )
                    .textSelection(.enabled)
            }

            SelectableTextView(
                text: response.explanation,
                language: question?.language ?? "en",
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
