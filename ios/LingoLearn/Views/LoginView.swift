import SwiftUI

struct LoginView: View {
    @StateObject private var viewModel = AuthenticationViewModel()

    var body: some View {
        NavigationView {
            VStack {
                TextField("Username", text: $viewModel.username)
                    .textFieldStyle(RoundedBorderTextFieldStyle())
                    .padding()
                SecureField("Password", text: $viewModel.password)
                    .textFieldStyle(RoundedBorderTextFieldStyle())
                    .padding()
                if viewModel.error != nil {
                    Text("Login failed. Please check your credentials.")
                        .foregroundColor(.red)
                }
                Button("Login") {
                    viewModel.login()
                }
                .padding()
                .background(Color.blue)
                .foregroundColor(.white)
                .cornerRadius(8)
                
                NavigationLink("Don't have an account? Sign up", destination: SignupView())
                    .padding()
            }
            .padding()
            .navigationTitle("Login")
        }
    }
}
