import SwiftUI

struct SnippetDetailPopup: ViewModifier {
    @Binding var showingSnippet: Snippet?
    @State private var vocabularyViewModel: VocabularyViewModel?
    @State private var showSnippetsView = false
    @State private var snippetSearchQuery = ""
    @State private var showDeleteConfirmation = false
    @State private var snippetToDelete: Snippet? = nil
    var onSnippetDeleted: ((Snippet) -> Void)?

    private func getVocabularyViewModel() -> VocabularyViewModel {
        if vocabularyViewModel == nil {
            vocabularyViewModel = VocabularyViewModel()
        }
        return vocabularyViewModel!
    }

    func body(content: Content) -> some View {
        content
            .overlay(
                Group {
                    if let snippet = showingSnippet {
                        Color.black.opacity(0.3)
                            .ignoresSafeArea()
                            .onTapGesture { showingSnippet = nil }

                        SnippetDetailView(
                            snippet: snippet,
                            onClose: {
                                showingSnippet = nil
                            },
                            onNavigateToSnippets: { searchText in
                                showingSnippet = nil
                                snippetSearchQuery = searchText
                                showSnippetsView = true
                            },
                            onDelete: {
                                snippetToDelete = snippet
                                showDeleteConfirmation = true
                            }
                        )
                        .transition(.scale.combined(with: .opacity))
                        .zIndex(1)
                    }
                }
            )
            .sheet(isPresented: $showSnippetsView) {
                NavigationView {
                    SnippetListViewWithSearch(query: snippetSearchQuery)
                }
            }
            .alert("Delete Snippet", isPresented: $showDeleteConfirmation) {
                Button("Cancel", role: .cancel) {
                    snippetToDelete = nil
                }
                Button("Delete", role: .destructive) {
                    if let snippet = snippetToDelete {
                        getVocabularyViewModel().deleteSnippet(id: snippet.id) { result in
                            switch result {
                            case .success:
                                onSnippetDeleted?(snippet)
                                showingSnippet = nil
                            case .failure:
                                break
                            }
                        }
                        snippetToDelete = nil
                    }
                }
            } message: {
                Text("Are you sure you want to delete this snippet?")
            }
    }
}

extension View {
    func snippetDetailPopup(
        showingSnippet: Binding<Snippet?>,
        onSnippetDeleted: ((Snippet) -> Void)? = nil
    ) -> some View {
        modifier(SnippetDetailPopup(showingSnippet: showingSnippet, onSnippetDeleted: onSnippetDeleted))
    }
}

