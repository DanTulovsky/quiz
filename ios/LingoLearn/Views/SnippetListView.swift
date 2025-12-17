import SwiftUI

struct SnippetListView: View {
    @StateObject private var viewModel = VocabularyViewModel()
    @State private var selectedStory: String = "Filter by story"
    @State private var selectedLevel: String = "Filter by level"
    @State private var selectedSourceLang: String = "Filter by source language"
    
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
                Button(action: {}) {
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
                VStack(spacing: 12) {
                    HStack {
                        Image(systemName: "magnifyingglass")
                            .foregroundColor(.secondary)
                        TextField("Search snippets...", text: $viewModel.searchQuery)
                    }
                    .padding(12)
                    .background(Color(.secondarySystemBackground))
                    .cornerRadius(10)
                    
                    Button(action: {
                        viewModel.getSnippets()
                    }) {
                        HStack {
                            Image(systemName: "magnifyingglass")
                            Text("Search")
                        }
                        .font(.subheadline)
                        .fontWeight(.medium)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                        .background(Color.blue.opacity(0.1))
                        .foregroundColor(.blue)
                        .cornerRadius(10)
                    }
                }
                .padding(.horizontal)
                
                // Filters
                VStack(spacing: 12) {
                    filterPicker(selection: $selectedStory, options: ["Filter by story"])
                    filterPicker(selection: $selectedLevel, options: ["Filter by level", "A1", "A2", "B1", "B2", "C1", "C2"])
                    filterPicker(selection: $selectedSourceLang, options: ["Filter by source language", "Italian", "Spanish", "French", "German", "English"])
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
            viewModel.getSnippets()
        }
        .navigationBarTitleDisplayMode(.inline)
    }
    
    @ViewBuilder
    private func filterPicker(selection: Binding<String>, options: [String]) -> some View {
        Menu {
            ForEach(options, id: \.self) { option in
                Button(option) { selection.wrappedValue = option }
            }
        } label: {
            HStack {
                Text(selection.wrappedValue)
                    .foregroundColor(selection.wrappedValue.contains("Filter") ? .secondary : .primary)
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
