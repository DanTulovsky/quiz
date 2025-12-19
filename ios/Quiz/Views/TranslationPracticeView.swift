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
                        .background(AppTheme.Colors.secondaryBackground)
                        .cornerRadius(AppTheme.CornerRadius.button)
                        Spacer()
                    }

                    // Optional Topic Field (shown on initial screen)
                    if viewModel.currentSentence == nil {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Optional topic")
                                .font(AppTheme.Typography.subheadlineFont)
                                .foregroundColor(AppTheme.Colors.secondaryText)
                            TextField("e.g., travel, ordering food, work", text: $viewModel.optionalTopic)
                                .padding(12)
                                .background(AppTheme.Colors.secondaryBackground)
                                .cornerRadius(AppTheme.CornerRadius.button)
                                .overlay(RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(AppTheme.Colors.borderGray, lineWidth: 1))
                        }
                    }

                    // Action Buttons
                    HStack(spacing: 15) {
                        Button(action: {
                            let lang = authViewModel.user?.preferredLanguage ?? "italian"
                            let level = authViewModel.user?.currentLevel ?? "A1"
                            viewModel.generateSentence(language: lang, level: level)
                        }) {
                            Text("Generate AI")
                                .font(AppTheme.Typography.subheadlineFont.weight(.medium))
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, AppTheme.Spacing.buttonVerticalPadding)
                                .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                                .foregroundColor(AppTheme.Colors.primaryBlue)
                                .cornerRadius(AppTheme.CornerRadius.button)
                        }

                        Button(action: {
                            let lang = authViewModel.user?.preferredLanguage ?? "italian"
                            let level = authViewModel.user?.currentLevel ?? "A1"
                            viewModel.fetchExistingSentence(language: lang, level: level)
                        }) {
                            Text("From Content")
                                .font(AppTheme.Typography.subheadlineFont.weight(.medium))
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, AppTheme.Spacing.buttonVerticalPadding)
                                .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                                .foregroundColor(AppTheme.Colors.primaryBlue)
                                .cornerRadius(AppTheme.CornerRadius.button)
                        }
                    }

                    if viewModel.isLoading && viewModel.currentSentence == nil {
                        ProgressView()
                            .padding(.top, 20)
                    } else if let sentence = viewModel.currentSentence {
                        promptCard(sentence).id("prompt_card")
                    }

                    // Error display for initial screen (when generation fails)
                    if viewModel.currentSentence == nil, let error = viewModel.error {
                        VStack(spacing: 12) {
                            HStack {
                                Image(systemName: "exclamationmark.triangle.fill")
                                    .foregroundColor(AppTheme.Colors.errorRed)
                                Text("Error")
                                    .font(AppTheme.Typography.subheadlineFont.weight(.semibold))
                                    .foregroundColor(AppTheme.Colors.errorRed)
                                Spacer()
                            }
                            Text(error.localizedDescription)
                                .font(AppTheme.Typography.subheadlineFont)
                                .foregroundColor(AppTheme.Colors.secondaryText)
                                .frame(maxWidth: .infinity, alignment: .leading)
                        }
                        .padding(AppTheme.Spacing.innerPadding)
                        .background(AppTheme.Colors.errorRed.opacity(0.1))
                        .cornerRadius(AppTheme.CornerRadius.button)
                        .overlay(RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(AppTheme.Colors.errorRed.opacity(0.3), lineWidth: 1))
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
                    .font(AppTheme.Typography.headingFont)
                Spacer()
                HStack(spacing: 8) {
                    BadgeView(text: sentence.sourceLanguage.uppercased(), color: AppTheme.Colors.primaryBlue)
                    BadgeView(text: "LEVEL \(sentence.languageLevel.uppercased())", color: AppTheme.Colors.primaryBlue)
                }
            }

            // Topic Input
            VStack(alignment: .leading, spacing: 8) {
                Text("Optional topic")
                    .font(AppTheme.Typography.subheadlineFont)
                    .foregroundColor(AppTheme.Colors.secondaryText)
                TextField("e.g., travel, ordering food, work", text: $viewModel.optionalTopic)
                    .padding(12)
                    .background(AppTheme.Colors.secondaryBackground)
                    .cornerRadius(AppTheme.CornerRadius.button)
                    .overlay(RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(AppTheme.Colors.borderGray, lineWidth: 1))
            }

            // Text to Translate Section
            VStack(alignment: .leading, spacing: AppTheme.Spacing.itemSpacing) {
                HStack {
                    Text("Text to translate")
                        .font(AppTheme.Typography.subheadlineFont.weight(.semibold))
                    Spacer()
                    TTSButton(text: sentence.sentenceText, language: sentence.sourceLanguage)
                }

                VStack(alignment: .leading, spacing: 8) {
                    Text(sentence.sentenceText)
                        .font(AppTheme.Typography.headingFont)

                    HStack(spacing: 4) {
                        Text("From existing content:")
                            .font(AppTheme.Typography.captionFont)
                            .foregroundColor(AppTheme.Colors.secondaryText)
                        Text(sentence.sourceType.replacingOccurrences(of: "_", with: " "))
                            .font(AppTheme.Typography.captionFont)
                            .foregroundColor(AppTheme.Colors.primaryBlue)
                    }
                }
                .padding(AppTheme.Spacing.innerPadding)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(AppTheme.Colors.primaryBlue.opacity(0.03))
                .cornerRadius(AppTheme.CornerRadius.button)
                .contentShape(Rectangle())
                .overlay(RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(AppTheme.Colors.primaryBlue.opacity(0.1), lineWidth: 1))
            }

            // User Input Section
            VStack(alignment: .leading, spacing: AppTheme.Spacing.itemSpacing) {
                Text("Your translation")
                    .font(AppTheme.Typography.subheadlineFont.weight(.semibold))

                TextEditor(text: $viewModel.userTranslation)
                    .frame(minHeight: 100)
                    .padding(8)
                    .background(AppTheme.Colors.cardBackground)
                    .cornerRadius(AppTheme.CornerRadius.button)
                    .overlay(RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(AppTheme.Colors.borderGray, lineWidth: 1))
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
                        .font(AppTheme.Typography.buttonFont)
                }
                .frame(maxWidth: .infinity)
                .padding()
                .background(viewModel.userTranslation.isEmpty ? Color.gray : AppTheme.Colors.primaryBlue)
                .foregroundColor(.white)
                .cornerRadius(AppTheme.CornerRadius.button)
                .contentShape(Rectangle())
            }
            .disabled(viewModel.userTranslation.isEmpty || viewModel.isLoading)
            if let error = viewModel.error {
                Text(error.localizedDescription)
                    .font(AppTheme.Typography.captionFont)
                    .foregroundColor(AppTheme.Colors.errorRed)
                    .padding(.top, 4)
            }
        }
        .appCard()
    }

    private func feedbackSection(_ feedback: TranslationPracticeSessionResponse) -> some View {
        VStack(alignment: .leading, spacing: AppTheme.Spacing.itemSpacing) {
            HStack {
                Image(systemName: "sparkles")
                    .foregroundColor(AppTheme.Colors.primaryBlue)
                Text("AI Feedback")
                    .font(AppTheme.Typography.headingFont)
                Spacer()
                if let score = feedback.aiScore {
                    Text("Score: \(Int(score * 100))%")
                        .font(AppTheme.Typography.captionFont.weight(.bold))
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                        .foregroundColor(AppTheme.Colors.primaryBlue)
                        .cornerRadius(6)
                }
            }

            Text(feedback.aiFeedback)
                .font(AppTheme.Typography.subheadlineFont)
                .lineSpacing(4)
        }
        .padding(AppTheme.Spacing.innerPadding)
        .background(AppTheme.Colors.primaryBlue.opacity(0.05))
        .cornerRadius(AppTheme.CornerRadius.button)
        .contentShape(Rectangle())
        .overlay(RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(AppTheme.Colors.primaryBlue.opacity(0.2), lineWidth: 1))
    }

    private var historySection: some View {
        VStack(alignment: .leading, spacing: 15) {
            HStack {
                Text("History")
                    .font(AppTheme.Typography.headingFont)
                Spacer()
                Text("Showing 1-\(viewModel.history.count) of \(viewModel.totalHistoryCount)")
                    .font(AppTheme.Typography.captionFont)
                    .foregroundColor(AppTheme.Colors.secondaryText)
            }

            VStack(spacing: AppTheme.Spacing.itemSpacing) {
                ForEach(viewModel.history) { session in
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            BadgeView(text: session.translationDirection.replacingOccurrences(of: "_", with: " ").uppercased(), color: AppTheme.Colors.accentIndigo)
                            Spacer()
                            if let score = session.aiScore {
                                Text("\(Int(score * 100))%")
                                    .font(AppTheme.Typography.badgeFont)
                                    .foregroundColor(score > 0.8 ? AppTheme.Colors.successGreen : AppTheme.Colors.primaryBlue)
                            }
                        }

                        Text(session.originalSentence)
                            .font(AppTheme.Typography.subheadlineFont.weight(.medium))
                            .lineLimit(1)

                        Text(session.userTranslation)
                            .font(AppTheme.Typography.captionFont)
                            .foregroundColor(AppTheme.Colors.secondaryText)
                            .lineLimit(1)
                    }
                    .padding(AppTheme.Spacing.innerPadding)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(AppTheme.Colors.secondaryBackground.opacity(0.5))
                    .cornerRadius(10)
                }
            }
        }
        .appCard()
    }
}
