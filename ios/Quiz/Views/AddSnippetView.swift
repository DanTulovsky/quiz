import SwiftUI

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
                        }, label: {
                            Text("Cancel")
                                .font(.headline)
                                .frame(maxWidth: .infinity)
                                .padding()
                                .background(Color.blue.opacity(0.1))
                                .foregroundColor(AppTheme.Colors.primaryBlue)
                                .cornerRadius(12)
                        })

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


