import SwiftUI

struct VocabularyView: View {
    @StateObject private var viewModel = VocabularyViewModel()
    
    var body: some View {
        List(viewModel.snippets, id: \.id) { snippet in
            VStack(alignment: .leading) {
                Text(snippet.text)
                    .font(.headline)
                Text(snippet.translation)
                    .font(.subheadline)
            }
        }
        .onAppear {
            viewModel.getSnippets()
        }
        .navigationTitle("Vocabulary")
    }
}
