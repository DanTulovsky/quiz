import SwiftUI

extension StoryDetailView {
    func extractSentence(from text: String, containing selectedText: String) -> String? {
        return TextUtils.extractSentence(from: text, containing: selectedText)
    }

    func highlightSnippets(in text: String) -> AttributedString {
        var attrStr = AttributedString(text)
        let sortedSnippets = viewModel.snippets.sorted {
            $0.originalText.count > $1.originalText.count
        }

        for snippet in sortedSnippets {
            var searchRange = attrStr.startIndex..<attrStr.endIndex
            while let range = attrStr[searchRange].range(
                    of: snippet.originalText, options: .caseInsensitive) {
                attrStr[range].underlineStyle = Text.LineStyle(pattern: .dash)
                attrStr[range].foregroundColor = .blue
                if let url = URL(string: "snippet://\(snippet.id)") {
                    attrStr[range].link = url
                }
                searchRange = range.upperBound..<attrStr.endIndex
            }
        }
        return attrStr
    }

    @ViewBuilder
    func optionRow(
        question: StorySectionQuestion, idx: Int, option: String, hasSubmitted: Bool,
        selectedIdx: Int?
    ) -> some View {
        let isCorrect = idx == question.correctAnswerIndex
        let isSelected = selectedIdx == idx

        HStack {
            if hasSubmitted {
                Image(
                    systemName: isCorrect
                        ? "checkmark.circle.fill" : (isSelected ? "xmark.circle.fill" : "circle")
                )
                .foregroundColor(isCorrect ? .green : (isSelected ? .red : .gray))
            } else {
                Circle()
                    .stroke(isSelected ? Color.blue : Color.gray, lineWidth: 1)
                    .frame(width: 18, height: 18)
                    .overlay(Circle().fill(isSelected ? Color.blue : Color.clear).padding(4))
            }

            Text(option)
                .font(AppTheme.Typography.subheadlineFont)
                .foregroundColor(
                    hasSubmitted
                        ? (isCorrect
                            ? AppTheme.Colors.successGreen
                            : (isSelected ? AppTheme.Colors.errorRed : AppTheme.Colors.primaryText))
                        : AppTheme.Colors.primaryText
                )
                .frame(maxWidth: .infinity, alignment: .leading)

            Spacer()
        }
        .padding(10)
        .background(
            hasSubmitted
                ? (isCorrect
                    ? AppTheme.Colors.successGreen.opacity(0.1)
                    : (isSelected
                        ? AppTheme.Colors.errorRed.opacity(0.1)
                        : AppTheme.Colors.secondaryBackground))
                : (isSelected
                    ? AppTheme.Colors.primaryBlue.opacity(0.1)
                    : AppTheme.Colors.secondaryBackground)
        )
        .cornerRadius(AppTheme.CornerRadius.badge)
        .contentShape(Rectangle())
        .onTapGesture {
            if !hasSubmitted {
                selectedAnswers[question.id] = idx
            }
        }
        .disabled(hasSubmitted)
    }

    @ViewBuilder
    func submitButton(questionId: Int, selectedIdx: Int?) -> some View {
        Button(
            action: {
                submittedQuestions.insert(questionId)
            },
            label: {
                Text("Submit Answer")
                    .font(AppTheme.Typography.subheadlineFont.weight(.bold))
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
                    .background(
                        selectedIdx == nil
                            ? AppTheme.Colors.primaryBlue.opacity(0.3)
                            : AppTheme.Colors.primaryBlue
                    )
                    .foregroundColor(.white)
                    .cornerRadius(AppTheme.CornerRadius.badge)
            }
        )
        .disabled(selectedIdx == nil)
        .padding(.top, 4)
    }

    @ViewBuilder
    func explanationView(explanation: String, question: StorySectionQuestion) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Divider()
            Text("Explanation")
                .font(AppTheme.Typography.captionFont.weight(.bold))
                .foregroundColor(AppTheme.Colors.secondaryText)
            SelectableTextView(
                text: explanation,
                language: viewModel.selectedStory?.language ?? "en",
                onTextSelected: { text in
                    selectedText = text
                    translationSentence = TextUtils.extractSentence(
                        from: explanation, containing: text)
                    showTranslationPopup = true
                }
            )
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(.top, 4)
    }

    @ViewBuilder
    func sectionContent(_ section: StorySectionWithQuestions) -> some View {
        VStack(alignment: .trailing) {
            TTSButton(text: section.content, language: viewModel.selectedStory?.language ?? "en")

            VStack(alignment: .leading) {
                SelectableTextView(
                    text: section.content,
                    language: viewModel.selectedStory?.language ?? "en",
                    onTextSelected: { text in
                        selectedText = text
                        translationSentence = TextUtils.extractSentence(
                            from: section.content, containing: text)
                        showTranslationPopup = true
                    },
                    highlightedSnippets: viewModel.snippets,
                    onSnippetTapped: { snippet in
                        showingSnippet = snippet
                    }
                )
                .frame(minHeight: 100)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .appInnerCard()
        .padding(.horizontal)
    }

    @ViewBuilder
    func questionView(_ question: StorySectionQuestion) -> some View {
        let hasSubmitted = submittedQuestions.contains(question.id)
        let selectedIdx = selectedAnswers[question.id]

        VStack(alignment: .leading, spacing: 12) {
            SelectableTextView(
                text: question.questionText,
                language: viewModel.selectedStory?.language ?? "en",
                onTextSelected: { text in
                    selectedText = text
                    translationSentence = TextUtils.extractSentence(
                        from: question.questionText, containing: text)
                    showTranslationPopup = true
                }
            )
            .frame(maxWidth: .infinity, alignment: .leading)

            ForEach(Array(question.options.enumerated()), id: \.offset) { idx, option in
                optionRow(
                    question: question, idx: idx, option: option, hasSubmitted: hasSubmitted,
                    selectedIdx: selectedIdx
                )
            }

            if !hasSubmitted {
                submitButton(questionId: question.id, selectedIdx: selectedIdx)
            } else if let explanation = question.explanation, !explanation.isEmpty {
                explanationView(explanation: explanation, question: question)
            }
        }
        .appInnerCard()
        .overlay(
            RoundedRectangle(cornerRadius: AppTheme.CornerRadius.innerCard)
                .stroke(
                    hasSubmitted
                        ? (selectedIdx == question.correctAnswerIndex
                            ? AppTheme.Colors.successGreen.opacity(0.3)
                            : AppTheme.Colors.errorRed.opacity(0.3))
                        : AppTheme.Colors.borderGray,
                    lineWidth: 1
                )
        )
        .padding(.horizontal)
    }
}
