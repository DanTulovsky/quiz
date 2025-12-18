import AuthenticationServices
import SafariServices
import SwiftUI
import Combine

struct LoginView: View {
    @EnvironmentObject var viewModel: AuthenticationViewModel
    @State private var showPassword = false
    @State private var isLoading = false
    @State private var showWebAuth = false

    var body: some View {
        NavigationView {
            ScrollView {
                VStack(spacing: 24) {
                    Spacer().frame(height: 40)

                    ZStack {
                        Circle()
                            .fill(Color.blue.opacity(0.1))
                            .frame(width: 80, height: 80)
                        Image(systemName: "brain")
                            .font(.system(size: 40))
                            .foregroundColor(.blue)
                    }

                    VStack(spacing: 8) {
                        Text("Language Quiz")
                            .font(.largeTitle)
                            .fontWeight(.bold)

                        Text("Sign in to start learning")
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                    }

                    VStack(alignment: .leading, spacing: 24) {
                        VStack(alignment: .leading, spacing: 8) {
                            HStack(spacing: 4) {
                                Text("Username")
                                    .font(.subheadline)
                                    .fontWeight(.medium)
                                Text("*")
                                    .foregroundColor(.red)
                            }

                            TextField("admin", text: $viewModel.username)
                                .textInputAutocapitalization(.never)
                                .autocorrectionDisabled(true)
                                .padding()
                                .background(AppTheme.Colors.secondaryBackground)
                                .cornerRadius(AppTheme.CornerRadius.button)
                        }

                        VStack(alignment: .leading, spacing: 8) {
                            HStack(spacing: 4) {
                                Text("Password")
                                    .font(.subheadline)
                                    .fontWeight(.medium)
                                Text("*")
                                    .foregroundColor(.red)
                            }

                            HStack {
                                if showPassword {
                                    TextField("", text: $viewModel.password)
                                        .textInputAutocapitalization(.never)
                                        .autocorrectionDisabled(true)
                                } else {
                                    SecureField("••••••••", text: $viewModel.password)
                                        .textInputAutocapitalization(.never)
                                        .autocorrectionDisabled(true)
                                }

                                Button(action: { showPassword.toggle() }) {
                                    Image(systemName: showPassword ? "eye.slash" : "eye")
                                        .foregroundColor(.secondary)
                                }
                            }
                            .padding()
                            .background(AppTheme.Colors.secondaryBackground)
                            .cornerRadius(AppTheme.CornerRadius.button)
                        }

                        Button(action: {
                            isLoading = true
                            viewModel.login()
                            DispatchQueue.main.asyncAfter(deadline: .now() + 1) {
                                isLoading = false
                            }
                        }) {
                            HStack {
                                if isLoading {
                                    ProgressView()
                                        .progressViewStyle(CircularProgressViewStyle(tint: .white))
                                } else {
                                    Text("Sign In")
                                        .fontWeight(.semibold)
                                }
                            }
                            .frame(maxWidth: .infinity)
                            .padding()
                            .background(
                                isLoading || viewModel.username.isEmpty
                                    || viewModel.password.isEmpty
                                    ? Color.gray : AppTheme.Colors.primaryBlue
                            )
                            .foregroundColor(.white)
                            .cornerRadius(AppTheme.CornerRadius.button)
                        }
                        .disabled(
                            isLoading || viewModel.username.isEmpty || viewModel.password.isEmpty)

                        if viewModel.error != nil {
                            Text("Login failed. Please check your credentials.")
                                .foregroundColor(AppTheme.Colors.errorRed)
                                .font(AppTheme.Typography.captionFont)
                                .padding()
                                .background(AppTheme.Colors.errorRed.opacity(0.1))
                                .cornerRadius(AppTheme.CornerRadius.button)
                        }

                        HStack {
                            Text("Don't have an account?")
                                .foregroundColor(.secondary)
                            NavigationLink("Sign up here", destination: SignupView())
                                .fontWeight(.medium)
                        }
                        .font(.subheadline)
                        .frame(maxWidth: .infinity)

                        HStack {
                            VStack { Divider() }
                            Text("or")
                                .foregroundColor(.secondary)
                                .padding(.horizontal, 8)
                            VStack { Divider() }
                        }

                        Button(action: {
                            // Don't allow Google login if already authenticated
                            guard !viewModel.isAuthenticated else {
                                return
                            }
                            isLoading = true
                            viewModel.initiateGoogleLogin()
                        }) {
                            HStack {
                                Image(systemName: "globe")
                                    .foregroundColor(.secondary)
                                Text("Sign in with Google")
                                    .foregroundColor(.secondary)
                                    .fontWeight(.medium)
                            }
                            .frame(maxWidth: .infinity)
                            .padding()
                            .background(AppTheme.Colors.cardBackground)
                            .overlay(
                                RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button)
                                    .stroke(AppTheme.Colors.borderGray, lineWidth: 1)
                            )
                            .cornerRadius(AppTheme.CornerRadius.button)
                        }
                    }
                    .padding(.horizontal, 24)

                    Spacer()
                }
            }
            .navigationBarHidden(true)
            .onChange(of: viewModel.googleAuthURL) { newURL in
                print("🔍 DEBUG: onChange(googleAuthURL) fired: newURL=\(newURL?.absoluteString ?? "nil"), isAuthenticated=\(viewModel.isAuthenticated), showWebAuth=\(showWebAuth)")
                // Only show WebAuthView if URL is set, user is not already authenticated, and sheet is not already showing
                if newURL != nil && !viewModel.isAuthenticated && !showWebAuth {
                    print("🔍 DEBUG: Showing WebAuthView sheet")
                    isLoading = false
                    showWebAuth = true
                } else if newURL == nil && showWebAuth {
                    // If URL is cleared while sheet is showing, dismiss it
                    print("🔍 DEBUG: Dismissing WebAuthView sheet because URL was cleared")
                    showWebAuth = false
                } else {
                    print("🔍 DEBUG: Not showing WebAuthView - newURL=\(newURL != nil), isAuthenticated=\(viewModel.isAuthenticated), showWebAuth=\(showWebAuth)")
                }
            }
            .onReceive(viewModel.$error) { error in
                if error != nil {
                    isLoading = false
                }
                        }
                    .sheet(isPresented: Binding(
                        get: {
                            let shouldShow = showWebAuth && !viewModel.isAuthenticated
                            if shouldShow != showWebAuth {
                                print("🔍 DEBUG: sheet getter - shouldShow changed: \(showWebAuth) -> \(shouldShow), isAuthenticated=\(viewModel.isAuthenticated)")
                            }
                            return shouldShow
                        },
                        set: { newValue in
                            print("🔍 DEBUG: sheet(isPresented) setter called: \(newValue), googleAuthURL=\(viewModel.googleAuthURL?.absoluteString ?? "nil"), isAuthenticated=\(viewModel.isAuthenticated)")
                            // If authenticated, force dismiss
                            if viewModel.isAuthenticated {
                                print("🔒 DEBUG: Forcing sheet dismiss because authenticated")
                                showWebAuth = false
                            } else {
                                showWebAuth = newValue
                            }
                        }
                    )) {
                // Only show WebAuthView if URL is set and user is not authenticated
                if let url = viewModel.googleAuthURL, !viewModel.isAuthenticated {
                    WebAuthView(
                        url: url,
                        onCallback: { components in
                            print("📞 Callback received with components: \(components)")

                            // Guard: Don't process callback if already authenticated
                            // This prevents re-processing if the system dialog re-triggers after authentication
                            if viewModel.isAuthenticated {
                                print("🔒 DEBUG: Ignoring callback - already authenticated")
                                showWebAuth = false
                                viewModel.googleAuthURL = nil
                                return
                            }

                            // Check for OAuth error first
                            if let error = components.queryItems?.first(where: { $0.name == "error" })?.value {
                                print("❌ OAuth error in callback: \(error)")
                                isLoading = false
                                showWebAuth = false
                                viewModel.googleAuthURL = nil
                                // Error will be shown via viewModel.error
                                return
                            }

                            // Check for authorization code
                            if let code = components.queryItems?.first(where: { $0.name == "code" }
                            )?.value {
                                let state = components.queryItems?.first(where: {
                                    $0.name == "state"
                                })?.value
                                print("✅ Code found: \(code.prefix(10))..., state: \(state ?? "nil")")

                                // Immediately dismiss sheet and clear URL to prevent re-triggering
                                // Do this synchronously to prevent any view updates
                                viewModel.googleAuthURL = nil
                                showWebAuth = false

                                // Process callback after a tiny delay to ensure sheet is dismissed
                                DispatchQueue.main.asyncAfter(deadline: .now() + 0.05) {
                                    isLoading = true
                                    viewModel.handleGoogleCallback(code: code, state: state)
                                }
                            } else {
                                // No code found - might be an error or incomplete callback
                                print("⚠️ No code found in callback")
                                isLoading = false
                                showWebAuth = false
                                viewModel.googleAuthURL = nil
                            }
                        },
                        onDismiss: {
                            print("🔍 DEBUG: WebAuthView onDismiss called, isAuthenticated=\(viewModel.isAuthenticated)")
                            isLoading = false
                            showWebAuth = false
                            // Clear googleAuthURL when WebAuthView is dismissed to prevent re-triggering
                            print("🔍 DEBUG: Clearing googleAuthURL in onDismiss, current value=\(viewModel.googleAuthURL?.absoluteString ?? "nil")")
                            viewModel.googleAuthURL = nil
                            print("🔍 DEBUG: googleAuthURL after onDismiss clear: \(viewModel.googleAuthURL?.absoluteString ?? "nil")")
                        },
                        viewModel: viewModel)
                } else {
                    EmptyView()
                        .onAppear {
                            print("🔍 DEBUG: Not creating WebAuthView - url=\(viewModel.googleAuthURL?.absoluteString ?? "nil"), isAuthenticated=\(viewModel.isAuthenticated)")
                        }
                }
            }
        }
    }
}

struct WebAuthView: UIViewControllerRepresentable {
    let url: URL
    let onCallback: (URLComponents) -> Void
    let onDismiss: () -> Void
    @ObservedObject var viewModel: AuthenticationViewModel

    func makeUIViewController(context: Context) -> UIViewController {
        print("🔍 DEBUG: makeUIViewController called - isAuthenticated=\(viewModel.isAuthenticated), existing session=\(context.coordinator.session != nil)")
        let viewController = UIViewController()
        viewController.view.backgroundColor = .clear

        // Guard: Don't start session if already authenticated
        // This prevents the session from restarting if the view is recreated after authentication
        if viewModel.isAuthenticated {
            print("🔒 DEBUG: WebAuthView created but user is already authenticated, not starting session")
            DispatchQueue.main.async {
                onDismiss()
            }
            return viewController
        }

        // Guard: Don't start a new session if one already exists
        if let existingSession = context.coordinator.session {
            print("🔒 DEBUG: WebAuthView session already exists, cancelling old one and not creating new")
            existingSession.cancel()
            context.coordinator.session = nil
            DispatchQueue.main.async {
                onDismiss()
            }
            return viewController
        }

        // Use ASWebAuthenticationSession with iOS URL scheme for proper OAuth flow
        // This uses the iOS client ID and custom URL scheme: com.googleusercontent.apps.53644033433-qpic9cnjknphdpa332d7flq7nvvdv520
        let callbackURLScheme = "com.googleusercontent.apps.53644033433-qpic9cnjknphdpa332d7flq7nvvdv520"

        // Capture coordinator reference for use in completion handler
        let coordinator = context.coordinator

        let session = ASWebAuthenticationSession(
            url: url,
            callbackURLScheme: callbackURLScheme,
            completionHandler: { callbackURL, error in
                DispatchQueue.main.async {
                    if let error = error {
                        print("🔍 DEBUG: ASWebAuthenticationSession error received, cancelling session")
                        // Clear session reference on error
                        coordinator.session = nil

                        if let authError = error as? ASWebAuthenticationSessionError,
                           authError.code == .canceledLogin {
                            print("ℹ️ User cancelled OAuth flow")
                            onDismiss()
                            return
                        }
                        print("❌ OAuth error: \(error.localizedDescription)")
                        onDismiss()
                        return
                    }

                    if let callbackURL = callbackURL {
                        print("✅ Received callback URL: \(callbackURL.absoluteString)")
                        print("🔍 DEBUG: Cancelling ASWebAuthenticationSession after receiving callback")
                        // Cancel and clear the session immediately after receiving callback to prevent re-triggering
                        let sessionToCancel = coordinator.session
                        coordinator.session = nil
                        sessionToCancel?.cancel()

                        // Process callback immediately - the session is already cancelled
                        if let components = URLComponents(url: callbackURL, resolvingAgainstBaseURL: false) {
                            onCallback(components)
                        } else {
                            print("❌ Failed to parse callback URL")
                            onDismiss()
                        }
                    } else {
                        print("⚠️ No callback URL received")
                        print("🔍 DEBUG: Cancelling ASWebAuthenticationSession - no callback URL")
                        // Cancel the session if no callback URL
                        let sessionToCancel = coordinator.session
                        coordinator.session = nil
                        sessionToCancel?.cancel()
                        onDismiss()
                    }
                }
            }
        )

        // Use SFSafariViewController presentation for better UX and credential access
        session.presentationContextProvider = coordinator
        session.prefersEphemeralWebBrowserSession = false // Use saved credentials

        coordinator.session = session

        // Start the session
        DispatchQueue.main.async {
            print("🔍 DEBUG: About to start ASWebAuthenticationSession - isAuthenticated=\(viewModel.isAuthenticated), session in coordinator=\(coordinator.session != nil)")
            // Double-check authentication state before starting
            if viewModel.isAuthenticated {
                print("🔒 DEBUG: Authentication state changed, cancelling session before start")
                coordinator.session = nil
                session.cancel()
                onDismiss()
                return
            }

            if !session.start() {
                print("❌ Failed to start ASWebAuthenticationSession")
                coordinator.session = nil
                onDismiss()
            } else {
                print("🔍 DEBUG: ASWebAuthenticationSession started successfully at \(Date())")
                print("🔍 DEBUG: Stack trace: \(Thread.callStackSymbols.prefix(5).joined(separator: "\n"))")
            }
        }

        return viewController
    }

    func updateUIViewController(_ uiViewController: UIViewController, context: Context) {
        print("🔍 DEBUG: updateUIViewController called - isAuthenticated=\(viewModel.isAuthenticated), session exists=\(context.coordinator.session != nil)")
        // If authentication state changed, cancel the session immediately
        if viewModel.isAuthenticated {
            if let session = context.coordinator.session {
                print("🔒 DEBUG: Authentication completed, cancelling session in updateUIViewController")
                session.cancel()
                context.coordinator.session = nil
                DispatchQueue.main.async {
                    onDismiss()
                }
            } else {
                print("🔍 DEBUG: updateUIViewController - authenticated but no session to cancel")
            }
        }
    }


    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    class Coordinator: NSObject, ASWebAuthenticationPresentationContextProviding {
        var session: ASWebAuthenticationSession?

        func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
            return UIApplication.shared.connectedScenes
                .compactMap { $0 as? UIWindowScene }
                .flatMap { $0.windows }
                .first { $0.isKeyWindow } ?? UIWindow()
        }
    }
}
