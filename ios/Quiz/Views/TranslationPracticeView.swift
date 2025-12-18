import SwiftUI

struct TranslationPracticeView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel = TranslationPracticeViewModel()
    @EnvironmentObject var authViewModel: AuthenticationViewModel

    var body: some View {
        ScrollView {
            ScrollViewReader { proxy in
                VStack(spacing: 25) {
                    // Header Language Selection
                    HStack {
                        Spacer()
                        Picker("Direction", selection: $viewModel.selectedDirection) {
                            let langName = authViewModel.user?.preferredLanguage?.capitalized ?? "Learning"
                            Text("\(langName) → English").tag("learning_to_en")
                            Text("English → \(langName)").tag("en_to_learning")
                        }
                        .pickerStyle(.menu)
                        .padding(8)
                        .background(Color(.secondarySystemBackground))
                        .cornerRadius(10)
                        Spacer()
                    }

                    // Action Buttons
                    HStack(spacing: 15) {
                        Button(action: {
                            let lang = authViewModel.user?.preferredLanguage ?? "italian"
                            let level = authViewModel.user?.currentLevel ?? "A1"
                            viewModel.generateSentence(language: lang, level: level)
                        }) {
                            Text("Generate AI")
                                .font(.subheadline)
                                .fontWeight(.medium)
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 12)
                                .background(Color.blue.opacity(0.1))
                                .foregroundColor(.blue)
                                .cornerRadius(10)
                        }

                        Button(action: {
                            let lang = authViewModel.user?.preferredLanguage ?? "italian"
                            let level = authViewModel.user?.currentLevel ?? "A1"
                            viewModel.fetchExistingSentence(language: lang, level: level)
                        }) {
                            Text("From Content")
                                .font(.subheadline)
                                .fontWeight(.medium)
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 12)
                                .background(Color.blue.opacity(0.1))
                                .foregroundColor(.blue)
                                .cornerRadius(10)
                        }
                    }

                    if viewModel.isLoading && viewModel.currentSentence == nil {
                        ProgressView()
                            .padding(.top, 20)
                    } else if let sentence = viewModel.currentSentence {
                        promptCard(sentence).id("prompt_card")
                    }

                    if !viewModel.history.isEmpty {
                        historySection
                    }
                }
                .padding()
                Color.clear.frame(height: 1).id("bottom")

                    .onChange(of: viewModel.currentSentence) { old, val in
                        if val != nil {
                            withAnimation {
                                proxy.scrollTo("prompt_card", anchor: .top)
                            }
                        }
                    }
                    .onChange(of: viewModel.feedback) { old, val in
                        if val != nil {
                            withAnimation {
                                proxy.scrollTo("bottom", anchor: .bottom)
                            }
                        }
                    }
            }
        }
        .navigationTitle("Translation Practice")
        .navigationBarTitleDisplayMode(.inline)
        .navigationBarBackButtonHidden(true)
        .toolbar {
            ToolbarItem(placement: .navigationBarLeading) {
                Button(action: { dismiss() }) {
                    HStack(spacing: 4) {
                        Image(systemName: "chevron.left")
                            .font(.system(size: 17, weight: .semibold))
                        Text("Back")
                            .font(.system(size: 17))
                    }
                    .foregroundColor(.blue)
                }
            }
        }
        .onAppear {
            viewModel.fetchHistory()
        }
    }

    @ViewBuilder
    private func promptCard(_ sentence: TranslationPracticeSentenceResponse) -> some View {
        VStack(alignment: .leading, spacing: 20) {
            // Card Header
            HStack {
                Text("Prompt")
                    .font(.title3)
                    .fontWeight(.bold)
                Spacer()
                HStack(spacing: 8) {
                    BadgeView(text: sentence.sourceLanguage.uppercased(), color: .blue)
                    BadgeView(text: "LEVEL \(sentence.languageLevel.uppercased())", color: .blue)
                }
            }

            // Topic Input
            VStack(alignment: .leading, spacing: 8) {
                Text("Optional topic")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                TextField("e.g., travel, ordering food, work", text: $viewModel.optionalTopic)
                    .padding(12)
                    .background(Color(.secondarySystemBackground))
                    .cornerRadius(8)
                    .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.gray.opacity(0.2), lineWidth: 1))
            }

            // Text to Translate Section
            VStack(alignment: .leading, spacing: 12) {
                HStack {
                    Text("Text to translate")
                        .font(.subheadline)
                        .fontWeight(.semibold)
                    Spacer()
                    TTSButton(text: sentence.sentenceText, language: sentence.sourceLanguage)
                }

                VStack(alignment: .leading, spacing: 8) {
                    Text(sentence.sentenceText)
                        .font(.title3)
                        .fontWeight(.medium)

                    HStack(spacing: 4) {
                        Text("From existing content:")
                            .font(.caption)
                            .foregroundColor(.secondary)
                        Text(sentence.sourceType.replacingOccurrences(of: "_", with: " "))
                            .font(.caption)
                            .foregroundColor(.blue)
                    }
                }
                .padding()
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(Color.blue.opacity(0.03))
                .cornerRadius(12)
                .contentShape(Rectangle())
                .overlay(RoundedRectangle(cornerRadius: 12).stroke(Color.blue.opacity(0.1), lineWidth: 1))
            }

            // User Input Section
            VStack(alignment: .leading, spacing: 12) {
                Text("Your translation")
                    .font(.subheadline)
                    .fontWeight(.semibold)

                TextEditor(text: $viewModel.userTranslation)
                    .frame(minHeight: 100)
                    .padding(8)
                    .background(Color(.systemBackground))
                    .cornerRadius(8)
                    .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.gray.opacity(0.2), lineWidth: 1))
            }

            if let feedback = viewModel.feedback {
                feedbackSection(feedback)
            }

            Button(action: {
                // Force resign focus to ensure binding is updated
                UIApplication.shared.sendAction(#selector(UIResponder.resignFirstResponder), to: nil, from: nil, for: nil)
                viewModel.submitTranslation()
            }) {
                HStack {
                    if viewModel.isLoading {
                        ProgressView()
                            .progressViewStyle(CircularProgressViewStyle(tint: .white))
                            .padding(.trailing, 8)
                    }
                    Text(viewModel.isLoading ? "Submitting..." : "Submit for feedback")
                        .font(.headline)
                }
                .frame(maxWidth: .infinity)
                .padding()
                .background(viewModel.userTranslation.isEmpty ? Color.gray : Color.blue)
                .foregroundColor(.white)
                .cornerRadius(12)
                .contentShape(Rectangle())
            }
            .disabled(viewModel.userTranslation.isEmpty || viewModel.isLoading)
            if let error = viewModel.error {
                Text(error.localizedDescription)
                    .font(.caption)
                    .foregroundColor(.red)
                    .padding(.top, 4)
            }
        }
        .padding(20)
        .background(Color(.systemBackground))
        .cornerRadius(16)
        .shadow(color: Color.black.opacity(0.05), radius: 10, x: 0, y: 5)
        .overlay(RoundedRectangle(cornerRadius: 16).stroke(Color.gray.opacity(0.1), lineWidth: 1))
    }

    private func feedbackSection(_ feedback: TranslationPracticeSessionResponse) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: "sparkles")
                    .foregroundColor(.orange)
                Text("AI Feedback")
                    .font(.headline)
                Spacer()
                if let score = feedback.aiScore {
                    Text("Score: \(Int(score * 100))%")
                        .font(.caption)
                        .fontWeight(.bold)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(Color.orange.opacity(0.1))
                        .foregroundColor(.orange)
                        .cornerRadius(6)
                }
            }

            Text(feedback.aiFeedback)
                .font(.subheadline)
                .lineSpacing(4)
        }
        .padding()
        .background(Color.orange.opacity(0.05))
        .cornerRadius(12)
                .contentShape(Rectangle())
        .overlay(RoundedRectangle(cornerRadius: 12).stroke(Color.orange.opacity(0.2), lineWidth: 1))
    }

    private var historySection: some View {
        VStack(alignment: .leading, spacing: 15) {
            HStack {
                Text("History")
                    .font(.title3)
                    .fontWeight(.bold)
                Spacer()
                Text("Showing 1-\(viewModel.history.count) of \(viewModel.totalHistoryCount)")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }

            VStack(spacing: 12) {
                ForEach(viewModel.history) { session in
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            BadgeView(text: session.translationDirection.replacingOccurrences(of: "_", with: " ").uppercased(), color: .gray)
                            Spacer()
                            if let score = session.aiScore {
                                Text("\(Int(score * 100))%")
                                    .font(.caption2)
                                    .fontWeight(.bold)
                                    .foregroundColor(score > 0.8 ? .green : .orange)
                            }
                        }

                        Text(session.originalSentence)
                            .font(.subheadline)
                            .fontWeight(.medium)
                            .lineLimit(1)

                        Text(session.userTranslation)
                            .font(.caption)
                            .foregroundColor(.secondary)
                            .lineLimit(1)
                    }
                    .padding()
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(Color(.secondarySystemBackground).opacity(0.5))
                    .cornerRadius(10)
                }
            }
        }
        .padding(20)
        .background(Color(.systemBackground))
        .cornerRadius(16)
        .shadow(color: Color.black.opacity(0.05), radius: 10, x: 0, y: 5)
        .overlay(RoundedRectangle(cornerRadius: 16).stroke(Color.gray.opacity(0.1), lineWidth: 1))
    }
}
