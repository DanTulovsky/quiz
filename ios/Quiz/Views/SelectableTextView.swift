import SwiftUI
import UIKit

struct SelectableTextView: UIViewRepresentable {
    let text: String
    let language: String
    let onTextSelected: (String) -> Void
    let highlightedSnippets: [Snippet]?

    init(text: String, language: String, onTextSelected: @escaping (String) -> Void, highlightedSnippets: [Snippet]? = nil) {
        self.text = text
        self.language = language
        self.onTextSelected = onTextSelected
        self.highlightedSnippets = highlightedSnippets
    }

    func makeUIView(context: Context) -> UITextView {
        let textView = UITextView()
        textView.delegate = context.coordinator
        updateTextView(textView)
        textView.isEditable = false
        textView.isSelectable = true
        textView.backgroundColor = .clear
        textView.textContainerInset = .zero
        textView.textContainer.lineFragmentPadding = 0
        textView.allowsEditingTextAttributes = false

        // Enable text selection
        textView.isUserInteractionEnabled = true

        context.coordinator.textView = textView
        context.coordinator.onTextSelected = onTextSelected
        return textView
    }

    func updateUIView(_ uiView: UITextView, context: Context) {
        updateTextView(uiView)
    }

    private func updateTextView(_ textView: UITextView) {
        let attributedString = NSMutableAttributedString(string: text)
        attributedString.addAttribute(.font, value: UIFont.preferredFont(forTextStyle: .body), range: NSRange(location: 0, length: text.count))

        // Apply snippet highlighting if available
        if let snippets = highlightedSnippets {
            let sortedSnippets = snippets.sorted { $0.originalText.count > $1.originalText.count }
            for snippet in sortedSnippets {
                let searchText = snippet.originalText
                var searchRange = NSRange(location: 0, length: text.count)
                while searchRange.location < text.count {
                    let range = (text as NSString).range(of: searchText, options: .caseInsensitive, range: searchRange)
                    if range.location != NSNotFound {
                        attributedString.addAttribute(.foregroundColor, value: UIColor.blue, range: range)
                        attributedString.addAttribute(.underlineStyle, value: NSUnderlineStyle.patternDash.rawValue, range: range)
                        searchRange = NSRange(location: range.location + range.length, length: text.count - (range.location + range.length))
                    } else {
                        break
                    }
                }
            }
        }

        textView.attributedText = attributedString
    }

    func makeCoordinator() -> Coordinator {
        let coordinator = Coordinator()
        coordinator.onTextSelected = onTextSelected
        return coordinator
    }

    class Coordinator: NSObject, UITextViewDelegate {
        var textView: UITextView?
        var onTextSelected: ((String) -> Void)?
        private var selectionTimer: Timer?

        func textViewDidChangeSelection(_ textView: UITextView) {
            // Cancel previous timer
            selectionTimer?.invalidate()

            // Check if there's a selection
            guard let selectedRange = textView.selectedTextRange,
                  !selectedRange.isEmpty else {
                return
            }

            let selectedText = textView.text(in: selectedRange) ?? ""
            let trimmedText = selectedText.trimmingCharacters(in: .whitespacesAndNewlines)

            // Only trigger if selection is meaningful (more than 1 character)
            if trimmedText.count > 1 {
                // Wait a bit for the selection menu to appear, then show our popup
                // This gives users a chance to use the native menu if they want
                selectionTimer = Timer.scheduledTimer(withTimeInterval: 0.8, repeats: false) { [weak self] _ in
                    guard let self = self,
                          let textView = self.textView,
                          let currentRange = textView.selectedTextRange,
                          !currentRange.isEmpty else {
                        return
                    }

                    let currentText = textView.text(in: currentRange) ?? ""
                    let currentTrimmed = currentText.trimmingCharacters(in: .whitespacesAndNewlines)

                    if currentTrimmed.count > 1 {
                        // Clear selection to hide native menu and show our popup
                        textView.selectedTextRange = nil
                        self.onTextSelected?(currentTrimmed)
                    }
                }
            }
        }
    }
}

