import SwiftUI
import AuthenticationServices
import SafariServices

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
                        Text("AI Language Quiz")
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
                                .background(Color(.secondarySystemBackground))
                                .cornerRadius(8)
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
                            .background(Color(.secondarySystemBackground))
                            .cornerRadius(8)
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
                            .background(Color(.systemGray4))
                            .foregroundColor(.secondary)
                            .cornerRadius(12)
                        }
                        .disabled(isLoading || viewModel.username.isEmpty || viewModel.password.isEmpty)

                        if viewModel.error != nil {
                            Text("Login failed. Please check your credentials.")
                                .foregroundColor(.red)
                                .font(.caption)
                                .padding()
                                .background(Color.red.opacity(0.1))
                                .cornerRadius(8)
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
                            viewModel.initiateGoogleLogin()
                            showWebAuth = true
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
                            .background(Color(.systemBackground))
                            .overlay(
                                RoundedRectangle(cornerRadius: 12)
                                    .stroke(Color(.systemGray4), lineWidth: 1)
                            )
                            .cornerRadius(12)
                        }
                    }
                    .padding(.horizontal, 24)

                    Spacer()
                }
            }
            .navigationBarHidden(true)
            .sheet(isPresented: $showWebAuth) {
                if let url = viewModel.googleAuthURL {
                    SafariWebView(url: url, onCallback: { components in
                        if let code = components.queryItems?.first(where: { $0.name == "code" })?.value {
                            let state = components.queryItems?.first(where: { $0.name == "state" })?.value
                            viewModel.handleGoogleCallback(code: code, state: state)
                            showWebAuth = false
                        }
                    })
                }
            }
        }
    }
}

struct SafariWebView: UIViewControllerRepresentable {
    let url: URL
    let onCallback: (URLComponents) -> Void

    func makeUIViewController(context: Context) -> SFSafariViewController {
        let safari = SFSafariViewController(url: url)
        safari.delegate = context.coordinator
        return safari
    }

    func updateUIViewController(_ uiViewController: SFSafariViewController, context: Context) {}

    func makeCoordinator() -> Coordinator {
        Coordinator(onCallback: onCallback)
    }

    class Coordinator: NSObject, SFSafariViewControllerDelegate {
        let onCallback: (URLComponents) -> Void

        init(onCallback: @escaping (URLComponents) -> Void) {
            self.onCallback = onCallback
        }

        func safariViewController(_ controller: SFSafariViewController, initialLoadDidRedirectTo URL: URL) {
            if let components = URLComponents(url: URL, resolvingAgainstBaseURL: false),
               components.path.contains("callback") {
                onCallback(components)
            }
        }
    }
}
