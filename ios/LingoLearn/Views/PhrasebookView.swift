import SwiftUI

struct PhrasebookView: View {
    @StateObject private var viewModel = PhrasebookViewModel()
    
    var body: some View {
        List {
            if let phrasebook = viewModel.phrasebook {
                ForEach(phrasebook.categories, id: \.name) { category in
                    Section(header: Text(category.name)) {
                        ForEach(category.phrases, id: \.phrase) { phrase in
                            VStack(alignment: .leading) {
                                Text(phrase.phrase)
                                    .font(.headline)
                                Text(phrase.translation)
                                    .font(.subheadline)
                            }
                        }
                    }
                }
            }
        }
        .onAppear {
            viewModel.getPhrasebook(language: .en)
        }
        .navigationTitle("Phrasebook")
    }
}
