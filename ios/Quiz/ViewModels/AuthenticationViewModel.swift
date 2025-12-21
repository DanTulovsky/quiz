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

    init(apiService: APIService = APIService.shared) {
        super.init(apiService: apiService)
        // Check auth status on init with a small delay to allow cookies to be set
        // This is especially important after OAuth callbacks
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) { [weak self] in
            guard let strongSelf = self else { return }
            apiService.authStatus()
                .receive(on: DispatchQueue.main)
                .sink(
                    receiveCompletion: { [weak self] completion in
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
                    },
                    receiveValue: { [weak self] response in
                        guard let self = self else { return }
                        self.clearError()
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
                )
                .store(in: &strongSelf.cancellables)
        }
    }

    func login() {
        let loginRequest = LoginRequest(username: username, password: password)
        apiService.login(request: loginRequest)
            .handleErrorOnly(on: self)
            .sink(receiveValue: { [weak self] response in
                guard let self else { return }
                self.clearError()
                self.isAuthenticated = response.success
                self.user = response.user
            })
            .store(in: &cancellables)
    }

    func signup() {
        let signupRequest = UserCreateRequest(username: username, email: email, password: password)
        apiService.signup(request: signupRequest)
            .handleErrorOnly(on: self)
            .sink(receiveValue: { _ in
                // For simplicity, we'll just consider the signup successful
                // and the user can now login.
                // A better approach would be to automatically log the user in.
            })
            .store(in: &cancellables)
    }

    func logout() {
        apiService.logout()
            .handleErrorOnly(on: self)
            .sink(receiveValue: { [weak self] _ in
                guard let self else { return }
                self.clearError()
                self.isAuthenticated = false
                TTSSynthesizerManager.shared.stop()
                TTSSynthesizerManager.shared.preferredVoice = nil
            })
            .store(in: &cancellables)
    }

    func initiateGoogleLogin() {
        // Don't initiate Google login if already authenticated
        guard !isAuthenticated else {
            return
        }

        let publisher = apiService.initiateGoogleLogin()
        publisher
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { [weak self] completion in
                    guard let self = self else { return }
                    switch completion {
                    case .failure(let error):
                        self.error = error
                    case .finished:
                        break
                    }
                },
                receiveValue: { [weak self] response in
                    guard let self = self else { return }
                    // Double-check we're still not authenticated before setting URL
                    guard !self.isAuthenticated else {
                        return
                    }
                    self.error = nil
                    if let url = URL(string: response.authUrl), url.scheme != nil, url.host != nil {
                        self.googleAuthURL = url
                    } else {
                        self.error = .invalidURL
                    }
                }
            )
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

        let publisher = apiService.handleGoogleCallback(code: code, state: state)
        let codeToProcess = code  // Capture code for use in closure
        publisher
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { [weak self] completion in
                    guard let self = self else { return }
                    DispatchQueue.main.async {
                        self.isProcessingGoogleCallback = false
                        switch completion {
                        case .failure(let error):
                            print("❌ Google callback failed: \(error.localizedDescription)")
                            self.error = error
                            self.isAuthenticated = false
                            // Clear googleAuthURL on error to prevent re-triggering
                            self.googleAuthURL = nil
                            // Remove from processed codes on error so it can be retried
                            self.removeProcessedCode(codeToProcess)
                        case .finished:
                            break
                        }
                    }
                },
                receiveValue: { [weak self] response in
                    guard let self = self else { return }
                    DispatchQueue.main.async {
                        self.error = nil
                        self.isAuthenticated = response.success
                        self.user = response.user
                        self.isProcessingGoogleCallback = false
                    }

                    // Clear googleAuthURL immediately after callback to prevent re-triggering
                    // This must happen before any async operations to prevent race conditions
                    self.googleAuthURL = nil

                    // Verify auth status to ensure session cookies are working
                    if response.success {
                        DispatchQueue.main.async {
                            self.apiService.authStatus()
                                .receive(on: DispatchQueue.main)
                                .sink(
                                    receiveCompletion: { completion in
                                        if case .failure(let error) = completion {
                                            print(
                                                "⚠️ Auth status check failed after OAuth: \(error.localizedDescription)"
                                            )
                                        }
                                    },
                                    receiveValue: { [weak self] authResponse in
                                        guard let self = self else { return }
                                        self.isAuthenticated = authResponse.authenticated
                                        self.user = authResponse.user
                                        // Ensure googleAuthURL is still cleared (defensive check)
                                        self.googleAuthURL = nil
                                    }
                                )
                                .store(in: &self.cancellables)
                        }
                    }
                }
            )
            .store(in: &cancellables)
    }

}
