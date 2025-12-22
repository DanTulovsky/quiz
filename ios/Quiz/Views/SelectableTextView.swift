import SwiftUI
import UIKit

private class SelectableSizingTextView: UITextView {
    var onTouchEnd: (() -> Void)?

    override var intrinsicContentSize: CGSize {
        guard !text.isEmpty, attributedText != nil else {
            return CGSize(width: UIView.noIntrinsicMetric, height: 0)
        }

        let width = bounds.width > 0 ? bounds.width : UIScreen.main.bounds.width - 64

        if let textLayoutManager = textLayoutManager,
           let textContentManager = textLayoutManager.textContentManager,
           let textContainer = textLayoutManager.textContainer {
            let containerSize = CGSize(width: width, height: .greatestFiniteMagnitude)
            textContainer.size = containerSize

            textLayoutManager.textViewportLayoutController.layoutViewport()

            var totalHeight: CGFloat = 0
            let documentRange = textContentManager.documentRange
            textLayoutManager.enumerateTextLayoutFragments(from: documentRange.location) { fragment in
                totalHeight = max(totalHeight, fragment.layoutFragmentFrame.maxY)
                return true
            }

            let height = ceil(totalHeight) + textContainerInset.top + textContainerInset.bottom
            return CGSize(width: UIView.noIntrinsicMetric, height: height)
        } else {
            let size = CGSize(width: width, height: .greatestFiniteMagnitude)
            let calculatedSize = sizeThatFits(size)
            return CGSize(width: UIView.noIntrinsicMetric, height: calculatedSize.height)
        }
    }

    override func layoutSubviews() {
        super.layoutSubviews()
        invalidateIntrinsicContentSize()
    }

    override func touchesEnded(_ touches: Set<UITouch>, with event: UIEvent?) {
        super.touchesEnded(touches, with: event)
        onTouchEnd?()
    }
}

struct SelectableTextView: UIViewRepresentable, Equatable {
    let text: String
    let language: String
    let onTextSelected: (String) -> Void
    let highlightedSnippets: [Snippet]?
    let textColor: UIColor?
    let onSnippetTapped: ((Snippet) -> Void)?
    @Environment(\.fontSizeMultiplier) var fontSizeMultiplier

    init(
        text: String, language: String, onTextSelected: @escaping (String) -> Void,
        highlightedSnippets: [Snippet]? = nil, textColor: UIColor? = nil,
        onSnippetTapped: ((Snippet) -> Void)? = nil
    ) {
        self.text = text
        self.language = language
        self.onTextSelected = onTextSelected
        self.highlightedSnippets = highlightedSnippets
        self.textColor = textColor
        self.onSnippetTapped = onSnippetTapped
    }

    func makeUIView(context: Context) -> UITextView {
        let textView = SelectableSizingTextView()
        textView.delegate = context.coordinator
        textView.isEditable = false
        textView.isSelectable = true
        textView.backgroundColor = .clear
        textView.textContainerInset = .zero
        textView.textContainer.lineFragmentPadding = 0
        textView.textContainer.widthTracksTextView = true
        textView.textContainer.heightTracksTextView = false
        textView.allowsEditingTextAttributes = false
        textView.isScrollEnabled = false
        textView.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        textView.setContentCompressionResistancePriority(.required, for: .vertical)

        textView.isUserInteractionEnabled = true
        textView.linkTextAttributes = [:]

        let coordinator = context.coordinator
        coordinator.textView = textView
        coordinator.onTextSelected = onTextSelected
        coordinator.onSnippetTapped = onSnippetTapped
        coordinator.highlightedSnippets = highlightedSnippets
        updateTextView(textView, snippets: highlightedSnippets)

        // Set up touch end handler
        textView.onTouchEnd = { [weak coordinator] in
            coordinator?.handleTouchEnd()
        }

        textView.layoutIfNeeded()
        return textView
    }

    func updateUIView(_ uiView: UITextView, context: Context) {
        uiView.linkTextAttributes = [:]

        context.coordinator.highlightedSnippets = highlightedSnippets
        context.coordinator.onSnippetTapped = onSnippetTapped
        updateTextView(uiView, snippets: highlightedSnippets)
        DispatchQueue.main.async {
            uiView.layoutIfNeeded()
            uiView.invalidateIntrinsicContentSize()
        }
    }

    private func updateTextView(_ textView: UITextView, snippets: [Snippet]?) {
        guard !text.isEmpty else {
            textView.attributedText = nil
            textView.text = ""
            return
        }

        let baseFont = UIFont.preferredFont(forTextStyle: .body)
        let scaledFontSize = baseFont.pointSize * fontSizeMultiplier
        let scaledFont = UIFont(descriptor: baseFont.fontDescriptor, size: scaledFontSize)

        let attributedString = NSMutableAttributedString(string: text)
        attributedString.addAttribute(
            .font, value: scaledFont, range: NSRange(location: 0, length: text.count))

        // Apply text color - use provided color or default to label color
        let color = textColor ?? UIColor.label
        attributedString.addAttribute(
            .foregroundColor, value: color, range: NSRange(location: 0, length: text.count))

        // Apply snippet highlighting if available
        if let snippets = snippets, !snippets.isEmpty {
            let sortedSnippets = snippets.sorted { $0.originalText.count > $1.originalText.count }
            let highlightColor = UIColor.systemBlue.withAlphaComponent(0.25)
            for snippet in sortedSnippets {
                let searchText = snippet.originalText.trimmingCharacters(
                    in: .whitespacesAndNewlines)
                guard !searchText.isEmpty else { continue }

                var searchRange = NSRange(location: 0, length: text.count)
                while searchRange.location < text.count {
                    let range = (text as NSString).range(
                        of: searchText, options: [.caseInsensitive, .diacriticInsensitive],
                        range: searchRange)
                    if range.location != NSNotFound {
                        if let url = URL(string: "snippet://\(snippet.id)") {
                            attributedString.addAttribute(.link, value: url, range: range)
                        }
                        attributedString.addAttribute(
                            .backgroundColor, value: highlightColor, range: range)
                        attributedString.addAttribute(
                            .underlineStyle, value: NSUnderlineStyle.patternDash.rawValue,
                            range: range)
                        attributedString.addAttribute(
                            .underlineColor, value: UIColor.systemBlue, range: range)
                        searchRange = NSRange(
                            location: range.location + range.length,
                            length: text.count - (range.location + range.length))
                    } else {
                        break
                    }
                }
            }
        }

        textView.attributedText = attributedString
        textView.invalidateIntrinsicContentSize()
    }

    static func == (lhs: SelectableTextView, rhs: SelectableTextView) -> Bool {
        // Compare all properties that affect the view
        // Also compare originalText to ensure content changes trigger updates
        let lhsSnippetTexts = lhs.highlightedSnippets?.map { "\($0.id):\($0.originalText)" } ?? []
        let rhsSnippetTexts = rhs.highlightedSnippets?.map { "\($0.id):\($0.originalText)" } ?? []
        return lhs.text == rhs.text && lhs.language == rhs.language
            && lhsSnippetTexts == rhsSnippetTexts
    }

    func makeCoordinator() -> Coordinator {
        let coordinator = Coordinator()
        coordinator.onTextSelected = onTextSelected
        return coordinator
    }

    class Coordinator: NSObject, UITextViewDelegate {
        var textView: UITextView?
        var onTextSelected: ((String) -> Void)?
        var onSnippetTapped: ((Snippet) -> Void)?
        var highlightedSnippets: [Snippet]?

        @available(
        iOS, deprecated: 17.0,
        message: "Use textView(_:shouldInteractWith:in:characterRange:) instead"
        )
        func textView(
            _ textView: UITextView, shouldInteractWith URL: URL, in characterRange: NSRange,
            interaction: UITextItemInteraction
        ) -> Bool {
            return handleSnippetURL(URL)
        }

        @available(iOS 17.0, *)
        func textView(
            _ textView: UITextView, shouldInteractWith URL: URL, in characterRange: NSRange
        ) -> Bool {
            return handleSnippetURL(URL)
        }

        private func handleSnippetURL(_ URL: URL) -> Bool {
            if URL.scheme == "snippet", let host = URL.host, let snippetId = Int(host) {
                if let snippet = highlightedSnippets?.first(where: { $0.id == snippetId }) {
                    onSnippetTapped?(snippet)
                    return false
                }
            }
            return true
        }

        func textViewDidChangeSelection(_ textView: UITextView) {
            // Selection tracking is now handled by touch end handler
        }

        func handleTouchEnd() {
            guard let textView = textView else {
                return
            }

            // Check if there's a valid selection
            guard let selectedRange = textView.selectedTextRange,
                  !selectedRange.isEmpty
            else {
                return
            }

            let selectedText = textView.text(in: selectedRange) ?? ""
            let trimmedText = selectedText.trimmingCharacters(in: .whitespacesAndNewlines)

            // Only trigger if selection is meaningful (more than 1 character)
            if trimmedText.count > 1 {
                // Clear selection to hide native menu and show our popup
                textView.selectedTextRange = nil
                onTextSelected?(trimmedText)
            }
        }
    }
}
