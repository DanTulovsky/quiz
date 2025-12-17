import SwiftUI

struct PhrasebookView: View {
    @StateObject private var viewModel = PhrasebookViewModel()

    var body: some View {
        VStack(spacing: 0) {
            // Header Stats
            HStack {
                Text("\(viewModel.categories.count) CATEGORIES")
                    .font(.caption)
                    .bold()
                    .padding(6)
                    .background(Color.blue.opacity(0.1))
                    .foregroundColor(.blue)
                    .clipShape(RoundedRectangle(cornerRadius: 6))

                Spacer()
            }
            .padding(.horizontal)
            .padding(.top)

            // Search Bar (Placeholder for now)
            HStack {
                Image(systemName: "magnifyingglass")
                    .foregroundColor(.secondary)
                TextField("Search phrasebook...", text: .constant(""))
            }
            .padding(10)
            .background(Color(.secondarySystemBackground))
            .cornerRadius(10)
            .padding()

            List(viewModel.categories, id: \.id) { category in
                NavigationLink(destination: PhrasebookCategoryView(categoryId: category.id, title: category.name)) {
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
        return Language(rawValue: langName)?.code ?? "it"
    }

    var body: some View {
        VStack(spacing: 0) {
            if let data = viewModel.selectedCategoryData {
                VStack(spacing: 12) {
                    // Search Bar
                    HStack {
                        Image(systemName: "magnifyingglass")
                            .foregroundColor(.secondary)
                        TextField("Search terms...", text: $searchText)
                    }
                    .padding(10)
                    .background(Color(.secondarySystemBackground))
                    .cornerRadius(10)
                    .padding(.horizontal)

                    // Section Picker
                    VStack(alignment: .leading, spacing: 4) {
                        Text("Section")
                            .font(.caption)
                            .foregroundColor(.secondary)

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
                            .background(Color(.secondarySystemBackground))
                            .cornerRadius(8)
                            .overlay(
                                RoundedRectangle(cornerRadius: 8)
                                    .stroke(Color.gray.opacity(0.2), lineWidth: 1)
                            )
                        }
                    }
                    .padding(.horizontal)
                }
                .padding(.vertical)

                List {
                    ForEach(data.sections.filter { selectedSection == "All Sections" || $0.title == selectedSection }, id: \.title) { section in
                        let filteredWords = section.words.filter {
                            searchText.isEmpty ||
                            $0.term.localizedCaseInsensitiveContains(searchText) ||
                            ($0.translations[languageCode]?.localizedCaseInsensitiveContains(searchText) ?? false)
                        }

                        if !filteredWords.isEmpty {
                            Section(header:
                                HStack {
                                    Text(section.title)
                                        .font(.headline)
                                        .foregroundColor(.primary)
                                    BadgeView(text: "\(filteredWords.count)", color: .blue)
                                }
                                .padding(.vertical, 4)
                            ) {
                                ForEach(filteredWords, id: \.term) { word in
                                    HStack(alignment: .center) {
                                        VStack(alignment: .leading, spacing: 4) {
                                            HStack {
                                                if let icon = word.icon { Text(icon) }
                                                Text(word.term.capitalized)
                                                    .font(.headline)
                                            }
                                            Text(word.translations[languageCode] ?? "N/A")
                                                .font(.subheadline)
                                                .foregroundColor(.blue)
                                            if let note = word.note {
                                                Text(note)
                                                    .font(.caption)
                                                    .italic()
                                                    .foregroundColor(.secondary)
                                            }
                                        }

                                        Spacer()

                                        HStack(spacing: 15) {
                                            TTSButton(text: word.term, language: authViewModel.user?.preferredLanguage ?? "italian")
                                            Button(action: {
                                                UIPasteboard.general.string = word.term
                                            }) {
                                                Image(systemName: "doc.on.doc")
                                                    .foregroundColor(.blue)
                                            }
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
                    .foregroundColor(.secondary)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .navigationTitle(title)
        .onAppear {
            viewModel.fetchCategoryData(id: categoryId)
        }
    }
}
