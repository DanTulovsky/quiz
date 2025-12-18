import SwiftUI

struct SnippetListView: View {
    @StateObject private var viewModel = VocabularyViewModel()
    @State private var showAddSnippet = false

    var body: some View {
        ScrollView {
            VStack(spacing: 20) {
                // Header Stats and Description
                VStack(alignment: .leading, spacing: 8) {
                    HStack {
                        Text("Snippets")
                            .font(.largeTitle)
                            .bold()
                        Spacer()
                        BadgeView(text: "\(viewModel.snippets.count)", color: .blue)
                    }
                    Text("Your saved words and phrases")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }
                .padding(.horizontal)

                // Add New Snippet Button
                Button(action: {
                    showAddSnippet = true
                }) {
                    HStack {
                        Image(systemName: "plus")
                        Text("Add New Snippet")
                    }
                    .font(.headline)
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.blue)
                    .foregroundColor(.white)
                    .cornerRadius(12)
                }
                .padding(.horizontal)

                // Search Section
                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField("Search snippets...", text: $viewModel.searchQuery)
                }
                .padding(12)
                .background(Color(.secondarySystemBackground))
                .cornerRadius(10)
                .padding(.horizontal)

                // Filters
                VStack(spacing: 12) {
                    storyFilterPicker()
                    levelFilterPicker()
                    sourceLanguageFilterPicker()
                }
                .padding(.horizontal)

                Divider().padding(.vertical, 10)

                // Snippets List
                VStack(spacing: 16) {
                    if viewModel.isLoading {
                        ProgressView()
                    } else if viewModel.filteredSnippets.isEmpty {
                        Text("No snippets found")
                            .foregroundColor(.secondary)
                            .padding()
                    } else {
                        ForEach(viewModel.filteredSnippets) { snippet in
                            SnippetCard(snippet: snippet)
                        }
                    }
                }
                .padding(.horizontal)
            }
            .padding(.vertical)
        }
        .onAppear {
            viewModel.fetchStories()
            viewModel.getSnippets()
        }
        .sheet(isPresented: $showAddSnippet) {
            AddSnippetView(viewModel: viewModel, isPresented: $showAddSnippet)
        }
        .navigationBarTitleDisplayMode(.inline)
    }

    @ViewBuilder
    private func storyFilterPicker() -> some View {
        Menu {
            Button("All stories") {
                viewModel.selectedStoryId = nil
            }
            ForEach(viewModel.stories) { story in
                Button(story.title) {
                    viewModel.selectedStoryId = story.id
                }
            }
        } label: {
            HStack {
                if let selectedId = viewModel.selectedStoryId,
                   let story = viewModel.stories.first(where: { $0.id == selectedId }) {
                    Text(story.title)
                        .foregroundColor(.primary)
                } else {
                    Text("Filter by story")
                        .foregroundColor(.secondary)
                }
                Spacer()
                Image(systemName: "chevron.up.chevron.down")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            .padding(12)
            .background(Color(.systemBackground))
            .cornerRadius(10)
            .overlay(RoundedRectangle(cornerRadius: 10).stroke(Color.gray.opacity(0.2), lineWidth: 1))
        }
    }

    @ViewBuilder
    private func levelFilterPicker() -> some View {
        Menu {
            Button("All levels") {
                viewModel.selectedLevel = nil
            }
            ForEach(["A1", "A2", "B1", "B2", "C1", "C2"], id: \.self) { level in
                Button(level) {
                    viewModel.selectedLevel = level
                }
            }
        } label: {
            HStack {
                Text(viewModel.selectedLevel ?? "Filter by level")
                    .foregroundColor(viewModel.selectedLevel == nil ? .secondary : .primary)
                Spacer()
                Image(systemName: "chevron.up.chevron.down")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            .padding(12)
            .background(Color(.systemBackground))
            .cornerRadius(10)
            .overlay(RoundedRectangle(cornerRadius: 10).stroke(Color.gray.opacity(0.2), lineWidth: 1))
        }
    }

    @ViewBuilder
    private func sourceLanguageFilterPicker() -> some View {
        Menu {
            Button("All languages") {
                viewModel.selectedSourceLang = nil
            }
            ForEach([Language.italian, Language.spanish, Language.french, Language.german, Language.english], id: \.self) { lang in
                Button(lang.rawValue.capitalized) {
                    viewModel.selectedSourceLang = lang
                }
            }
        } label: {
            HStack {
                if let lang = viewModel.selectedSourceLang {
                    Text(lang.rawValue.capitalized)
                        .foregroundColor(.primary)
                } else {
                    Text("Filter by source language")
                        .foregroundColor(.secondary)
                }
                Spacer()
                Image(systemName: "chevron.up.chevron.down")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            .padding(12)
            .background(Color(.systemBackground))
            .cornerRadius(10)
            .overlay(RoundedRectangle(cornerRadius: 10).stroke(Color.gray.opacity(0.2), lineWidth: 1))
        }
    }
}

struct SnippetCard: View {
    let snippet: Snippet

    var body: some View {
        VStack(alignment: .leading, spacing: 15) {
            Text(snippet.originalText)
                .font(.headline)
                .fontWeight(.bold)

            Text(snippet.translatedText)
                .font(.subheadline)
                .foregroundColor(.blue)

            if let context = snippet.context {
                HStack(alignment: .top, spacing: 8) {
                    Image(systemName: "character.bubble") // Translation icon replacement
                        .font(.caption)
                        .foregroundColor(.blue)
                    Text("\"\(context)\"")
                        .font(.caption)
                        .italic()
                        .foregroundColor(.secondary)
                }
            }

            HStack(spacing: 10) {
                if let lang = snippet.sourceLanguage, let target = snippet.targetLanguage {
                    BadgeView(text: "\(lang.uppercased()) -> \(target.uppercased())", color: .gray)
                }

                if let level = snippet.difficultyLevel {
                    BadgeView(text: level, color: .blue)
                }

                if snippet.questionId != nil {
                    Button(action: {}) {
                        HStack(spacing: 4) {
                            Image(systemName: "arrow.up.right.square")
                            Text("View Question")
                        }
                        .font(.caption)
                        .foregroundColor(.blue)
                    }
                }

                Spacer()

                HStack(spacing: 12) {
                    Button(action: {}) {
                        Image(systemName: "square.and.pencil")
                            .foregroundColor(.blue)
                            .padding(6)
                            .background(Color.blue.opacity(0.1))
                            .cornerRadius(6)
                    }
                    Button(action: {}) {
                        Image(systemName: "trash")
                            .foregroundColor(.red)
                            .padding(6)
                            .background(Color.red.opacity(0.1))
                            .cornerRadius(6)
                    }
                }
            }
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(16)
        .shadow(color: Color.black.opacity(0.05), radius: 8, x: 0, y: 4)
        .overlay(RoundedRectangle(cornerRadius: 16).stroke(Color.gray.opacity(0.1), lineWidth: 1))
    }
}

struct AddSnippetView: View {
    @ObservedObject var viewModel: VocabularyViewModel
    @Binding var isPresented: Bool

    @State private var originalText = ""
    @State private var translatedText = ""
    @State private var sourceLanguage = Language.italian
    @State private var targetLanguage = Language.english
    @State private var context = ""
    @State private var isSubmitting = false
    @State private var errorMessage: String?

    private var isFormValid: Bool {
        !originalText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty &&
        !translatedText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    var body: some View {
        NavigationView {
            ScrollView {
                VStack(spacing: 20) {
                    // Original Text
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            Text("Original Text")
                                .font(.headline)
                            Text("*")
                                .foregroundColor(.red)
                        }
                        TextField("Enter the original text...", text: $originalText)
                            .textFieldStyle(.plain)
                            .padding(12)
                            .background(Color(.secondarySystemBackground))
                            .cornerRadius(10)
                    }

                    // Translation
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            Text("Translation")
                                .font(.headline)
                            Text("*")
                                .foregroundColor(.red)
                        }
                        TextField("Enter the translation...", text: $translatedText)
                            .textFieldStyle(.plain)
                            .padding(12)
                            .background(Color(.secondarySystemBackground))
                            .cornerRadius(10)
                    }

                    // Source Language
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Source Language")
                            .font(.headline)
                        Menu {
                            ForEach([Language.italian, Language.spanish, Language.french, Language.german, Language.english], id: \.self) { lang in
                                Button(lang.rawValue.capitalized) {
                                    sourceLanguage = lang
                                }
                            }
                        } label: {
                            HStack {
                                Text(sourceLanguage.rawValue.capitalized)
                                Spacer()
                                Image(systemName: "chevron.up.chevron.down")
                                    .font(.caption)
                            }
                            .padding(12)
                            .background(Color(.systemBackground))
                            .cornerRadius(10)
                            .overlay(RoundedRectangle(cornerRadius: 10).stroke(Color.gray.opacity(0.2), lineWidth: 1))
                        }
                    }

                    // Target Language
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Target Language")
                            .font(.headline)
                        Menu {
                            ForEach([Language.italian, Language.spanish, Language.french, Language.german, Language.english], id: \.self) { lang in
                                Button(lang.rawValue.capitalized) {
                                    targetLanguage = lang
                                }
                            }
                        } label: {
                            HStack {
                                Text(targetLanguage.rawValue.capitalized)
                                Spacer()
                                Image(systemName: "chevron.up.chevron.down")
                                    .font(.caption)
                            }
                            .padding(12)
                            .background(Color(.systemBackground))
                            .cornerRadius(10)
                            .overlay(RoundedRectangle(cornerRadius: 10).stroke(Color.gray.opacity(0.2), lineWidth: 1))
                        }
                    }

                    // Context/Notes
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Context/Notes")
                            .font(.headline)
                        TextEditor(text: $context)
                            .frame(minHeight: 100)
                            .padding(8)
                            .background(Color(.secondarySystemBackground))
                            .cornerRadius(10)
                            .overlay(
                                Group {
                                    if context.isEmpty {
                                        Text("Add context or notes about this snippet...")
                                            .foregroundColor(.secondary)
                                            .padding(.horizontal, 12)
                                            .padding(.vertical, 16)
                                            .frame(maxWidth: .infinity, alignment: .topLeading)
                                            .allowsHitTesting(false)
                                    }
                                }
                            )
                    }

                    if let error = errorMessage {
                        Text(error)
                            .font(.caption)
                            .foregroundColor(.red)
                            .padding()
                            .frame(maxWidth: .infinity)
                            .background(Color.red.opacity(0.1))
                            .cornerRadius(8)
                    }

                    // Buttons
                    VStack(spacing: 12) {
                        Button(action: {
                            isPresented = false
                        }) {
                            Text("Cancel")
                                .font(.headline)
                                .frame(maxWidth: .infinity)
                                .padding()
                                .background(Color.blue.opacity(0.1))
                                .foregroundColor(.blue)
                                .cornerRadius(12)
                        }

                        Button(action: submitSnippet) {
                            if isSubmitting {
                                ProgressView()
                                    .frame(maxWidth: .infinity)
                                    .padding()
                            } else {
                                Text("Add Snippet")
                                    .font(.headline)
                                    .frame(maxWidth: .infinity)
                                    .padding()
                            }
                        }
                        .disabled(!isFormValid || isSubmitting)
                        .background(isFormValid && !isSubmitting ? Color.blue : Color.gray.opacity(0.3))
                        .foregroundColor(.white)
                        .cornerRadius(12)
                    }
                }
                .padding()
            }
            .navigationTitle("Add New Snippet")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: {
                        isPresented = false
                    }) {
                        Image(systemName: "xmark")
                            .foregroundColor(.primary)
                            .padding(8)
                            .background(Color.blue.opacity(0.1))
                            .clipShape(Circle())
                    }
                }
            }
        }
    }

    private func submitSnippet() {
        errorMessage = nil
        isSubmitting = true

        let request = CreateSnippetRequest(
            originalText: originalText.trimmingCharacters(in: .whitespacesAndNewlines),
            translatedText: translatedText.trimmingCharacters(in: .whitespacesAndNewlines),
            sourceLanguage: sourceLanguage.rawValue,
            targetLanguage: targetLanguage.rawValue,
            context: context.isEmpty ? nil : context.trimmingCharacters(in: .whitespacesAndNewlines),
            questionId: nil
        )

        viewModel.createSnippet(request: request) { result in
            isSubmitting = false
            switch result {
            case .success:
                isPresented = false
            case .failure(let error):
                errorMessage = error.localizedDescription
            }
        }
    }
}
