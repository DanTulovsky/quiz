import SwiftUI

struct ErrorDisplay: View {
    let error: APIService.APIError?
    let onDismiss: (() -> Void)?
    let showDetailsButton: Bool
    let onShowDetails: (() -> Void)?
    let showRetryButton: Bool
    let onRetry: (() -> Void)?

    init(
        error: APIService.APIError?,
        onDismiss: (() -> Void)? = nil,
        showDetailsButton: Bool = false,
        onShowDetails: (() -> Void)? = nil,
        showRetryButton: Bool = false,
        onRetry: (() -> Void)? = nil
    ) {
        self.error = error
        self.onDismiss = onDismiss
        self.showDetailsButton = showDetailsButton
        self.onShowDetails = onShowDetails
        self.showRetryButton = showRetryButton
        self.onRetry = onRetry
    }

    var body: some View {
        if let error = error {
            VStack(alignment: .leading, spacing: 12) {
                HStack {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundColor(AppTheme.Colors.errorRed)
                    Text("Error")
                        .font(AppTheme.Typography.subheadlineFont.weight(.semibold))
                        .foregroundColor(AppTheme.Colors.errorRed)
                    Spacer()
                    if let code = error.errorCode {
                        Text(code)
                            .font(AppTheme.Typography.captionFont.weight(.bold))
                            .foregroundColor(AppTheme.Colors.errorRed)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(AppTheme.Colors.errorRed.opacity(0.15))
                            .cornerRadius(6)
                    }
                    if let onDismiss = onDismiss {
                        Button(action: onDismiss) {
                            Image(systemName: "xmark.circle.fill")
                                .foregroundColor(AppTheme.Colors.secondaryText)
                        }
                    }
                }
                Text(error.localizedDescription)
                    .font(AppTheme.Typography.subheadlineFont)
                    .foregroundColor(AppTheme.Colors.secondaryText)
                    .frame(maxWidth: .infinity, alignment: .leading)

                HStack(spacing: 12) {
                    if showDetailsButton, error.errorDetails != nil, let onShowDetails = onShowDetails {
                        Button(action: onShowDetails) {
                            HStack(spacing: 4) {
                                Text("View Details")
                                    .font(AppTheme.Typography.captionFont)
                                Image(systemName: "chevron.right")
                                    .font(.system(size: 10))
                            }
                            .foregroundColor(AppTheme.Colors.primaryBlue)
                        }
                    }
                    if showRetryButton, let onRetry = onRetry {
                        Button(action: onRetry) {
                            HStack(spacing: 4) {
                                Image(systemName: "arrow.clockwise")
                                    .font(.system(size: 12))
                                Text("Retry")
                                    .font(AppTheme.Typography.captionFont)
                            }
                            .foregroundColor(AppTheme.Colors.primaryBlue)
                        }
                    }
                }
                .padding(.top, 4)
            }
            .padding(AppTheme.Spacing.innerPadding)
            .background(AppTheme.Colors.errorRed.opacity(0.1))
            .cornerRadius(AppTheme.CornerRadius.button)
        }
    }
}
