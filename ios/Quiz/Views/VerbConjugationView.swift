import SwiftUI

struct VerbConjugationView: View {
    @StateObject private var viewModel = VerbViewModel()
    @EnvironmentObject var authViewModel: AuthenticationViewModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                // Header
                HStack(alignment: .top) {
                    Image(systemName: "book.closed.fill")
                        .font(.largeTitle)
                        .foregroundColor(AppTheme.Colors.primaryBlue)

                    VStack(alignment: .leading) {
                        Text("Verb Conjugations")
                            .font(AppTheme.Typography.headingFont)
                        Text(
                            "\(authViewModel.user?.preferredLanguage?.capitalized ?? "Italian") "
                                + "verb conjugation tables"
                        )
                        .font(AppTheme.Typography.subheadlineFont)
                        .foregroundColor(AppTheme.Colors.secondaryText)
                    }
                }
                .padding(.horizontal)

                // Stats and Expand All
                HStack {
                    BadgeView(
                        text: "\(viewModel.verbs.count) VERBS", color: AppTheme.Colors.primaryBlue)

                    Spacer()

                    Button(
                        action: {
                            if viewModel.expandedTenses.count
                                == (viewModel.selectedVerbDetail?.tenses.count ?? 0)
                            {
                                viewModel.collapseAll()
                            } else {
                                viewModel.expandAll()
                            }
                        },
                        label: {
                            HStack {
                                Image(
                                    systemName: viewModel.expandedTenses.isEmpty
                                        ? "chevron.down" : "chevron.up")
                                Text(
                                    viewModel.expandedTenses.isEmpty ? "Expand All" : "Collapse All"
                                )
                            }
                            .font(AppTheme.Typography.captionFont)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                            .cornerRadius(AppTheme.CornerRadius.badge)
                        })
                }
                .padding(.horizontal)

                // Verb Picker
                Menu {
                    Picker("Select Verb", selection: $viewModel.selectedVerb) {
                        ForEach(viewModel.verbs, id: \.infinitive) { verb in
                            Text("\(verb.infinitive) (\(verb.infinitiveEn))").tag(verb.infinitive)
                        }
                    }
                } label: {
                    HStack {
                        Text(
                            viewModel.selectedVerb.isEmpty
                                ? "Select a verb" : viewModel.selectedVerb
                        )
                        .foregroundColor(AppTheme.Colors.primaryText)
                        if let verb = viewModel.verbs.first(where: {
                            $0.infinitive == viewModel.selectedVerb
                        }) {
                            Text("(\(verb.infinitiveEn))")
                                .foregroundColor(AppTheme.Colors.secondaryText)
                        }
                        Spacer()
                        Image(systemName: "chevron.up.chevron.down")
                            .font(AppTheme.Typography.captionFont)
                            .foregroundColor(AppTheme.Colors.secondaryText)
                    }
                    .padding()
                    .background(AppTheme.Colors.secondaryBackground)
                    .cornerRadius(AppTheme.CornerRadius.button)
                    .overlay(
                        RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button)
                            .stroke(AppTheme.Colors.borderGray, lineWidth: 1)
                    )
                }
                .padding(.horizontal)

                // Empty state for verbs
                if viewModel.verbs.isEmpty && !viewModel.isLoading {
                    EmptyStateView(
                        icon: "abc",
                        title: "No Verbs Available",
                        message: "Verb conjugations are not available for this language yet."
                    )
                    .padding()
                }

                // Tenses List
                if viewModel.isLoading && viewModel.selectedVerbDetail == nil {
                    ProgressView()
                        .frame(maxWidth: .infinity, minHeight: 200)
                } else if let detail = viewModel.selectedVerbDetail {
                    VStack(spacing: 0) {
                        ForEach(detail.tenses, id: \.tenseId) { tense in
                            TenseAccordionRow(
                                tense: tense,
                                language: authViewModel.user?.preferredLanguage ?? "italian",
                                isExpanded: viewModel.expandedTenses.contains(tense.tenseId)
                            ) {
                                viewModel.toggleTense(tense.tenseId)
                            }
                            Divider()
                        }
                    }
                    .background(AppTheme.Colors.cardBackground)
                    .cornerRadius(AppTheme.CornerRadius.button)
                    .padding(.horizontal)
                }
            }
            .padding(.vertical)
        }
        .onAppear {
            viewModel.fetchLanguages()
            let langName = (authViewModel.user?.preferredLanguage ?? "it")
            // If languages are already loaded, use them; otherwise fetch verbs with fallback
            if !viewModel.availableLanguages.isEmpty {
                let langCode =
                    viewModel.availableLanguages.find(byCodeOrName: langName)?.code ?? "it"
                viewModel.fetchVerbs(language: langCode)
            } else {
                // Fallback to "it" if languages not loaded yet; will be updated in onChange
                viewModel.fetchVerbs(language: "it")
            }
        }
        .onChange(of: viewModel.availableLanguages) { _, languages in
            if !languages.isEmpty {
                let langName = (authViewModel.user?.preferredLanguage ?? "it")
                let langCode = languages.find(byCodeOrName: langName)?.code ?? "it"
                viewModel.fetchVerbs(language: langCode)
            }
        }
        .navigationBarHidden(true)
    }
}

struct TenseAccordionRow: View {
    let tense: Tense
    let language: String
    let isExpanded: Bool
    let action: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            Button(action: action) {
                HStack {
                    Text(tense.tenseName)
                        .font(AppTheme.Typography.headingFont)
                        .foregroundColor(AppTheme.Colors.primaryText)

                    Spacer()

                    BadgeView(
                        text: tense.tenseNameEn.uppercased(), color: AppTheme.Colors.primaryBlue)

                    Image(systemName: "chevron.down")
                        .font(AppTheme.Typography.captionFont)
                        .foregroundColor(AppTheme.Colors.secondaryText)
                        .rotationEffect(.degrees(isExpanded ? 180 : 0))
                }
                .padding()
            }

            if isExpanded {
                VStack(alignment: .leading, spacing: 12) {
                    Text(tense.description)
                        .font(AppTheme.Typography.captionFont)
                        .foregroundColor(AppTheme.Colors.secondaryText)
                        .padding(.horizontal)

                    ForEach(tense.conjugations, id: \.form) { conj in
                        HStack(alignment: .top, spacing: 8) {
                            Text(conj.pronoun)
                                .font(AppTheme.Typography.subheadlineFont)
                                .foregroundColor(AppTheme.Colors.secondaryText)
                                .frame(width: 70, alignment: .leading)

                            VStack(alignment: .leading, spacing: 4) {
                                Text(conj.form)
                                    .font(AppTheme.Typography.subheadlineFont.weight(.bold))
                                    .foregroundColor(AppTheme.Colors.primaryBlue)

                                HStack(alignment: .top, spacing: 8) {
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(conj.exampleSentence)
                                            .font(AppTheme.Typography.captionFont)
                                            .italic()
                                        Text(conj.exampleSentenceEn)
                                            .font(AppTheme.Typography.badgeFont)
                                            .foregroundColor(AppTheme.Colors.secondaryText)
                                    }

                                    TTSButton(text: conj.exampleSentence, language: language)
                                }
                            }
                        }
                        .padding(.horizontal)
                    }
                }
                .padding(.bottom)
                .transition(.opacity)
            }
        }
    }
}
