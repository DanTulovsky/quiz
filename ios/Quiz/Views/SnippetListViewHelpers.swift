import SwiftUI

extension SnippetListView {
    var hasActiveFilters: Bool {
        viewModel.selectedStoryId != nil || viewModel.selectedLevel != nil
            || viewModel.selectedSourceLang != nil
    }

    var filterSummary: String {
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
    func storyFilterPicker() -> some View {
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
    func levelFilterPicker() -> some View {
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
    func sourceLanguageFilterPicker() -> some View {
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





