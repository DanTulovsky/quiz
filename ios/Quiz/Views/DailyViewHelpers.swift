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
}
