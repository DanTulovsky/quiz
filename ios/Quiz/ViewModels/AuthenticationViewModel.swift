import Foundation
import Combine
import AuthenticationServices

class AuthenticationViewModel: ObservableObject {
    @Published var username = ""
    @Published var password = ""
    @Published var email = ""
    @Published var isAuthenticated = false
    @Published var user: User? = nil
    @Published var error: APIService.APIError?
    @Published var googleAuthURL: URL?

    var apiService: APIService
    private var cancellables = Set<AnyCancellable>()
    private var isProcessingGoogleCallback = false
    private var processedCodes = Set<String>()

    init(apiService: APIService = APIService.shared) {
        self.apiService = apiService
        // Check auth status on init with a small delay to allow cookies to be set
        // This is especially important after OAuth callbacks
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) { [weak self] in
            guard let strongSelf = self else { return }
            apiService.authStatus()
                .receive(on: DispatchQueue.main)
                .sink(receiveCompletion: { [weak self] completion in
                    guard let self = self else { return }
                    switch completion {
                    case .failure(let error):
                        // Only set error if not already authenticated (might be a transient network issue)
                        if !self.isAuthenticated {
                            self.error = error
                        }
                    case .finished:
                        break
                    }
                }, receiveValue: { [weak self] response in
                    guard let self = self else { return }
                    self.error = nil
                    // Only update auth state if not already authenticated
                    // This prevents overriding a successful OAuth login
                    if !self.isAuthenticated {
                        self.isAuthenticated = response.authenticated
                        if response.authenticated {
                            self.user = response.user
                        }
                    } else if response.authenticated {
                        // If already authenticated, just update user info
                        self.user = response.user
                    }
                })
                .store(in: &strongSelf.cancellables)
        }
    }

    func login() {
        let loginRequest = LoginRequest(username: username, password: password)
        apiService.login(request: loginRequest)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                guard let self else { return }
                switch completion {
                case .failure(let error):
                    self.error = error
                    self.isAuthenticated = false
                case .finished:
                    break
                }
            }, receiveValue: { [weak self] response in
                guard let self else { return }
                self.error = nil
                self.isAuthenticated = response.success
                self.user = response.user
            })
            .store(in: &cancellables)
    }

    func signup() {
        let signupRequest = UserCreateRequest(username: username, email: email, password: password)
        apiService.signup(request: signupRequest)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                guard let self else { return }
                switch completion {
                case .failure(let error):
                    self.error = error
                case .finished:
                    break
                }
            }, receiveValue: { _ in
                // For simplicity, we'll just consider the signup successful
                // and the user can now login.
                // A better approach would be to automatically log the user in.
            })
            .store(in: &cancellables)
    }

    func logout() {
        apiService.logout()
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                guard let self else { return }
                switch completion {
                case .failure(let error):
                    self.error = error
                case .finished:
                    break
                }
            }, receiveValue: { [weak self] _ in
                guard let self else { return }
                self.error = nil
                self.isAuthenticated = false
            })
            .store(in: &cancellables)
    }

    func initiateGoogleLogin() {
        // Don't initiate Google login if already authenticated
        guard !isAuthenticated else {
            print("🔒 DEBUG: initiateGoogleLogin() called but already authenticated (isAuthenticated=\(isAuthenticated))")
            return
        }

        print("🔍 DEBUG: initiateGoogleLogin() starting, isAuthenticated=\(isAuthenticated), googleAuthURL=\(googleAuthURL?.absoluteString ?? "nil")")

        let publisher = apiService.initiateGoogleLogin()
        publisher
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                guard let self = self else { return }
                switch completion {
                case .failure(let error):
                    print("❌ DEBUG: initiateGoogleLogin() failed: \(error.localizedDescription)")
                    self.error = error
                case .finished:
                    print("✅ DEBUG: initiateGoogleLogin() publisher finished")
                    break
                }
            }, receiveValue: { [weak self] response in
                guard let self = self else { return }
                // Double-check we're still not authenticated before setting URL
                guard !self.isAuthenticated else {
                    print("⚠️ DEBUG: Authentication state changed during Google login initiation, skipping URL set (isAuthenticated=\(self.isAuthenticated))")
                    return
                }
                self.error = nil
                if let url = URL(string: response.authUrl) {
                    print("🔍 DEBUG: Setting googleAuthURL to: \(url.absoluteString), isAuthenticated=\(self.isAuthenticated)")
                    self.googleAuthURL = url
                    print("🔍 DEBUG: googleAuthURL after setting: \(self.googleAuthURL?.absoluteString ?? "nil")")
                } else {
                    print("❌ DEBUG: Failed to create URL from authUrl: \(response.authUrl)")
                }
            })
            .store(in: &cancellables)
    }

    func handleGoogleCallback(code: String, state: String?) {
        // Guard: Don't process callback if already authenticated
        // This prevents re-processing if the system dialog re-triggers after authentication
        if isAuthenticated {
            print("🔒 DEBUG: Ignoring Google callback - already authenticated")
            googleAuthURL = nil
            return
        }

        // Prevent duplicate processing of the same authorization code
        if isProcessingGoogleCallback || processedCodes.contains(code) {
            print("⚠️ Ignoring duplicate Google callback - code already processed")
            return
        }

        isProcessingGoogleCallback = true
        processedCodes.insert(code)

        print("🔄 Handling Google callback - code: \(code.prefix(10))..., state: \(state ?? "nil")")
        let publisher = apiService.handleGoogleCallback(code: code, state: state)
        let codeToProcess = code // Capture code for use in closure
        publisher
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                guard let self = self else { return }
                self.isProcessingGoogleCallback = false
                switch completion {
                case .failure(let error):
                    print("❌ Google callback failed: \(error.localizedDescription)")
                    self.error = error
                    self.isAuthenticated = false
                    // Clear googleAuthURL on error to prevent re-triggering
                    print("🔍 DEBUG: Clearing googleAuthURL on error, isAuthenticated=\(self.isAuthenticated), current googleAuthURL=\(self.googleAuthURL?.absoluteString ?? "nil")")
                    self.googleAuthURL = nil
                    print("🔍 DEBUG: googleAuthURL after clearing on error: \(self.googleAuthURL?.absoluteString ?? "nil")")
                    // Remove from processed codes on error so it can be retried
                    self.processedCodes.remove(codeToProcess)
                case .finished:
                    print("✅ Google callback completed successfully")
                    break
                }
            }, receiveValue: { [weak self] response in
                guard let self = self else { return }
                print("✅ Google callback response - success: \(response.success), user: \(response.user.username)")
                self.error = nil
                self.isAuthenticated = response.success
                self.user = response.user
                self.isProcessingGoogleCallback = false

                // Clear googleAuthURL immediately after callback to prevent re-triggering
                // This must happen before any async operations to prevent race conditions
                print("🔍 DEBUG: Clearing googleAuthURL after callback, isAuthenticated=\(self.isAuthenticated), current googleAuthURL=\(self.googleAuthURL?.absoluteString ?? "nil")")
                self.googleAuthURL = nil
                print("🔍 DEBUG: googleAuthURL after clearing: \(self.googleAuthURL?.absoluteString ?? "nil")")

                // Verify auth status to ensure session cookies are working
                if response.success {
                    self.apiService.authStatus()
                        .receive(on: DispatchQueue.main)
                        .sink(receiveCompletion: { completion in
                            if case .failure(let error) = completion {
                                print("⚠️ Auth status check failed after OAuth: \(error.localizedDescription)")
                            }
                        }, receiveValue: { [weak self] authResponse in
                            guard let self = self else { return }
                            print("✅ Auth status verified - authenticated: \(authResponse.authenticated)")
                            self.isAuthenticated = authResponse.authenticated
                            self.user = authResponse.user
                            // Ensure googleAuthURL is still cleared (defensive check)
                            print("🔍 DEBUG: Defensive check - clearing googleAuthURL after auth status, current value=\(self.googleAuthURL?.absoluteString ?? "nil")")
                            self.googleAuthURL = nil
                            print("🔍 DEBUG: googleAuthURL after defensive clear: \(self.googleAuthURL?.absoluteString ?? "nil")")
                        })
                        .store(in: &self.cancellables)
                }
            })
            .store(in: &cancellables)
    }
}
