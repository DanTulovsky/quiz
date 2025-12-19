import SwiftUI

struct SnippetListViewWithSearch: View {
    let query: String

    var body: some View {
        SnippetListView(initialSearchQuery: query)
    }
}

struct SnippetListView: View {
    @Environment(\.dismiss) private var dismiss
    @EnvironmentObject var authViewModel: AuthenticationViewModel
    @StateObject private var viewModel = VocabularyViewModel()
    @StateObject private var settingsViewModel = SettingsViewModel()
    @State private var showAddSnippet = false
    @State private var showEditSnippet = false
    @State private var showDeleteConfirmation = false
    @State private var showSnippetDetail = false
    @State private var selectedSnippet: Snippet? = nil
    @State private var filtersExpanded = false

    let initialSearchQuery: String?

    init(initialSearchQuery: String? = nil) {
        self.initialSearchQuery = initialSearchQuery
    }

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

                // Search and Filters
                VStack(spacing: 0) {
                    Button(action: {
                        withAnimation {
                            filtersExpanded.toggle()
                        }
                    }) {
                        HStack {
                            Image(systemName: "line.3.horizontal.decrease.circle")
                                .foregroundColor(.secondary)
                            Text("Search and Filters")
                                .font(.subheadline)
                                .fontWeight(.medium)
                                .foregroundColor(.primary)

                            if hasActiveFilters {
                                Text(filterSummary)
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                                    .padding(.horizontal, 8)
                                    .padding(.vertical, 4)
                                    .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                                    .cornerRadius(8)
                            }

                            Spacer()

                            Image(systemName: filtersExpanded ? "chevron.up" : "chevron.down")
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
                    .buttonStyle(PlainButtonStyle())

                    if filtersExpanded {
                        VStack(spacing: 12) {
                            // Search Section
                            HStack {
                                Image(systemName: "magnifyingglass")
                                    .foregroundColor(.secondary)
                                TextField("Search snippets...", text: $viewModel.searchQuery)
                            }
                            .padding(12)
                            .background(AppTheme.Colors.secondaryBackground)
                            .cornerRadius(AppTheme.CornerRadius.button)

                            storyFilterPicker()
                            levelFilterPicker()
                            sourceLanguageFilterPicker()
                        }
                        .padding(.top, 12)
                    }
                }
                .padding(.horizontal)

                Divider().padding(.vertical, 10)

                // Snippets List
                VStack(spacing: 16) {
                    if viewModel.isLoading {
                        ProgressView()
                    } else if viewModel.filteredSnippets.isEmpty {
                        EmptyStateView(
                            icon: "text.quote",
                            title: "No Snippets Found",
                            message: hasActiveFilters
                                ? "Try adjusting your filters to see more results."
                                : "Start building your vocabulary by adding your first snippet!",
                            actionTitle: hasActiveFilters ? nil : "Add Snippet",
                            action: hasActiveFilters ? nil : { showAddSnippet = true }
                        )
                    } else {
                        ForEach(viewModel.filteredSnippets) { snippet in
                            SnippetCard(
                                snippet: snippet,
                                onTap: {
                                    selectedSnippet = snippet
                                    showSnippetDetail = true
                                })
                        }
                    }
                }
                .padding(.horizontal)
            }
            .padding(.vertical)
        }
        .onAppear {
            if let query = initialSearchQuery {
                viewModel.searchQuery = query
            }
            viewModel.fetchStories()
            viewModel.fetchLanguages()
            viewModel.getSnippets()
            settingsViewModel.fetchLevels(language: nil)
        }
        .sheet(isPresented: $showAddSnippet) {
            AddSnippetView(viewModel: viewModel, isPresented: $showAddSnippet)
        }
        .sheet(isPresented: $showSnippetDetail) {
            if let snippet = selectedSnippet {
                SnippetDetailSheetView(
                    snippet: snippet,
                    viewModel: viewModel,
                    isPresented: $showSnippetDetail,
                    onEdit: {
                        showSnippetDetail = false
                        selectedSnippet = snippet
                        showEditSnippet = true
                    },
                    onDelete: {
                        showSnippetDetail = false
                        selectedSnippet = snippet
                        showDeleteConfirmation = true
                    })
            }
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

    private var hasActiveFilters: Bool {
        viewModel.selectedStoryId != nil || viewModel.selectedLevel != nil
            || viewModel.selectedSourceLang != nil
    }

    private var filterSummary: String {
        var parts: [String] = []
        if let selectedId = viewModel.selectedStoryId,
            let story = viewModel.stories.first(where: { $0.id == selectedId })
        {
            parts.append(story.title)
        }
        if let level = viewModel.selectedLevel {
            parts.append(level)
        }
        if let selectedLangCode = viewModel.selectedSourceLang,
            let lang = viewModel.availableLanguages.find(byCodeOrName: selectedLangCode)
        {
            parts.append(lang.name.capitalized)
        }
        return parts.joined(separator: ", ")
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
            ForEach(viewModel.availableLanguages) { lang in
                Button(lang.name.capitalized) {
                    viewModel.selectedSourceLang = lang.code
                }
            }
        } label: {
            HStack {
                if let selectedLangCode = viewModel.selectedSourceLang,
                    let lang = viewModel.availableLanguages.find(byCodeOrName: selectedLangCode)
                {
                    Text(lang.name.capitalized)
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
    let onTap: () -> Void

    var body: some View {
        Button(action: onTap) {
            HStack(alignment: .center, spacing: 12) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(snippet.originalText)
                        .font(.body)
                        .fontWeight(.medium)
                        .foregroundColor(.primary)
                        .lineLimit(2)
                        .multilineTextAlignment(.leading)

                    if let lang = snippet.sourceLanguage, let target = snippet.targetLanguage {
                        Text("\(lang.uppercased()) → \(target.uppercased())")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }

                Spacer()

                Image(systemName: "chevron.right")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(AppTheme.Colors.cardBackground)
            .cornerRadius(AppTheme.CornerRadius.card)
            .shadow(
                color: AppTheme.Shadow.card.color,
                radius: AppTheme.Shadow.card.radius,
                x: AppTheme.Shadow.card.x,
                y: AppTheme.Shadow.card.y
            )
            .overlay(
                RoundedRectangle(cornerRadius: AppTheme.CornerRadius.card)
                    .stroke(AppTheme.Colors.borderGray, lineWidth: 1)
            )
        }
        .buttonStyle(PlainButtonStyle())
    }
}

struct SnippetDetailSheetView: View {
    let snippet: Snippet
    @ObservedObject var viewModel: VocabularyViewModel
    @Binding var isPresented: Bool
    let onEdit: () -> Void
    let onDelete: () -> Void

    var body: some View {
        NavigationView {
            ScrollView {
                VStack(alignment: .leading, spacing: 24) {
                    // Original Text
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Original Text")
                            .font(.subheadline)
                            .fontWeight(.medium)
                            .foregroundColor(.secondary)
                        Text(snippet.originalText)
                            .font(.title2)
                            .fontWeight(.semibold)
                            .foregroundColor(.primary)
                    }

                    // Translation
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Translation")
                            .font(.subheadline)
                            .fontWeight(.medium)
                            .foregroundColor(.secondary)
                        Text(snippet.translatedText)
                            .font(.title3)
                            .foregroundColor(AppTheme.Colors.primaryBlue)
                    }

                    // Language and Level Info
                    HStack(spacing: 12) {
                        if let lang = snippet.sourceLanguage, let target = snippet.targetLanguage {
                            BadgeView(
                                text: "\(lang.uppercased()) → \(target.uppercased())",
                                color: AppTheme.Colors.accentIndigo)
                        }
                        if let level = snippet.difficultyLevel {
                            BadgeView(text: level, color: AppTheme.Colors.primaryBlue)
                        }
                    }

                    // Context
                    if let context = snippet.context, !context.isEmpty {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Context")
                                .font(.subheadline)
                                .fontWeight(.medium)
                                .foregroundColor(.secondary)
                            Text("\"\(context)\"")
                                .font(.body)
                                .foregroundColor(.primary)
                                .padding()
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(AppTheme.Colors.secondaryBackground)
                                .cornerRadius(AppTheme.CornerRadius.button)
                        }
                    }

                    // Action Buttons
                    VStack(spacing: 12) {
                        Button(action: onEdit) {
                            HStack {
                                Image(systemName: "square.and.pencil")
                                Text("Edit")
                            }
                            .font(.headline)
                            .foregroundColor(.white)
                            .frame(maxWidth: .infinity)
                            .padding()
                            .background(AppTheme.Colors.primaryBlue)
                            .cornerRadius(AppTheme.CornerRadius.button)
                        }

                        Button(action: onDelete) {
                            HStack {
                                Image(systemName: "trash")
                                Text("Delete")
                            }
                            .font(.headline)
                            .foregroundColor(.white)
                            .frame(maxWidth: .infinity)
                            .padding()
                            .background(Color.red)
                            .cornerRadius(AppTheme.CornerRadius.button)
                        }
                    }
                    .padding(.top, 8)
                }
                .padding()
            }
            .navigationTitle("Snippet Details")
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
}

struct AddSnippetView: View {
    @ObservedObject var viewModel: VocabularyViewModel
    @Binding var isPresented: Bool

    @State private var originalText = ""
    @State private var translatedText = ""
    @State private var sourceLanguage: String = "it"
    @State private var targetLanguage: String = "en"
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
                            ForEach(viewModel.availableLanguages) { lang in
                                Button(lang.name.capitalized) {
                                    sourceLanguage = lang.code
                                }
                            }
                        } label: {
                            HStack {
                                Text(
                                    viewModel.availableLanguages.find(byCodeOrName: sourceLanguage)?
                                        .name.capitalized
                                        ?? sourceLanguage.uppercased())
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
                            ForEach(viewModel.availableLanguages) { lang in
                                Button(lang.name.capitalized) {
                                    targetLanguage = lang.code
                                }
                            }
                        } label: {
                            HStack {
                                Text(
                                    viewModel.availableLanguages.find(byCodeOrName: targetLanguage)?
                                        .name.capitalized
                                        ?? targetLanguage.uppercased())
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
            sourceLanguage: sourceLanguage,
            targetLanguage: targetLanguage,
            context: context.isEmpty
                ? nil : context.trimmingCharacters(in: .whitespacesAndNewlines),
            questionId: nil,
            sectionId: nil,
            storyId: nil
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
    @State private var sourceLanguage: String = "it"
    @State private var targetLanguage: String = "en"
    @State private var context = ""
    @State private var isSubmitting = false
    @State private var errorMessage: String? = nil

    private func mapLanguageStringToCode(_ languageString: String?, defaultCode: String) -> String {
        guard let langString = languageString?.lowercased() else {
            return defaultCode
        }

        // Check if the string is already a valid code in available languages (O(1) lookup)
        if viewModel.availableLanguages.contains(where: { $0.code.lowercased() == langString }) {
            return langString
        }

        // Check if the string matches a language name (using extension for consistency)
        if let matchingLang = viewModel.availableLanguages.find(byCodeOrName: langString) {
            return matchingLang.code
        }

        // For unsupported languages, return default
        return defaultCode
    }

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
                        Menu {
                            ForEach(viewModel.availableLanguages) { lang in
                                Button(lang.name.capitalized) {
                                    sourceLanguage = lang.code
                                }
                            }
                        } label: {
                            HStack {
                                Text(
                                    viewModel.availableLanguages.find(byCodeOrName: sourceLanguage)?
                                        .name.capitalized
                                        ?? sourceLanguage.uppercased())
                                Spacer()
                                Image(systemName: "chevron.up.chevron.down")
                                    .font(.caption)
                            }
                            .padding(12)
                            .background(Color(.secondarySystemBackground))
                            .cornerRadius(8)
                        }
                    }

                    VStack(alignment: .leading, spacing: 8) {
                        Text("Target Language").font(.subheadline).fontWeight(.medium)
                        Menu {
                            ForEach(viewModel.availableLanguages) { lang in
                                Button(lang.name.capitalized) {
                                    targetLanguage = lang.code
                                }
                            }
                        } label: {
                            HStack {
                                Text(
                                    viewModel.availableLanguages.find(byCodeOrName: targetLanguage)?
                                        .name.capitalized
                                        ?? targetLanguage.uppercased())
                                Spacer()
                                Image(systemName: "chevron.up.chevron.down")
                                    .font(.caption)
                            }
                            .padding(12)
                            .background(Color(.secondarySystemBackground))
                            .cornerRadius(8)
                        }
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
            sourceLanguage = mapLanguageStringToCode(
                snippet.sourceLanguage, defaultCode: "it")
            targetLanguage = mapLanguageStringToCode(
                snippet.targetLanguage, defaultCode: "en")
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
            sourceLanguage: sourceLanguage,
            targetLanguage: targetLanguage,
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
