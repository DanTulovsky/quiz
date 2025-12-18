import SwiftUI

@main
struct QuizApp: App {
    @StateObject private var authViewModel = AuthenticationViewModel()

    var body: some Scene {
        WindowGroup {
            MainView()
                .environmentObject(authViewModel)
            // Note: OAuth callbacks are handled by ASWebAuthenticationSession's completion handler
            // in WebAuthView, not here. onOpenURL is kept for other URL scheme handling if needed.
        }
    }
}
