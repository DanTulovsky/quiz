import SwiftUI

struct TranslationPracticeView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel = TranslationPracticeViewModel()
    @EnvironmentObject var authViewModel: AuthenticationViewModel
    @State private var showErrorDetails = false
    @State private var selectedHistorySession: TranslationPracticeSessionResponse?

    var body: some View {
        ScrollView {
            ScrollViewReader { proxy in
                VStack(spacing: 25) {
                    // Header Language Selection
                    HStack {
                        Spacer()
                        Picker("Direction", selection: $viewModel.selectedDirection) {
                            let langName =
                                authViewModel.user?.preferredLanguage?.capitalized ?? "Learning"
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
                            TextField(
                                "e.g., travel, ordering food, work", text: $viewModel.optionalTopic
                            )
                            .padding(12)
                            .background(AppTheme.Colors.secondaryBackground)
                            .cornerRadius(AppTheme.CornerRadius.button)
                            .overlay(
                                RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(
                                    AppTheme.Colors.borderGray, lineWidth: 1))
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
                        LoadingView(message: "Loading sentence...")
                            .padding(.top, 20)
                    } else if let sentence = viewModel.currentSentence {
                        promptCard(sentence).id("prompt_card")
                    }

                    // Error display for initial screen (when generation fails)
                    if viewModel.currentSentence == nil {
                        ErrorDisplay(
                            error: viewModel.error,
                            onDismiss: {
                                viewModel.clearError()
                            },
                            showDetailsButton: true,
                            onShowDetails: {
                                showErrorDetails = true
                            }
                        )
                        .padding(.horizontal)
                    }

                    if viewModel.history.isEmpty {
                        EmptyStateView(
                            icon: "arrow.left.and.right",
                            title: "No Translation History",
                            message: "Your translation practice history will appear here after you complete some translations."
                        )
                        .padding()
                    } else {
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
                            .scaledFont(size: 17, weight: .semibold)
                        Text("Back")
                            .scaledFont(size: 17)
                    }
                    .foregroundColor(.blue)
                }
            }
        }
        .onAppear {
            viewModel.fetchHistory()
        }
        .sheet(isPresented: $showErrorDetails) {
            if let error = viewModel.error {
                errorDetailsSheet(error: error)
            }
        }
        .sheet(item: $selectedHistorySession) { session in
            historyDetailSheet(session: session)
        }
    }

    @ViewBuilder
    private func errorDetailsSheet(error: APIService.APIError) -> some View {
        NavigationView {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if let code = error.errorCode {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Error Code")
                                .font(AppTheme.Typography.subheadlineFont.weight(.semibold))
                                .foregroundColor(AppTheme.Colors.secondaryText)
                            Text(code)
                                .font(AppTheme.Typography.headingFont)
                                .foregroundColor(AppTheme.Colors.errorRed)
                        }
                    }

                    VStack(alignment: .leading, spacing: 8) {
                        Text("Message")
                            .font(AppTheme.Typography.subheadlineFont.weight(.semibold))
                            .foregroundColor(AppTheme.Colors.secondaryText)
                        Text(error.localizedDescription)
                            .font(AppTheme.Typography.subheadlineFont)
                    }

                    if let details = error.errorDetails {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Details")
                                .font(AppTheme.Typography.subheadlineFont.weight(.semibold))
                                .foregroundColor(AppTheme.Colors.secondaryText)

                            ScrollView {
                                Text(formatErrorDetails(details))
                                    .font(.system(.caption, design: .monospaced))
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                    .padding()
                                    .background(AppTheme.Colors.secondaryBackground)
                                    .cornerRadius(AppTheme.CornerRadius.button)
                            }
                            .frame(maxHeight: 400)
                        }
                    }
                }
                .padding()
            }
            .navigationTitle("Error Details")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") {
                        showErrorDetails = false
                    }
                }
            }
        }
    }

    private func formatErrorDetails(_ details: String) -> String {
        var textToFormat = details

        textToFormat = textToFormat.replacingOccurrences(of: "\\n", with: "\n")
        textToFormat = textToFormat.replacingOccurrences(of: "\\\"", with: "\"")
        textToFormat = textToFormat.replacingOccurrences(of: "\\\\", with: "\\")

        if let jsonData = textToFormat.data(using: .utf8),
            let jsonObject = try? JSONSerialization.jsonObject(with: jsonData, options: []),
            let prettyData = try? JSONSerialization.data(
                withJSONObject: jsonObject, options: [.prettyPrinted]),
            let prettyString = String(data: prettyData, encoding: .utf8)
        {
            return prettyString
        }

        if let jsonArrayRange = findJSONArrayRange(in: textToFormat) {
            let jsonSubstring = String(textToFormat[jsonArrayRange])
            if let jsonData = jsonSubstring.data(using: .utf8),
                let jsonObject = try? JSONSerialization.jsonObject(with: jsonData, options: []),
                let prettyData = try? JSONSerialization.data(
                    withJSONObject: jsonObject, options: [.prettyPrinted]),
                let prettyString = String(data: prettyData, encoding: .utf8)
            {
                let before = String(textToFormat[..<jsonArrayRange.lowerBound])
                let after = String(textToFormat[jsonArrayRange.upperBound...])
                return before + "\n\n" + prettyString + "\n\n" + after
            }
        }

        return textToFormat
    }

    private func findJSONArrayRange(in text: String) -> Range<String.Index>? {
        guard let startIndex = text.range(of: "[{")?.lowerBound else { return nil }

        var bracketCount = 0
        var braceCount = 0
        var inString = false
        var escapeNext = false
        var currentIndex = startIndex

        while currentIndex < text.endIndex {
            let char = text[currentIndex]

            if escapeNext {
                escapeNext = false
                currentIndex = text.index(after: currentIndex)
                continue
            }

            if char == "\\" {
                escapeNext = true
                currentIndex = text.index(after: currentIndex)
                continue
            }

            if char == "\"" {
                inString.toggle()
            } else if !inString {
                if char == "[" {
                    bracketCount += 1
                } else if char == "]" {
                    bracketCount -= 1
                    if bracketCount == 0 && braceCount == 0 {
                        return startIndex..<text.index(after: currentIndex)
                    }
                } else if char == "{" {
                    braceCount += 1
                } else if char == "}" {
                    braceCount -= 1
                }
            }

            currentIndex = text.index(after: currentIndex)
        }

        return nil
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
                    BadgeView(
                        text: sentence.sourceLanguage.uppercased(),
                        color: AppTheme.Colors.primaryBlue)
                    BadgeView(
                        text: "LEVEL \(sentence.languageLevel.uppercased())",
                        color: AppTheme.Colors.primaryBlue)
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
                    .overlay(
                        RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(
                            AppTheme.Colors.borderGray, lineWidth: 1))
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
                        Text("Source:")
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
                .overlay(
                    RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(
                        AppTheme.Colors.primaryBlue.opacity(0.1), lineWidth: 1))
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
                    .overlay(
                        RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button).stroke(
                            AppTheme.Colors.borderGray, lineWidth: 1))
            }

            if let feedback = viewModel.feedback {
                feedbackSection(feedback)
            }

            Button(action: {
                // Force resign focus to ensure binding is updated
                UIApplication.shared.sendAction(
                    #selector(UIResponder.resignFirstResponder), to: nil, from: nil, for: nil)
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
                .background(
                    viewModel.userTranslation.isEmpty ? Color.gray : AppTheme.Colors.primaryBlue
                )
                .foregroundColor(.white)
                .cornerRadius(AppTheme.CornerRadius.button)
                .contentShape(Rectangle())
            }
            .disabled(viewModel.userTranslation.isEmpty || viewModel.isLoading)

            ErrorDisplay(
                error: viewModel.error,
                onDismiss: {
                    viewModel.clearError()
                },
                showDetailsButton: true,
                onShowDetails: {
                    showErrorDetails = true
                }
            )
            if viewModel.error != nil {
                Spacer()
                    .frame(height: AppTheme.Spacing.innerPadding)
            }
        }
        .appCard()
    }

    @ViewBuilder
    private func historyDetailSheet(session: TranslationPracticeSessionResponse) -> some View {
        NavigationView {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    VStack(alignment: .leading, spacing: 12) {
                        let langName =
                            authViewModel.user?.preferredLanguage?.capitalized ?? "Learning"
                        let directionText =
                            session.translationDirection == "learning_to_en"
                            ? "\(langName) → English"
                            : "English → \(langName)"
                        HStack {
                            BadgeView(
                                text: directionText.uppercased(),
                                color: AppTheme.Colors.accentIndigo)
                            Spacer()
                            if let score = session.aiScore {
                                Text("Score: \(Int((score / 5.0) * 100))%")
                                    .font(AppTheme.Typography.captionFont.weight(.bold))
                                    .padding(.horizontal, 8)
                                    .padding(.vertical, 4)
                                    .background(
                                        score >= 4.0
                                            ? AppTheme.Colors.successGreen.opacity(0.1)
                                            : score >= 3.0
                                                ? AppTheme.Colors.primaryBlue.opacity(0.1)
                                                : AppTheme.Colors.errorRed.opacity(0.1)
                                    )
                                    .foregroundColor(
                                        score >= 4.0
                                            ? AppTheme.Colors.successGreen
                                            : score >= 3.0
                                                ? AppTheme.Colors.primaryBlue
                                                : AppTheme.Colors.errorRed
                                    )
                                    .cornerRadius(6)
                            }
                        }

                        VStack(alignment: .leading, spacing: 8) {
                            Text("Original")
                                .font(AppTheme.Typography.captionFont)
                                .foregroundColor(AppTheme.Colors.secondaryText)
                            Text(session.originalSentence)
                                .font(AppTheme.Typography.subheadlineFont)
                        }
                        .padding(AppTheme.Spacing.innerPadding)
                        .background(AppTheme.Colors.secondaryBackground.opacity(0.5))
                        .cornerRadius(10)

                        VStack(alignment: .leading, spacing: 8) {
                            Text("Your Translation")
                                .font(AppTheme.Typography.captionFont)
                                .foregroundColor(AppTheme.Colors.secondaryText)
                            Text(session.userTranslation)
                                .font(AppTheme.Typography.subheadlineFont)
                        }
                        .padding(AppTheme.Spacing.innerPadding)
                        .background(AppTheme.Colors.secondaryBackground.opacity(0.5))
                        .cornerRadius(10)
                    }
                    .padding()

                    feedbackSection(session)
                        .padding(.horizontal)
                }
                .padding(.bottom)
            }
            .navigationTitle("Translation Details")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") {
                        selectedHistorySession = nil
                    }
                }
            }
        }
    }

    private func feedbackSection(_ feedback: TranslationPracticeSessionResponse) -> some View {
        VStack(alignment: .leading, spacing: AppTheme.Spacing.itemSpacing) {
            feedbackHeader(score: feedback.aiScore)
            feedbackContent(feedback.aiFeedback)
        }
        .padding(AppTheme.Spacing.innerPadding)
        .background(AppTheme.Colors.primaryBlue.opacity(0.05))
        .cornerRadius(AppTheme.CornerRadius.button)
        .contentShape(Rectangle())
        .overlay(feedbackOverlay)
    }

    @ViewBuilder
    private func feedbackHeader(score: Float?) -> some View {
        HStack {
            Image(systemName: "sparkles")
                .foregroundColor(AppTheme.Colors.primaryBlue)
            Text("AI Feedback")
                .font(AppTheme.Typography.headingFont)
            Spacer()
            if let score = score {
                scoreBadge(score: score)
            }
        }
    }

    private func scoreBadge(score: Float) -> some View {
        let percentage = Int((score / 5.0) * 100)
        return Text("Score: \(percentage)%")
            .font(AppTheme.Typography.captionFont.weight(.bold))
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(AppTheme.Colors.primaryBlue.opacity(0.1))
            .foregroundColor(AppTheme.Colors.primaryBlue)
            .cornerRadius(6)
    }

    private func feedbackContent(_ markdown: String) -> some View {
        MarkdownTextView(
            markdown: markdown,
            font: UIFont.preferredFont(forTextStyle: .subheadline),
            textColor: UIColor.label
        )
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var feedbackOverlay: some View {
        RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button)
            .stroke(AppTheme.Colors.primaryBlue.opacity(0.2), lineWidth: 1)
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
                    Button(action: {
                        selectedHistorySession = session
                    }) {
                        VStack(alignment: .leading, spacing: 8) {
                            HStack {
                                let langName =
                                    authViewModel.user?.preferredLanguage?.capitalized ?? "Learning"
                                let directionText =
                                    session.translationDirection == "learning_to_en"
                                    ? "\(langName) → English"
                                    : "English → \(langName)"
                                BadgeView(
                                    text: directionText.uppercased(),
                                    color: AppTheme.Colors.accentIndigo)
                                Spacer()
                                if let score = session.aiScore {
                                    Text("\(Int((score / 5.0) * 100))%")
                                        .font(AppTheme.Typography.badgeFont)
                                        .foregroundColor(
                                            score >= 4.0
                                                ? AppTheme.Colors.successGreen
                                                : score >= 3.0
                                                    ? AppTheme.Colors.primaryBlue
                                                    : AppTheme.Colors.errorRed)
                                }
                            }

                            Text(session.originalSentence)
                                .font(AppTheme.Typography.subheadlineFont.weight(.medium))
                                .lineLimit(1)
                                .foregroundColor(.primary)

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
                    .buttonStyle(PlainButtonStyle())
                }
            }
        }
        .appCard()
    }
}
