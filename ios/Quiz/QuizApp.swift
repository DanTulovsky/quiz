import SwiftUI
import AVFoundation

@main
struct QuizApp: App {
    @StateObject private var authViewModel = AuthenticationViewModel()

    init() {
        // Configure audio session at app launch, before any audio playback
        // This ensures the session is ready for background playback
        _ = TTSSynthesizerManager.shared
    }

    var body: some Scene {
        WindowGroup {
            MainView()
                .environmentObject(authViewModel)
            // Note: OAuth callbacks are handled by ASWebAuthenticationSession's completion handler
            // in WebAuthView, not here. onOpenURL is kept for other URL scheme handling if needed.
        }
    }
}
