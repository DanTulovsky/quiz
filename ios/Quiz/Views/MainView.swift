import SwiftUI

struct MainView: View {
    @AppStorage("app_theme") private var appTheme: String = "system"
    @AppStorage("app_font_size") private var appFontSize: String = "M"
    @EnvironmentObject var authViewModel: AuthenticationViewModel

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

                    // Section 3: AI History
                    NavigationView {
                        List {
                            Section("AI History") {
                                NavigationLink(destination: AIConversationListView()) {
                                    Label("AI Conversations", systemImage: "bubble.left.and.bubble.right")
                                }
                                NavigationLink(destination: BookmarkedMessagesView()) {
                                    Label("Bookmarked Messages", systemImage: "bookmark")
                                }
                            }
                        }
                        .navigationTitle("AI History")
                    }
                    .tabItem {
                        Image(systemName: "clock.arrow.circlepath")
                        Text("History")
                    }

                    // Section 4: Reference
                    NavigationView {
                        List {
                            Section("Reference") {
                                NavigationLink(destination: SnippetListView()) {
                                    Label("Snippets", systemImage: "text.quote")
                                }
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
    }
}
