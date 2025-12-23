import SwiftUI
import AVFoundation
import UserNotifications
import UIKit

class AppDelegate: NSObject, UIApplicationDelegate {
    func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil
    ) -> Bool {
        return true
    }

    func application(
        _ application: UIApplication,
        didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data
    ) {
        NotificationService.shared.didRegisterForRemoteNotifications(deviceToken: deviceToken)
    }

    func application(
        _ application: UIApplication,
        didFailToRegisterForRemoteNotificationsWithError error: Error
    ) {
        NotificationService.shared.didFailToRegisterForRemoteNotifications(error: error)
    }
}

@main
struct QuizApp: App {
    @StateObject private var authViewModel = AuthenticationViewModel()
    @StateObject private var notificationService = NotificationService.shared
    @UIApplicationDelegateAdaptor(AppDelegate.self) var appDelegate

    init() {
        // Configure audio session at app launch, before any audio playback
        // This ensures the session is ready for background playback
        _ = TTSSynthesizerManager.shared

        // Configure notification service
        NotificationService.shared.configure(apiService: APIService.shared)
    }

    var body: some Scene {
        WindowGroup {
            MainView()
                .environmentObject(authViewModel)
                .environmentObject(notificationService)
                .onAppear {
                    // Request notification permissions when app appears
                    if authViewModel.isAuthenticated {
                        NotificationService.shared.requestAuthorization()
                    }
                }
                .onChange(of: authViewModel.isAuthenticated) { _, isAuthenticated in
                    if isAuthenticated {
                        NotificationService.shared.requestAuthorization()
                    }
                }
                .onOpenURL { url in
                    handleDeepLink(url: url)
                }
            // Note: OAuth callbacks are handled by ASWebAuthenticationSession's completion handler
            // in WebAuthView, not here. onOpenURL is kept for other URL scheme handling if needed.
        }
    }

    private func handleDeepLink(url: URL) {
        guard url.scheme == "com.wetsnow.quiz" else { return }

        let deepLink = url.host ?? ""
        NotificationCenter.default.post(
            name: NSNotification.Name("HandleDeepLink"),
            object: nil,
            userInfo: ["deep_link": deepLink]
        )
    }
}
