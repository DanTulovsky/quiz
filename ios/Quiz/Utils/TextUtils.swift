import Foundation

enum TextUtils {
    static func extractSentence(from text: String, containing selectedText: String) -> String? {
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
}
