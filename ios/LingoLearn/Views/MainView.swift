import SwiftUI

struct MainView: View {
    @StateObject private var authViewModel = AuthenticationViewModel()

    var body: some View {
        if authViewModel.isAuthenticated {
            TabView {
                QuizView()
                    .tabItem {
                        Image(systemName: "house")
                        Text("Home")
                    }
                                NavigationView {
                                    List {
                                        NavigationLink("Stories", destination: StoryListView())
                                        NavigationLink("Translation Practice", destination: TranslationPracticeView())
                                        NavigationLink("Verb Conjugation", destination: VerbConjugationView())
                                    }
                                    .navigationTitle("Learn")
                                }
                                .tabItem {
                                    Image(systemName: "book")
                                    Text("Learn")
                                }                                                    VocabularyView()
                                                        .tabItem {
                                                            Image(systemName: "list.bullet")
                                                            Text("Vocabulary")
                                                        }
                                                    PhrasebookView()
                                                        .tabItem {
                                                            Image(systemName: "text.book.closed")
                                                            Text("Phrasebook")
                                                        }
                                                                    NavigationView {
                                                                        VStack {
                                                                            SettingsView()
                                                                            Button("Logout") {
                                                                                authViewModel.logout()
                                                                            }
                                                                            .padding()
                                                                            .background(Color.red)
                                                                            .foregroundColor(.white)
                                                                            .cornerRadius(8)
                                                                        }
                                                                    }
                                                                    .tabItem {
                                                                        Image(systemName: "person")
                                                                        Text("Profile")
                                                                    }
            }
        } else {
            LoginView()
        }
    }
}
