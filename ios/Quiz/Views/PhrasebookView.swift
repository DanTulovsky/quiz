import SwiftUI

struct PhrasebookView: View {
    @StateObject private var viewModel = PhrasebookViewModel()

    var body: some View {
        VStack(spacing: 0) {
            // Header Stats
            HStack {
                BadgeView(
                    text: "\(viewModel.categories.count) CATEGORIES",
                    color: AppTheme.Colors.primaryBlue)

                Spacer()
            }
            .padding(.horizontal)
            .padding(.top)

            // Search Bar (Placeholder for now)
            HStack {
                Image(systemName: "magnifyingglass")
                    .foregroundColor(AppTheme.Colors.secondaryText)
                TextField("Search phrasebook...", text: .constant(""))
            }
            .padding(10)
            .background(AppTheme.Colors.secondaryBackground)
            .cornerRadius(AppTheme.CornerRadius.button)
            .padding()

            List(viewModel.categories, id: \.id) { category in
                NavigationLink(
                    destination: PhrasebookCategoryView(
                        categoryId: category.id, title: category.name)
                ) {
                    HStack {
                        if let emoji = category.emoji {
                            Text(emoji)
                                .font(.title3)
                        }
                        Text(category.name)
                            .font(.headline)
                    }
                    .padding(.vertical, 4)
                }
            }
            .listStyle(.plain)
        }
        .onAppear {
            viewModel.fetchCategories()
        }
        .navigationTitle("Phrasebook")
    }
}

struct PhrasebookCategoryView: View {
    @StateObject private var viewModel = PhrasebookViewModel()
    @EnvironmentObject var authViewModel: AuthenticationViewModel
    let categoryId: String
    let title: String

    @State private var searchText = ""
    @State private var selectedSection = "All Sections"

    var languageCode: String {
        let langName = (authViewModel.user?.preferredLanguage ?? "it")
        return viewModel.availableLanguages.find(byCodeOrName: langName)?.code ?? "it"
    }

    var body: some View {
        VStack(spacing: 0) {
            if let data = viewModel.selectedCategoryData {
                VStack(spacing: 12) {
                    // Search Bar
                    HStack {
                        Image(systemName: "magnifyingglass")
                            .foregroundColor(AppTheme.Colors.secondaryText)
                        TextField("Search terms...", text: $searchText)
                    }
                    .padding(10)
                    .background(AppTheme.Colors.secondaryBackground)
                    .cornerRadius(AppTheme.CornerRadius.button)
                    .padding(.horizontal)

                    // Section Picker
                    VStack(alignment: .leading, spacing: 4) {
                        Text("Section")
                            .font(AppTheme.Typography.captionFont)
                            .foregroundColor(AppTheme.Colors.secondaryText)

                        Menu {
                            Button("All Sections") { selectedSection = "All Sections" }
                            ForEach(data.sections, id: \.title) { section in
                                Button(section.title) { selectedSection = section.title }
                            }
                        } label: {
                            HStack {
                                Text(selectedSection)
                                    .foregroundColor(.primary)
                                Spacer()
                                Image(systemName: "chevron.up.chevron.down")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }
                            .padding(10)
                            .background(AppTheme.Colors.secondaryBackground)
                            .cornerRadius(AppTheme.CornerRadius.button)
                            .overlay(
                                RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button)
                                    .stroke(AppTheme.Colors.borderGray, lineWidth: 1)
                            )
                        }
                    }
                    .padding(.horizontal)
                }
                .padding(.vertical)

                List {
                    ForEach(
                        data.sections.filter {
                            selectedSection == "All Sections" || $0.title == selectedSection
                        },
                        id: \.title
                    ) { section in
                        let filteredWords = section.words.filter {
                            searchText.isEmpty
                                || $0.term.localizedCaseInsensitiveContains(searchText)
                                || ($0.translations[languageCode]?.localizedCaseInsensitiveContains(
                                    searchText) ?? false)
                        }

                        if !filteredWords.isEmpty {
                            Section(
                                header:
                                    HStack {
                                        Text(section.title)
                                            .font(AppTheme.Typography.headingFont)
                                            .foregroundColor(AppTheme.Colors.primaryText)
                                        BadgeView(
                                            text: "\(filteredWords.count)",
                                            color: AppTheme.Colors.primaryBlue)
                                    }
                                    .padding(.vertical, 4)
                            ) {
                                ForEach(filteredWords, id: \.term) { word in
                                    HStack(alignment: .center) {
                                        VStack(alignment: .leading, spacing: 4) {
                                            HStack {
                                                if let icon = word.icon { Text(icon) }
                                                Text(word.term.capitalized)
                                                    .font(AppTheme.Typography.headingFont)
                                            }
                                            Text(word.translations[languageCode] ?? "N/A")
                                                .font(AppTheme.Typography.subheadlineFont)
                                                .foregroundColor(AppTheme.Colors.primaryBlue)
                                            if let note = word.note {
                                                Text(note)
                                                    .font(AppTheme.Typography.captionFont)
                                                    .italic()
                                                    .foregroundColor(AppTheme.Colors.secondaryText)
                                            }
                                        }

                                        Spacer()

                                        HStack(spacing: 15) {
                                            TTSButton(
                                                text: word.translations[languageCode] ?? word.term,
                                                language: authViewModel.user?.preferredLanguage
                                                    ?? "italian"
                                            )
                                            Button(
                                                action: {
                                                    UIPasteboard.general.string =
                                                        word.translations[languageCode] ?? word.term
                                                },
                                                label: {
                                                    Image(systemName: "doc.on.doc")
                                                        .foregroundColor(
                                                            AppTheme.Colors.primaryBlue)
                                                }
                                            )
                                            .buttonStyle(.plain)
                                        }
                                    }
                                    .padding(.vertical, 4)
                                }
                            }
                        }
                    }
                }
                .listStyle(.plain)
            } else if viewModel.isLoading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                Text("No data available for this category.")
                    .foregroundColor(AppTheme.Colors.secondaryText)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .navigationTitle(title)
        .onAppear {
            viewModel.fetchCategoryData(id: categoryId)
            viewModel.fetchLanguages()
        }
    }
}
