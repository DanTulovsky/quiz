import SwiftUI

struct SignupView: View {
    @EnvironmentObject var viewModel: AuthenticationViewModel

    var body: some View {
        VStack {
            VStack(alignment: .leading, spacing: AppTheme.Spacing.itemSpacing) {
                VStack(alignment: .leading, spacing: 8) {
                    Text("Username")
                        .font(AppTheme.Typography.subheadlineFont.weight(.medium))
                    TextField("Username", text: $viewModel.username)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled(true)
                        .padding()
                        .background(AppTheme.Colors.secondaryBackground)
                        .cornerRadius(AppTheme.CornerRadius.button)
                }
                
                VStack(alignment: .leading, spacing: 8) {
                    Text("Email")
                        .font(AppTheme.Typography.subheadlineFont.weight(.medium))
                    TextField("Email", text: $viewModel.email)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled(true)
                        .padding()
                        .background(AppTheme.Colors.secondaryBackground)
                        .cornerRadius(AppTheme.CornerRadius.button)
                }
                
                VStack(alignment: .leading, spacing: 8) {
                    Text("Password")
                        .font(AppTheme.Typography.subheadlineFont.weight(.medium))
                    SecureField("Password", text: $viewModel.password)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled(true)
                        .padding()
                        .background(AppTheme.Colors.secondaryBackground)
                        .cornerRadius(AppTheme.CornerRadius.button)
                }
                
                if viewModel.error != nil {
                    Text("Signup failed. Please try again.")
                        .foregroundColor(AppTheme.Colors.errorRed)
                        .font(AppTheme.Typography.captionFont)
                        .padding()
                        .background(AppTheme.Colors.errorRed.opacity(0.1))
                        .cornerRadius(AppTheme.CornerRadius.button)
                }
                
                Button("Sign Up") {
                    viewModel.signup()
                }
                .buttonStyle(PrimaryButtonStyle(isDisabled: viewModel.username.isEmpty || viewModel.password.isEmpty))
            }
        }
        .padding()
    }
}
