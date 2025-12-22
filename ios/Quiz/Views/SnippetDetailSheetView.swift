import SwiftUI

struct SnippetDetailSheetView: View {
    let snippet: Snippet
    @ObservedObject var viewModel: VocabularyViewModel
    @Binding var isPresented: Bool
    let onEdit: () -> Void
    let onDelete: () -> Void

    var body: some View {
        NavigationView {
            ScrollView {
                VStack(alignment: .leading, spacing: 24) {
                    // Original Text
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Original Text")
                            .font(.subheadline)
                            .fontWeight(.medium)
                            .foregroundColor(.secondary)
                        Text(snippet.originalText)
                            .font(.title2)
                            .fontWeight(.semibold)
                            .foregroundColor(.primary)
                    }

                    // Translation
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Translation")
                            .font(.subheadline)
                            .fontWeight(.medium)
                            .foregroundColor(.secondary)
                        Text(snippet.translatedText)
                            .font(.title3)
                            .foregroundColor(AppTheme.Colors.primaryBlue)
                    }

                    // Language and Level Info
                    HStack(spacing: 12) {
                        if let lang = snippet.sourceLanguage, let target = snippet.targetLanguage {
                            BadgeView(
                                text: "\(lang.uppercased()) → \(target.uppercased())",
                                color: AppTheme.Colors.accentIndigo)
                        }
                        if let level = snippet.difficultyLevel {
                            BadgeView(text: level, color: AppTheme.Colors.primaryBlue)
                        }
                    }

                    // Context
                    if let context = snippet.context, !context.isEmpty {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Context")
                                .font(.subheadline)
                                .fontWeight(.medium)
                                .foregroundColor(.secondary)
                            Text("\"\(context)\"")
                                .font(.body)
                                .foregroundColor(.primary)
                                .padding()
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(AppTheme.Colors.secondaryBackground)
                                .cornerRadius(AppTheme.CornerRadius.button)
                        }
                    }

                    // Action Buttons
                    VStack(spacing: 12) {
                        Button(action: onEdit) {
                            HStack {
                                Image(systemName: "square.and.pencil")
                                Text("Edit")
                            }
                            .font(.headline)
                            .foregroundColor(.white)
                            .frame(maxWidth: .infinity)
                            .padding()
                            .background(AppTheme.Colors.primaryBlue)
                            .cornerRadius(AppTheme.CornerRadius.button)
                        }

                        Button(action: onDelete) {
                            HStack {
                                Image(systemName: "trash")
                                Text("Delete")
                            }
                            .font(.headline)
                            .foregroundColor(.white)
                            .frame(maxWidth: .infinity)
                            .padding()
                            .background(Color.red)
                            .cornerRadius(AppTheme.CornerRadius.button)
                        }
                    }
                    .padding(.top, 8)
                }
                .padding()
            }
            .navigationTitle("Snippet Details")
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
}

