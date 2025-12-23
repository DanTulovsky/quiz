import Foundation
import UserNotifications
import Combine
import UIKit

class NotificationService: NSObject, ObservableObject {
    static let shared = NotificationService()

    @Published var isAuthorized = false
    @Published var deviceToken: String?

    private var apiService: APIService?
    private var cancellables = Set<AnyCancellable>()

    private override init() {
        super.init()
        UNUserNotificationCenter.current().delegate = self
    }

    func configure(apiService: APIService) {
        self.apiService = apiService
    }

    func checkAuthorizationStatus() {
        UNUserNotificationCenter.current().getNotificationSettings { [weak self] settings in
            DispatchQueue.main.async {
                self?.isAuthorized = settings.authorizationStatus == .authorized
            }
        }
    }

    func requestAuthorization() {
        UNUserNotificationCenter.current().requestAuthorization(
            options: [.alert, .sound, .badge]
        ) { [weak self] granted, error in
            DispatchQueue.main.async {
                self?.isAuthorized = granted
                if granted {
                    self?.registerForRemoteNotifications()
                } else if let error = error {
                    print("❌ Failed to request notification authorization: \(error.localizedDescription)")
                }
            }
        }
    }

    func registerForRemoteNotifications() {
        DispatchQueue.main.async {
            UIApplication.shared.registerForRemoteNotifications()
        }
    }

    func registerExistingDeviceToken() {
        guard let token = deviceToken, !token.isEmpty, let apiService = apiService else {
            print("❌ Cannot register device token: token is empty or API service not configured")
            return
        }

        apiService.registerDeviceToken(deviceToken: token)
            .sink(
                receiveCompletion: { completion in
                    if case .failure(let error) = completion {
                        print("❌ Failed to register device token: \(error.localizedDescription)")
                    }
                },
                receiveValue: { _ in
                    print("✅ Device token registered successfully")
                }
            )
            .store(in: &cancellables)
    }

    func didRegisterForRemoteNotifications(deviceToken: Data) {
        let tokenParts = deviceToken.map { data in String(format: "%02.2hhx", data) }
        let token = tokenParts.joined()

        // Validate token is not empty
        guard !token.isEmpty else {
            print("❌ Device token is empty")
            return
        }

        DispatchQueue.main.async {
            self.deviceToken = token
        }

        // Send device token to backend
        if let apiService = self.apiService {
            apiService.registerDeviceToken(deviceToken: token)
                .sink(
                    receiveCompletion: { completion in
                        if case .failure(let error) = completion {
                            print("❌ Failed to register device token: \(error.localizedDescription)")
                        }
                    },
                    receiveValue: { _ in
                        print("✅ Device token registered successfully")
                    }
                )
                .store(in: &self.cancellables)
        } else {
            print("❌ APIService not configured in NotificationService")
        }
    }

    func didFailToRegisterForRemoteNotifications(error: Error) {
        print("❌ Failed to register for remote notifications: \(error.localizedDescription)")
    }
}

extension NotificationService: UNUserNotificationCenterDelegate {
    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        // Show notification even when app is in foreground
        // Use .list in addition to .banner for better compatibility across iOS versions
        if #available(iOS 14.0, *) {
            completionHandler([.banner, .list, .sound, .badge])
        } else {
            completionHandler([.alert, .sound, .badge])
        }
    }

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse,
        withCompletionHandler completionHandler: @escaping () -> Void
    ) {
        let userInfo = response.notification.request.content.userInfo

        // Handle deep link
        if let deepLink = userInfo["deep_link"] as? String {
            handleDeepLink(deepLink)
        }

        completionHandler()
    }

    private func handleDeepLink(_ deepLink: String) {
        // Post notification to handle deep link navigation
        NotificationCenter.default.post(
            name: NSNotification.Name("HandleDeepLink"),
            object: nil,
            userInfo: ["deep_link": deepLink]
        )
    }
}
