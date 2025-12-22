import AuthenticationServices
import Combine
import Foundation

class AuthenticationViewModel: BaseViewModel {
    @Published var username = ""
    @Published var password = ""
    @Published var email = ""
    @Published var isAuthenticated = false
    @Published var user: User? = nil
    @Published var googleAuthURL: URL?
    private let stateQueue = DispatchQueue(
        label: "com.wetsnow.quiz.auth.state", attributes: .concurrent)
    private var _isProcessingGoogleCallback = false
    private var _processedCodes = Set<String>()

    private var isProcessingGoogleCallback: Bool {
        get {
            return stateQueue.sync { _isProcessingGoogleCallback }
        }
        set {
            stateQueue.async(flags: .barrier) {
                self._isProcessingGoogleCallback = newValue
            }
        }
    }

    private var processedCodes: Set<String> {
        get {
            return stateQueue.sync { _processedCodes }
        }
        set {
            stateQueue.async(flags: .barrier) {
                self._processedCodes = newValue
            }
        }
    }

    private func containsProcessedCode(_ code: String) -> Bool {
        return stateQueue.sync { _processedCodes.contains(code) }
    }

    private func insertProcessedCode(_ code: String) {
        stateQueue.async(flags: .barrier) {
            self._processedCodes.insert(code)
        }
    }

    private func removeProcessedCode(_ code: String) {
        stateQueue.async(flags: .barrier) {
            self._processedCodes.remove(code)
        }
    }

    override init(apiService: APIService = APIService.shared) {
        super.init(apiService: apiService)
        // Check auth status on init with a small delay to allow cookies to be set
        // This is especially important after OAuth callbacks
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) { [weak self] in
            guard let self = self else { return }
            apiService.authStatus()
                .handleErrorOnly(on: self)
                .sinkValue(on: self) { [weak self] response in
                    guard let self = self else { return }
                    // Only set error if not already authenticated (might be a transient network issue)
                    if self.error != nil && !self.isAuthenticated {
                        // Error was set by handleErrorOnly, keep it
                    } else {
                        self.clearError()
                    }
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
                }
                .store(in: &self.cancellables)
        }
    }

    func login() {
        let loginRequest = LoginRequest(username: username, password: password)
        apiService.login(request: loginRequest)
            .handleErrorOnly(on: self)
            .sinkValue(on: self) { [weak self] response in
                guard let self else { return }
                self.clearError()
                self.isAuthenticated = response.success
                self.user = response.user
            }
            .store(in: &cancellables)
    }

    func signup() {
        let signupRequest = UserCreateRequest(username: username, email: email, password: password)
        apiService.signup(request: signupRequest)
            .handleErrorOnly(on: self)
            .sinkVoid(on: self) {
                // For simplicity, we'll just consider the signup successful
                // and the user can now login.
                // A better approach would be to automatically log the user in.
            }
            .store(in: &cancellables)
    }

    func logout() {
        apiService.logout()
            .handleErrorOnly(on: self)
            .sinkVoid(on: self) { [weak self] in
                guard let self else { return }
                self.clearError()
                self.isAuthenticated = false
                TTSSynthesizerManager.shared.stop()
                TTSSynthesizerManager.shared.preferredVoice = nil
            }
            .store(in: &cancellables)
    }

    func initiateGoogleLogin() {
        // Don't initiate Google login if already authenticated
        guard !isAuthenticated else {
            return
        }

        apiService.initiateGoogleLogin()
            .handleErrorOnly(on: self)
            .sinkValue(on: self) { [weak self] response in
                guard let self = self else { return }
                // Double-check we're still not authenticated before setting URL
                guard !self.isAuthenticated else {
                    return
                }
                self.clearError()
                if let url = URL(string: response.authUrl), url.scheme != nil, url.host != nil {
                    self.googleAuthURL = url
                } else {
                    self.error = .invalidURL
                }
            }
            .store(in: &cancellables)
    }

    func handleGoogleCallback(code: String, state: String?) {
        // Guard: Don't process callback if already authenticated
        // This prevents re-processing if the system dialog re-triggers after authentication
        if isAuthenticated {
            googleAuthURL = nil
            return
        }

        // Prevent duplicate processing of the same authorization code (thread-safe check)
        if isProcessingGoogleCallback || containsProcessedCode(code) {
            return
        }

        isProcessingGoogleCallback = true
        insertProcessedCode(code)

        let codeToProcess = code  // Capture code for use in closure
        apiService.handleGoogleCallback(code: code, state: state)
            .handleErrorOnly(on: self)
            .sink(
                receiveCompletion: { [weak self] completion in
                    guard let self = self else { return }
                    self.isProcessingGoogleCallback = false
                    if case .failure(let error) = completion {
                        print("❌ Google callback failed: \(error.localizedDescription)")
                        self.isAuthenticated = false
                        // Clear googleAuthURL on error to prevent re-triggering
                        self.googleAuthURL = nil
                        // Remove from processed codes on error so it can be retried
                        self.removeProcessedCode(codeToProcess)
                    }
                },
                receiveValue: { [weak self] response in
                    guard let self = self else { return }
                    self.clearError()
                    self.isAuthenticated = response.success
                    self.user = response.user
                    self.isProcessingGoogleCallback = false

                    // Clear googleAuthURL immediately after callback to prevent re-triggering
                    // This must happen before any async operations to prevent race conditions
                    self.googleAuthURL = nil

                    // Verify auth status to ensure session cookies are working
                    if response.success {
                        self.apiService.authStatus()
                            .handleErrorOnly(on: self)
                            .sinkValue(on: self) { [weak self] authResponse in
                                guard let self = self else { return }
                                self.isAuthenticated = authResponse.authenticated
                                self.user = authResponse.user
                                // Ensure googleAuthURL is still cleared (defensive check)
                                self.googleAuthURL = nil
                            }
                            .store(in: &self.cancellables)
                    }
                }
            )
            .store(in: &cancellables)
    }

}
