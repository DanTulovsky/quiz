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
        updateTextView(textView, context: context)
        textView.isEditable = false
        textView.isSelectable = true
        textView.backgroundColor = .clear
        textView.textContainerInset = .zero
        textView.textContainer.lineFragmentPadding = 0

        // Add long press gesture to detect selection
        let longPress = UILongPressGestureRecognizer(target: context.coordinator, action: #selector(Coordinator.handleLongPress(_:)))
        longPress.minimumPressDuration = 0.5
        textView.addGestureRecognizer(longPress)

        // Add tap gesture to check selection
        let tap = UITapGestureRecognizer(target: context.coordinator, action: #selector(Coordinator.handleTap(_:)))
        textView.addGestureRecognizer(tap)

        context.coordinator.textView = textView
        return textView
    }

    func updateUIView(_ uiView: UITextView, context: Context) {
        updateTextView(uiView, context: context)
    }

    private func updateTextView(_ textView: UITextView, context: Context) {
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
        Coordinator(onTextSelected: onTextSelected)
    }

    class Coordinator: NSObject {
        var textView: UITextView?
        let onTextSelected: (String) -> Void

        init(onTextSelected: @escaping (String) -> Void) {
            self.onTextSelected = onTextSelected
        }

        @objc func handleLongPress(_ gesture: UILongPressGestureRecognizer) {
            guard gesture.state == .began,
                  let textView = textView,
                  let selectedRange = textView.selectedTextRange,
                  !selectedRange.isEmpty else {
                return
            }

            let selectedText = textView.text(in: selectedRange) ?? ""
            if selectedText.trimmingCharacters(in: .whitespacesAndNewlines).count > 1 {
                onTextSelected(selectedText.trimmingCharacters(in: .whitespacesAndNewlines))
            }
        }

        @objc func handleTap(_ gesture: UITapGestureRecognizer) {
            // Small delay to allow selection to be set
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                guard let textView = self.textView,
                      let selectedRange = textView.selectedTextRange,
                      !selectedRange.isEmpty else {
                    return
                }

                let selectedText = textView.text(in: selectedRange) ?? ""
                if selectedText.trimmingCharacters(in: .whitespacesAndNewlines).count > 1 {
                    self.onTextSelected(selectedText.trimmingCharacters(in: .whitespacesAndNewlines))
                }
            }
        }
    }
}

