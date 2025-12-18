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
                        .foregroundColor(.blue)

                    VStack(alignment: .leading) {
                        Text("Verb Conjugations")
                            .font(.title)
                            .bold()
                        Text("\(authViewModel.user?.preferredLanguage?.capitalized ?? "Italian") verb conjugation tables")
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                    }
                }
                .padding(.horizontal)

                // Stats and Expand All
                HStack {
                    BadgeView(text: "\(viewModel.verbs.count) VERBS", color: .blue)

                    Spacer()

                    Button(action: {
                        if viewModel.expandedTenses.count == (viewModel.selectedVerbDetail?.tenses.count ?? 0) {
                            viewModel.collapseAll()
                        } else {
                            viewModel.expandAll()
                        }
                    }) {
                        HStack {
                            Image(systemName: viewModel.expandedTenses.isEmpty ? "chevron.down" : "chevron.up")
                            Text(viewModel.expandedTenses.isEmpty ? "Expand All" : "Collapse All")
                        }
                        .font(.caption)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 6)
                        .background(Color.blue.opacity(0.1))
                        .cornerRadius(8)
                    }
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
                        Text(viewModel.selectedVerb.isEmpty ? "Select a verb" : viewModel.selectedVerb)
                            .foregroundColor(.primary)
                        if let v = viewModel.verbs.first(where: { $0.infinitive == viewModel.selectedVerb }) {
                            Text("(\(v.infinitiveEn))")
                                .foregroundColor(.secondary)
                        }
                        Spacer()
                        Image(systemName: "chevron.up.chevron.down")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                    .padding()
                    .background(Color(.secondarySystemBackground))
                    .cornerRadius(10)
                    .overlay(
                        RoundedRectangle(cornerRadius: 10)
                            .stroke(Color.gray.opacity(0.2), lineWidth: 1)
                    )
                }
                .padding(.horizontal)

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
                    .background(Color(.systemBackground))
                    .cornerRadius(10)
                    .padding(.horizontal)
                }
            }
            .padding(.vertical)
        }
        .onAppear {
            let langName = (authViewModel.user?.preferredLanguage ?? "it")
            let lang = Language(rawValue: langName)?.code ?? "it"
            viewModel.fetchVerbs(language: lang)
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
                        .font(.headline)
                        .foregroundColor(.primary)

                    Spacer()

                    BadgeView(text: tense.tenseNameEn.uppercased(), color: .blue)

                    Image(systemName: "chevron.down")
                        .font(.caption)
                        .foregroundColor(.secondary)
                        .rotationEffect(.degrees(isExpanded ? 180 : 0))
                }
                .padding()
            }

            if isExpanded {
                VStack(alignment: .leading, spacing: 12) {
                    Text(tense.description)
                        .font(.caption)
                        .foregroundColor(.secondary)
                        .padding(.horizontal)

                    ForEach(tense.conjugations, id: \.form) { conj in
                        HStack(alignment: .top, spacing: 8) {
                            Text(conj.pronoun)
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                                .frame(width: 70, alignment: .leading)

                            VStack(alignment: .leading, spacing: 4) {
                                Text(conj.form)
                                    .font(.subheadline)
                                    .bold()
                                    .foregroundColor(.blue)

                                HStack(alignment: .top, spacing: 8) {
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(conj.exampleSentence)
                                            .font(.caption)
                                            .italic()
                                        Text(conj.exampleSentenceEn)
                                            .font(.caption2)
                                            .foregroundColor(.secondary)
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
