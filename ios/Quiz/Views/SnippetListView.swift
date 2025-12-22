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
    @State private var selectedSnippet: Snippet?
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
                }, label: {
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
                })
                .padding(.horizontal)

                // Search and Filters
                VStack(spacing: 0) {
                    Button(action: {
                        withAnimation {
                            filtersExpanded.toggle()
                        }
                    }, label: {
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
                    })
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
            viewModel.fetchItems()
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
                Button(action: { dismiss() }, label: {
                    HStack(spacing: 4) {
                        Image(systemName: "chevron.left")
                            .scaledFont(size: 17, weight: .semibold)
                        Text("Back")
                            .scaledFont(size: 17)
                    }
                    .foregroundColor(AppTheme.Colors.primaryBlue)
                })
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
           let story = viewModel.stories.first(where: { $0.id == selectedId }) {
            parts.append(story.title)
        }
        if let level = viewModel.selectedLevel {
            parts.append(level)
        }
        if let selectedLangCode = viewModel.selectedSourceLang,
           let lang = viewModel.availableLanguages.find(byCodeOrName: selectedLangCode) {
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
                   let lang = viewModel.availableLanguages.find(byCodeOrName: selectedLangCode) {
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


