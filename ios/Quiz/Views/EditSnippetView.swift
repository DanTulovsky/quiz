import SwiftUI

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
    @State private var errorMessage: String?

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
                    }, label: {
                        Image(systemName: "xmark")
                            .foregroundColor(.primary)
                            .padding(8)
                            .background(Color.blue.opacity(0.1))
                            .clipShape(Circle())
                    })
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


