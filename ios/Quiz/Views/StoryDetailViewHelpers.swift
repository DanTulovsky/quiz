import SwiftUI

extension StoryDetailView {
    func extractSentence(from text: String, containing selectedText: String) -> String? {
        guard let range = text.range(of: selectedText, options: .caseInsensitive) else {
            return nil
        }

        let startIndex = text.startIndex
        let endIndex = text.endIndex

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
}
