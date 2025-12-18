import SwiftUI

struct StoryDetailView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel = StoryViewModel()
    let storyId: Int
    @State private var showingSnippet: Snippet? = nil
    @State private var selectedAnswers: [Int: Int] = [:]
    @State private var submittedQuestions: Set<Int> = [] // QuestionID: OptionIndex

    @StateObject private var ttsManager = TTSSynthesizerManager.shared

    var body: some View {
        ZStack {
            VStack(spacing: 0) {
                if let story = viewModel.selectedStory {
                    // Header
                    VStack(alignment: .leading, spacing: 15) {
                        HStack {
                            Image(systemName: "book")
                            Text(story.title)
                                .font(.title3)
                                .fontWeight(.bold)
                        }

                        HStack(spacing: 10) {
                            Button(action: { viewModel.mode = .section }) {
                                Label("Section", systemImage: "list.bullet.indent")
                                    .padding(.horizontal, 12)
                                    .padding(.vertical, 8)
                                    .background(viewModel.mode == .section ? Color.blue : Color.blue.opacity(0.1))
                                    .foregroundColor(viewModel.mode == .section ? .white : .blue)
                                    .cornerRadius(8)
                            }

                            HStack(spacing: 8) {
                                Button(action: { viewModel.mode = .reading }) {
                                    Label("Reading", systemImage: "text.bubble")
                                        .padding(.horizontal, 12)
                                        .padding(.vertical, 8)
                                        .background(viewModel.mode == .reading ? Color.blue : Color.blue.opacity(0.1))
                                        .foregroundColor(viewModel.mode == .reading ? .white : .blue)
                                        .cornerRadius(8)
                                }

                                if viewModel.mode == .reading {
                                    TTSButton(text: viewModel.fullStoryContent, language: story.language)
                                        .padding(8)
                                        .background(Color.blue.opacity(0.1))
                                        .cornerRadius(8)
                                }
                            }
                        }
                    }
                    .padding()
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(Color(.systemBackground))
                    .shadow(color: Color.black.opacity(0.05), radius: 5, x: 0, y: 2)

                    if viewModel.mode == .section {
                        // Pagination for Section mode
                        HStack {
                            Button(action: { viewModel.previousPage() }) {
                                Image(systemName: "chevron.left.2")
                                    .padding(8)
                                    .background(Color.gray.opacity(0.1))
                                    .cornerRadius(6)
                            }
                            .disabled(viewModel.currentSectionIndex == 0)

                            Button(action: { viewModel.previousPage() }) {
                                Image(systemName: "chevron.left")
                                    .padding(8)
                                    .background(Color.gray.opacity(0.1))
                                    .cornerRadius(6)
                            }
                            .disabled(viewModel.currentSectionIndex == 0)

                            Spacer()
                            Text("\(viewModel.currentSectionIndex + 1) / \(story.sections.count)")
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                            Spacer()

                            Button(action: { viewModel.nextPage() }) {
                                Image(systemName: "chevron.right")
                                    .padding(8)
                                    .background(Color.gray.opacity(0.1))
                                    .cornerRadius(6)
                            }
                            .disabled(viewModel.currentSectionIndex == story.sections.count - 1)

                            Button(action: { viewModel.nextPage() }) {
                                Image(systemName: "chevron.right.2")
                                    .padding(8)
                                    .background(Color.gray.opacity(0.1))
                                    .cornerRadius(6)
                            }
                            .disabled(viewModel.currentSectionIndex == story.sections.count - 1)

                            BadgeView(text: "A1", color: .blue)
                        }
                        .padding()
                    }

                    ScrollView {
                        ScrollViewReader { proxy in
                            VStack(alignment: .leading, spacing: 20) {
                                if let error = ttsManager.errorMessage {
                                    Text(error)
                                        .font(.caption)
                                        .foregroundColor(.red)
                                        .padding()
                                        .background(Color.red.opacity(0.1))
                                        .cornerRadius(8)
                                        .padding(.horizontal)
                                }

                                if viewModel.mode == .section, let section = viewModel.currentSection {
                                    sectionContent(section)

                                    if !section.questions.isEmpty {
                                        Divider().padding(.vertical)
                                        Text("Comprehension Questions")
                                            .font(.headline)
                                            .padding(.horizontal)

                                        ForEach(section.questions) { question in
                                            questionView(question)
                                        }
                                    }
                                } else if viewModel.mode == .reading {
                                    ForEach(story.sections, id: \.id) { section in
                                        Text(highlightSnippets(in: section.content))
                                            .padding()
                                    }
                                }
                                Color.clear.frame(height: 1).id("bottom")
                            }
                                    .onChange(of: viewModel.currentSectionIndex) { _, _ in
            selectedAnswers.removeAll()
            submittedQuestions.removeAll()
        }
                        .onChange(of: submittedQuestions) { old, val in
                                if !val.isEmpty {
                                    withAnimation {
                                        proxy.scrollTo("bottom", anchor: .bottom)
                                    }
                                }
                            }
                        }
                    }
                } else if viewModel.isLoading {
                    ProgressView("Loading Story...")
                } else {
                    Text("Select a story to begin")
                        .foregroundColor(.secondary)
                }
            }

            // Snippet Popup
            if let snippet = showingSnippet {
                Color.black.opacity(0.3)
                    .ignoresSafeArea()
                    .onTapGesture { showingSnippet = nil }

                SnippetDetailView(snippet: snippet) {
                    showingSnippet = nil
                }
                .transition(.scale.combined(with: .opacity))
                .zIndex(1)
            }
        }
        .onAppear {
            viewModel.getStory(id: storyId)
        }
        .environment(\.openURL, OpenURLAction { url in
            if url.scheme == "snippet", let host = url.host, let id = Int(host) {
                if let snippet = viewModel.snippets.first(where: { $0.id == id }) {
                    showingSnippet = snippet
                    return .handled
                }
            }
            return .systemAction
        })
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
    }

    @ViewBuilder
    private func sectionContent(_ section: StorySectionWithQuestions) -> some View {
        VStack(alignment: .trailing) {
            TTSButton(text: section.content, language: viewModel.selectedStory?.language ?? "en")

            VStack(alignment: .leading) {
                let highlighted = highlightSnippets(in: section.content)
                Text(highlighted)
                    .lineSpacing(6)
                    .font(.body)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(12)
        .padding(.horizontal)
    }

    private func highlightSnippets(in text: String) -> AttributedString {
        var attrStr = AttributedString(text)
        let sortedSnippets = viewModel.snippets.sorted { $0.originalText.count > $1.originalText.count }

        for snippet in sortedSnippets {
            var searchRange = attrStr.startIndex..<attrStr.endIndex
            while let range = attrStr[searchRange].range(of: snippet.originalText, options: .caseInsensitive) {
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
    private func optionRow(question: StorySectionQuestion, idx: Int, option: String, hasSubmitted: Bool, selectedIdx: Int?) -> some View {
        let isCorrect = idx == question.correctAnswerIndex
        let isSelected = selectedIdx == idx

        Button(action: {
            if !hasSubmitted {
                selectedAnswers[question.id] = idx
            }
        }) {
            HStack {
                if hasSubmitted {
                    Image(systemName: isCorrect ? "checkmark.circle.fill" : (isSelected ? "xmark.circle.fill" : "circle"))
                        .foregroundColor(isCorrect ? .green : (isSelected ? .red : .gray))
                } else {
                    Circle()
                        .stroke(isSelected ? Color.blue : Color.gray, lineWidth: 1)
                        .frame(width: 18, height: 18)
                        .overlay(Circle().fill(isSelected ? Color.blue : Color.clear).padding(4))
                }

                Text(option)
                    .font(.subheadline)
                    .foregroundColor(hasSubmitted ? (isCorrect ? .green : (isSelected ? .red : .primary)) : .primary)
                Spacer()
            }
            .padding(10)
            .background(
                hasSubmitted ?
                (isCorrect ? Color.green.opacity(0.1) : (isSelected ? Color.red.opacity(0.1) : Color.gray.opacity(0.05))) :
                (isSelected ? Color.blue.opacity(0.1) : Color.gray.opacity(0.05))
            )
            .cornerRadius(8)
        }
        .disabled(hasSubmitted)
    }

    @ViewBuilder
    private func questionView(_ question: StorySectionQuestion) -> some View {
        let hasSubmitted = submittedQuestions.contains(question.id)
        let selectedIdx = selectedAnswers[question.id]

        VStack(alignment: .leading, spacing: 12) {
            Text(question.questionText)
                .font(.subheadline)
                .fontWeight(.medium)

            ForEach(Array(question.options.enumerated()), id: \.offset) { idx, option in
                optionRow(question: question, idx: idx, option: option, hasSubmitted: hasSubmitted, selectedIdx: selectedIdx)
            }

            if !hasSubmitted {
                Button(action: {
                    submittedQuestions.insert(question.id)
                }) {
                    Text("Submit Answer")
                        .font(.subheadline)
                        .fontWeight(.bold)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                        .background(selectedIdx == nil ? Color.blue.opacity(0.3) : Color.blue)
                        .foregroundColor(.white)
                        .cornerRadius(8)
                }
                .disabled(selectedIdx == nil)
                .padding(.top, 4)
            } else if let explanation = question.explanation, !explanation.isEmpty {
                VStack(alignment: .leading, spacing: 8) {
                    Divider()
                    Text("Explanation")
                        .font(.caption)
                        .fontWeight(.bold)
                        .foregroundColor(.secondary)
                    Text(explanation)
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
                .padding(.top, 4)
            }
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(12)
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .stroke(hasSubmitted ? (selectedIdx == question.correctAnswerIndex ? Color.green.opacity(0.3) : Color.red.opacity(0.3)) : Color.gray.opacity(0.1), lineWidth: 1)
        )
        .padding(.horizontal)
    }
}
