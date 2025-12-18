import SwiftUI

struct SnippetListView: View {
    @Environment(\.dismiss) private var dismiss
    @EnvironmentObject var authViewModel: AuthenticationViewModel
    @StateObject private var viewModel = VocabularyViewModel()
    @StateObject private var settingsViewModel = SettingsViewModel()
    @State private var showAddSnippet = false
    @State private var showEditSnippet = false
    @State private var showDeleteConfirmation = false
    @State private var selectedSnippet: Snippet? = nil

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
                        BadgeView(
                            text: "\(viewModel.snippets.count)", color: AppTheme.Colors.primaryBlue)
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
                    .background(AppTheme.Colors.primaryBlue)
                    .foregroundColor(.white)
                    .cornerRadius(AppTheme.CornerRadius.button)
                }
                .padding(.horizontal)

                // Search Section
                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField("Search snippets...", text: $viewModel.searchQuery)
                }
                .padding(12)
                .background(AppTheme.Colors.secondaryBackground)
                .cornerRadius(AppTheme.CornerRadius.button)
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
                            SnippetCard(
                                snippet: snippet,
                                onEdit: {
                                    selectedSnippet = snippet
                                    showEditSnippet = true
                                },
                                onDelete: {
                                    selectedSnippet = snippet
                                    showDeleteConfirmation = true
                                })
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
            let language = authViewModel.user?.preferredLanguage
            settingsViewModel.fetchLevels(language: language)
        }
        .sheet(isPresented: $showAddSnippet) {
            AddSnippetView(viewModel: viewModel, isPresented: $showAddSnippet)
        }
        .sheet(isPresented: $showEditSnippet) {
            if let snippet = selectedSnippet {
                EditSnippetView(
                    viewModel: viewModel, snippet: snippet, isPresented: $showEditSnippet)
            }
        }
        .alert("Delete Snippet", isPresented: $showDeleteConfirmation) {
            Button("Cancel", role: .cancel) {}
            Button("Delete", role: .destructive) {
                if let snippet = selectedSnippet {
                    viewModel.deleteSnippet(id: snippet.id) { _ in }
                }
            }
        } message: {
            Text("Are you sure you want to delete this snippet? This action cannot be undone.")
        }
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
                    .foregroundColor(AppTheme.Colors.primaryBlue)
                }
            }
        }
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
                    let story = viewModel.stories.first(where: { $0.id == selectedId })
                {
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
            .background(AppTheme.Colors.cardBackground)
            .cornerRadius(AppTheme.CornerRadius.button)
            .overlay(
                RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(
                    AppTheme.Colors.borderGray, lineWidth: 1))
        }
    }

    @ViewBuilder
    private func levelFilterPicker() -> some View {
        Menu {
            Button("All levels") {
                viewModel.selectedLevel = nil
            }
            ForEach(settingsViewModel.availableLevels, id: \.self) { level in
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
            .background(AppTheme.Colors.cardBackground)
            .cornerRadius(AppTheme.CornerRadius.button)
            .overlay(
                RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(
                    AppTheme.Colors.borderGray, lineWidth: 1))
        }
    }

    @ViewBuilder
    private func sourceLanguageFilterPicker() -> some View {
        Menu {
            Button("All languages") {
                viewModel.selectedSourceLang = nil
            }
            ForEach(
                [
                    Language.italian, Language.spanish, Language.french, Language.german,
                    Language.english,
                ], id: \.self
            ) { lang in
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
            .background(AppTheme.Colors.cardBackground)
            .cornerRadius(AppTheme.CornerRadius.button)
            .overlay(
                RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(
                    AppTheme.Colors.borderGray, lineWidth: 1))
        }
    }
}

struct SnippetCard: View {
    let snippet: Snippet
    let onEdit: () -> Void
    let onDelete: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 15) {
            Text(snippet.originalText)
                .font(.headline)
                .fontWeight(.bold)

            Text(snippet.translatedText)
                .font(.subheadline)
                .foregroundColor(AppTheme.Colors.primaryBlue)

            if let context = snippet.context {
                HStack(alignment: .top, spacing: 8) {
                    Image(systemName: "character.bubble")  // Translation icon replacement
                        .font(.caption)
                        .foregroundColor(AppTheme.Colors.primaryBlue)
                    Text("\"\(context)\"")
                        .font(.caption)
                        .italic()
                        .foregroundColor(.secondary)
                }
            }

            HStack(spacing: 10) {
                if let lang = snippet.sourceLanguage, let target = snippet.targetLanguage {
                    BadgeView(
                        text: "\(lang.uppercased()) -> \(target.uppercased())",
                        color: AppTheme.Colors.accentIndigo)
                }

                if let level = snippet.difficultyLevel {
                    BadgeView(text: level, color: AppTheme.Colors.primaryBlue)
                }

                if snippet.questionId != nil {
                    Button(action: {}) {
                        HStack(spacing: 4) {
                            Image(systemName: "arrow.up.right.square")
                            Text("View Question")
                        }
                        .font(.caption)
                        .foregroundColor(AppTheme.Colors.primaryBlue)
                    }
                }

                Spacer()

                HStack(spacing: 12) {
                    Button(action: onEdit) {
                        Image(systemName: "square.and.pencil")
                            .foregroundColor(AppTheme.Colors.primaryBlue)
                            .padding(6)
                            .background(Color.blue.opacity(0.1))
                            .cornerRadius(6)
                    }
                    Button(action: onDelete) {
                        Image(systemName: "trash")
                            .foregroundColor(.red)
                            .padding(6)
                            .background(Color.red.opacity(0.1))
                            .cornerRadius(6)
                    }
                }
            }
        }
        .appCard()
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
        !originalText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            && !translatedText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
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
                            .background(AppTheme.Colors.secondaryBackground)
                            .cornerRadius(AppTheme.CornerRadius.button)
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
                            .background(AppTheme.Colors.secondaryBackground)
                            .cornerRadius(AppTheme.CornerRadius.button)
                    }

                    // Source Language
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Source Language")
                            .font(.headline)
                        Menu {
                            ForEach(
                                [
                                    Language.italian, Language.spanish, Language.french,
                                    Language.german, Language.english,
                                ], id: \.self
                            ) { lang in
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
                            .overlay(
                                RoundedRectangle(cornerRadius: 10).stroke(
                                    Color.gray.opacity(0.2), lineWidth: 1))
                        }
                    }

                    // Target Language
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Target Language")
                            .font(.headline)
                        Menu {
                            ForEach(
                                [
                                    Language.italian, Language.spanish, Language.french,
                                    Language.german, Language.english,
                                ], id: \.self
                            ) { lang in
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
                            .overlay(
                                RoundedRectangle(cornerRadius: 10).stroke(
                                    Color.gray.opacity(0.2), lineWidth: 1))
                        }
                    }

                    // Context/Notes
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Context/Notes")
                            .font(.headline)
                        TextEditor(text: $context)
                            .frame(minHeight: 100)
                            .padding(8)
                            .background(AppTheme.Colors.secondaryBackground)
                            .cornerRadius(AppTheme.CornerRadius.button)
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
                                .foregroundColor(AppTheme.Colors.primaryBlue)
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
                        .background(
                            isFormValid && !isSubmitting
                                ? AppTheme.Colors.primaryBlue : Color.gray.opacity(0.3)
                        )
                        .foregroundColor(.white)
                        .cornerRadius(AppTheme.CornerRadius.button)
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
            context: context.isEmpty
                ? nil : context.trimmingCharacters(in: .whitespacesAndNewlines),
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

struct EditSnippetView: View {
    @ObservedObject var viewModel: VocabularyViewModel
    let snippet: Snippet
    @Binding var isPresented: Bool

    @State private var originalText = ""
    @State private var translatedText = ""
    @State private var sourceLanguage = Language.italian
    @State private var targetLanguage = Language.english
    @State private var context = ""
    @State private var isSubmitting = false
    @State private var errorMessage: String? = nil

    var body: some View {
        NavigationView {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    if let errorMessage = errorMessage {
                        Text(errorMessage)
                            .foregroundColor(.red)
                            .font(.caption)
                            .padding()
                            .background(Color.red.opacity(0.1))
                            .cornerRadius(8)
                    }

                    VStack(alignment: .leading, spacing: 8) {
                        Text("Original Text").font(.subheadline).fontWeight(.medium)
                        TextField("Enter word or phrase", text: $originalText)
                            .padding(12)
                            .background(Color(.secondarySystemBackground))
                            .cornerRadius(8)
                    }

                    VStack(alignment: .leading, spacing: 8) {
                        Text("Translation").font(.subheadline).fontWeight(.medium)
                        TextField("Enter translation", text: $translatedText)
                            .padding(12)
                            .background(Color(.secondarySystemBackground))
                            .cornerRadius(8)
                    }

                    VStack(alignment: .leading, spacing: 8) {
                        Text("Source Language").font(.subheadline).fontWeight(.medium)
                        Picker("Source Language", selection: $sourceLanguage) {
                            Text("Italian").tag(Language.italian)
                            Text("Spanish").tag(Language.spanish)
                            Text("French").tag(Language.french)
                            Text("German").tag(Language.german)
                            Text("English").tag(Language.english)
                        }
                        .pickerStyle(.menu)
                        .padding(12)
                        .background(Color(.secondarySystemBackground))
                        .cornerRadius(8)
                    }

                    VStack(alignment: .leading, spacing: 8) {
                        Text("Target Language").font(.subheadline).fontWeight(.medium)
                        Picker("Target Language", selection: $targetLanguage) {
                            Text("Italian").tag(Language.italian)
                            Text("Spanish").tag(Language.spanish)
                            Text("French").tag(Language.french)
                            Text("German").tag(Language.german)
                            Text("English").tag(Language.english)
                        }
                        .pickerStyle(.menu)
                        .padding(12)
                        .background(Color(.secondarySystemBackground))
                        .cornerRadius(8)
                    }

                    VStack(alignment: .leading, spacing: 8) {
                        Text("Context (Optional)").font(.subheadline).fontWeight(.medium)
                        TextEditor(text: $context)
                            .frame(minHeight: 100)
                            .padding(8)
                            .background(Color(.secondarySystemBackground))
                            .cornerRadius(8)
                    }

                    Button(action: submitEdit) {
                        Text(isSubmitting ? "Saving..." : "Save Changes")
                            .font(.headline)
                            .foregroundColor(.white)
                            .frame(maxWidth: .infinity)
                            .padding()
                            .background(canSubmit ? Color.blue : Color.gray)
                            .cornerRadius(10)
                    }
                    .disabled(!canSubmit || isSubmitting)
                }
                .padding()
            }
            .navigationTitle("Edit Snippet")
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
        .onAppear {
            originalText = snippet.originalText
            translatedText = snippet.translatedText
            sourceLanguage = Language(rawValue: snippet.sourceLanguage ?? "italian") ?? .italian
            targetLanguage = Language(rawValue: snippet.targetLanguage ?? "english") ?? .english
            context = snippet.context ?? ""
        }
    }

    private var canSubmit: Bool {
        !originalText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            && !translatedText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            && sourceLanguage != targetLanguage
    }

    private func submitEdit() {
        errorMessage = nil
        isSubmitting = true

        let request = UpdateSnippetRequest(
            originalText: originalText.trimmingCharacters(in: .whitespacesAndNewlines),
            translatedText: translatedText.trimmingCharacters(in: .whitespacesAndNewlines),
            sourceLanguage: sourceLanguage.rawValue,
            targetLanguage: targetLanguage.rawValue,
            context: context.isEmpty ? nil : context.trimmingCharacters(in: .whitespacesAndNewlines)
        )

        viewModel.updateSnippet(id: snippet.id, request: request) { result in
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
