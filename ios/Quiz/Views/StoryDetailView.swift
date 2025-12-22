import SwiftUI

struct StoryDetailView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject var viewModel = StoryViewModel()
    let storyId: Int
    @State var showingSnippet: Snippet?
    @State var selectedAnswers: [Int: Int] = [:]
    @State var submittedQuestions: Set<Int> = [] // QuestionID: OptionIndex
    @State var selectedText: String?
    @State var showTranslationPopup = false
    @State var translationSentence: String?

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
                                .font(AppTheme.Typography.headingFont)
                        }

                        HStack(spacing: 10) {
                            Button(action: { viewModel.mode = .section }, label: {
                                Label("Section", systemImage: "list.bullet.indent")
                                    .padding(.horizontal, 12)
                                    .padding(.vertical, 8)
                                    .background(
                                        viewModel.mode == .section
                                            ? AppTheme.Colors.primaryBlue
                                            : AppTheme.Colors.primaryBlue.opacity(0.1)
                                    )
                                    .foregroundColor(viewModel.mode == .section ? .white : AppTheme.Colors.primaryBlue)
                                    .cornerRadius(AppTheme.CornerRadius.badge)
                            })

                            HStack(spacing: 8) {
                                Button(action: { viewModel.mode = .reading }, label: {
                                    Label("Reading", systemImage: "text.bubble")
                                        .padding(.horizontal, 12)
                                        .padding(.vertical, 8)
                                        .background(
                                            viewModel.mode == .reading
                                                ? AppTheme.Colors.primaryBlue
                                                : AppTheme.Colors.primaryBlue.opacity(0.1)
                                        )
                                        .foregroundColor(
                                            viewModel.mode == .reading ? .white : AppTheme.Colors.primaryBlue
                                        )
                                        .cornerRadius(AppTheme.CornerRadius.badge)
                                })

                                if viewModel.mode == .reading {
                                    TTSButton(text: viewModel.fullStoryContent, language: story.language)
                                        .padding(8)
                                        .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                                        .cornerRadius(AppTheme.CornerRadius.badge)
                                }
                            }
                        }
                    }
                    .appCard()

                    if viewModel.mode == .section {
                        // Pagination for Section mode
                        HStack {
                            Button(action: { viewModel.goToBeginning() }, label: {
                                Image(systemName: "chevron.left.2")
                                    .padding(8)
                                    .background(AppTheme.Colors.secondaryBackground)
                                    .cornerRadius(6)
                            })
                            .disabled(viewModel.currentSectionIndex == 0)

                            Button(action: { viewModel.previousPage() }, label: {
                                Image(systemName: "chevron.left")
                                    .padding(8)
                                    .background(AppTheme.Colors.secondaryBackground)
                                    .cornerRadius(6)
                            })
                            .disabled(viewModel.currentSectionIndex == 0)

                            Spacer()
                            Text("\(viewModel.currentSectionIndex + 1) / \(story.sections.count)")
                                .font(AppTheme.Typography.subheadlineFont)
                                .foregroundColor(AppTheme.Colors.secondaryText)
                            Spacer()

                            Button(action: { viewModel.nextPage() }, label: {
                                Image(systemName: "chevron.right")
                                    .padding(8)
                                    .background(AppTheme.Colors.secondaryBackground)
                                    .cornerRadius(6)
                            })
                            .disabled(viewModel.currentSectionIndex == story.sections.count - 1)

                            Button(action: { viewModel.goToEnd() }, label: {
                                Image(systemName: "chevron.right.2")
                                    .padding(8)
                                    .background(AppTheme.Colors.secondaryBackground)
                                    .cornerRadius(6)
                            })
                            .disabled(viewModel.currentSectionIndex == story.sections.count - 1)

                            BadgeView(text: "A1", color: AppTheme.Colors.primaryBlue)
                        }
                        .padding()
                    }

                    ScrollView {
                        ScrollViewReader { _ in
                            VStack(alignment: .leading, spacing: 20) {
                                if let error = ttsManager.errorMessage {
                                    Text(error)
                                        .font(AppTheme.Typography.captionFont)
                                        .foregroundColor(AppTheme.Colors.errorRed)
                                        .padding()
                                        .background(AppTheme.Colors.errorRed.opacity(0.1))
                                        .cornerRadius(AppTheme.CornerRadius.badge)
                                        .padding(.horizontal)
                                }

                                if viewModel.mode == .section, let section = viewModel.currentSection {
                                    sectionContent(section)

                                    if !section.questions.isEmpty {
                                        Divider().padding(.vertical)
                                        Text("Comprehension Questions")
                                            .font(AppTheme.Typography.headingFont)
                                            .padding(.horizontal)

                                        ForEach(section.questions) { question in
                                            questionView(question)
                                        }
                                    }
                                } else if viewModel.mode == .reading {
                                    ForEach(story.sections, id: \.id) { section in
                                        SelectableTextView(
                                            text: section.content,
                                            language: story.language,
                                            onTextSelected: { text in
                                                selectedText = text
                    translationSentence = TextUtils.extractSentence(
                        from: section.content, containing: text
                    )
                                                showTranslationPopup = true
                                            },
                                            highlightedSnippets: viewModel.snippets,
                                            onSnippetTapped: { snippet in
                                                showingSnippet = snippet
                                            }
                                        )
                                        .frame(minHeight: 100)
                                        .padding()
                                    }
                                }
                                Color.clear.frame(height: 1).id("bottom")
                            }
                            .onChange(of: viewModel.currentSectionIndex) { _, _ in
                                selectedAnswers.removeAll()
                                submittedQuestions.removeAll()
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
        .sheet(isPresented: $showTranslationPopup) {
            if let text = selectedText, let story = viewModel.selectedStory {
                TranslationPopupView(
                    selectedText: text,
                    sourceLanguage: story.language,
                    questionId: nil,
                    sectionId: viewModel.currentSection?.id,
                    storyId: story.id,
                    sentence: translationSentence,
                    onClose: {
                        showTranslationPopup = false
                        selectedText = nil
                        translationSentence = nil
                    },
                    onSnippetSaved: { snippet in
                        // Optimistically add the snippet immediately for instant UI update
                        // Create a new array to ensure SwiftUI detects the change
                        if !viewModel.snippets.contains(where: { $0.id == snippet.id }) {
                            viewModel.snippets += [snippet]
                        }
                        // Then reload to ensure consistency with server
                        viewModel.loadSnippets(storyId: story.id)
                    }
                )
            }
        }
        .navigationBarTitleDisplayMode(.inline)
        .navigationBarBackButtonHidden(true)
        .toolbar {
            ToolbarItem(placement: .navigationBarLeading) {
                Button(action: { dismiss() }, label: {
                    HStack(spacing: 4) {
                        Image(systemName: "chevron.left")
                            .scaledFont(size: 17, weight: .semibold)
                        Text("Back")
                            .scaledFont(size: 17)
                    }
                    .foregroundColor(.blue)
                })
            }
        }
        .snippetDetailPopup(
            showingSnippet: $showingSnippet,
            onSnippetDeleted: { snippet in
                viewModel.snippets.removeAll { $0.id == snippet.id }
            }
        )
    }

}
