import SwiftUI
import Combine

class TTSInitializationManager: ObservableObject {
    private var cancellables = Set<AnyCancellable>()
    var isInitialized = false
    private var loadedLanguages: [LanguageInfo] = []

    func initialize(apiService: APIService, userLanguage: String?) {
        guard !isInitialized else { return }
        isInitialized = true

        // First, load languages to populate the default voice cache
        apiService.getLanguages()
            .receive(on: DispatchQueue.main)
            .flatMap { [weak self] languages -> AnyPublisher<UserLearningPreferences, APIService.APIError> in
                guard let self = self else {
                    return Fail(error: APIService.APIError.invalidResponse).eraseToAnyPublisher()
                }
                // Store languages for later use
                self.loadedLanguages = languages
                // Update the default voice cache (we're on main thread from receive(on:))
                TTSSynthesizerManager.shared.updateDefaultVoiceCache(languages: languages)

                // Then load user preferences
                return apiService.getLearningPreferences()
            }
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { completion in
                    if case .failure(let error) = completion {
                        print("Failed to initialize TTS settings: \(error.localizedDescription)")
                    }
                },
                receiveValue: { [weak self] preferences in
                    guard let self = self else { return }
                    // Set the preferred voice from user preferences (we're on main thread from receive(on:))
                    if let savedVoice = preferences.ttsVoice, !savedVoice.isEmpty {
                        TTSSynthesizerManager.shared.preferredVoice = savedVoice
                    } else if let userLanguage = userLanguage {
                        // If no saved voice, find the default voice for the user's language
                        let langKey = userLanguage.lowercased()
                        if let languageInfo = self.loadedLanguages.first(where: {
                            $0.name.lowercased() == langKey || $0.code.lowercased() == langKey
                        }), let defaultVoice = languageInfo.ttsVoice {
                            TTSSynthesizerManager.shared.preferredVoice = defaultVoice
                        }
                    }
                }
            )
            .store(in: &cancellables)
    }
}

struct MainView: View {
    @AppStorage("app_theme") private var appTheme: String = "system"
    @AppStorage("app_font_size") private var appFontSize: String = "M"
    @EnvironmentObject var authViewModel: AuthenticationViewModel
    @StateObject private var ttsInitManager = TTSInitializationManager()

    private var colorScheme: ColorScheme? {
        switch appTheme {
        case "light": return .light
        case "dark": return .dark
        default: return nil
        }
    }

    private var dynamicTypeSize: DynamicTypeSize {
        switch appFontSize {
        case "S": return .small
        case "M": return .medium
        case "L": return .large
        case "XL": return .xLarge
        default: return .medium
        }
    }

    var body: some View {
        Group {
            if authViewModel.isAuthenticated {
                TabView {
                    // Section 1: Menu
                    NavigationView {
                        List {
                            Section("Menu") {
                                NavigationLink(destination: QuizView()) {
                                    Label("Quiz", systemImage: "questionmark.circle")
                                }
                                NavigationLink(destination: QuizView(questionType: "vocabulary")) {
                                    Label("Vocabulary", systemImage: "text.book.closed")
                                }
                                NavigationLink(destination: QuizView(questionType: "reading_comprehension")) {
                                    Label("Reading", systemImage: "doc.text")
                                }
                                NavigationLink(destination: StoryListView()) {
                                    Label("Story", systemImage: "book")
                                }
                            }
                        }
                        .navigationTitle("Quiz")
                    }
                    .tabItem {
                        Image(systemName: "house")
                        Text("Home")
                    }

                    // Section 2: Practice
                    NavigationView {
                        List {
                            Section("Practice") {
                                NavigationLink(destination: DailyView()) {
                                    Label("Daily", systemImage: "calendar")
                                }
                                NavigationLink(destination: WordOfTheDayView()) {
                                    Label("Word of the Day", systemImage: "sparkles")
                                }
                                NavigationLink(destination: TranslationPracticeView()) {
                                    Label("Translation Practice", systemImage: "arrow.left.and.right")
                                }
                            }
                        }
                        .navigationTitle("Practice")
                    }
                    .tabItem {
                        Image(systemName: "checkmark.circle")
                        Text("Practice")
                    }

                    // Section 3: History
                    NavigationView {
                        List {
                            Section("History") {
                                NavigationLink(destination: AIConversationListView()) {
                                    Label("AI Conversations", systemImage: "bubble.left.and.bubble.right")
                                }
                                NavigationLink(destination: BookmarkedMessagesView()) {
                                    Label("Bookmarked Messages", systemImage: "bookmark")
                                }
                                NavigationLink(destination: SnippetListView()) {
                                    Label("Snippets", systemImage: "text.quote")
                                }
                            }
                        }
                        .navigationTitle("History")
                    }
                    .tabItem {
                        Image(systemName: "clock.arrow.circlepath")
                        Text("History")
                    }

                    // Section 4: Reference
                    NavigationView {
                        List {
                            Section("Reference") {
                                NavigationLink(destination: PhrasebookView()) {
                                    Label("Phrasebook", systemImage: "character.book.closed")
                                }
                                NavigationLink(destination: VerbConjugationView()) {
                                    Label("Verb Conjugations", systemImage: "abc")
                                }
                            }
                        }
                        .navigationTitle("Reference")
                    }
                    .tabItem {
                        Image(systemName: "info.circle")
                        Text("Reference")
                    }

                    // Section 5: Profile
                    NavigationView {
                        SettingsView()
                            .navigationTitle("Profile")
                    }
                    .tabItem {
                        Image(systemName: "person")
                        Text("Profile")
                    }
                }
                .environmentObject(authViewModel)
            } else {
                LoginView()
                    .environmentObject(authViewModel)
            }
        }
        .preferredColorScheme(colorScheme)
        .environment(\.dynamicTypeSize, dynamicTypeSize)
        .onChange(of: authViewModel.isAuthenticated) { _, isAuthenticated in
            if isAuthenticated && !ttsInitManager.isInitialized {
                ttsInitManager.initialize(
                    apiService: APIService.shared,
                    userLanguage: authViewModel.user?.preferredLanguage
                )
            }
        }
        .onAppear {
            if authViewModel.isAuthenticated && !ttsInitManager.isInitialized {
                ttsInitManager.initialize(
                    apiService: APIService.shared,
                    userLanguage: authViewModel.user?.preferredLanguage
                )
            }
        }
    }
}
