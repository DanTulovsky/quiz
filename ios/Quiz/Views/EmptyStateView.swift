import SwiftUI

struct EmptyStateView: View {
    let icon: String
    let title: String
    let message: String
    let actionTitle: String?
    let action: (() -> Void)?

    init(
        icon: String = "tray",
        title: String,
        message: String,
        actionTitle: String? = nil,
        action: (() -> Void)? = nil
    ) {
        self.icon = icon
        self.title = title
        self.message = message
        self.actionTitle = actionTitle
        self.action = action
    }

    var body: some View {
        VStack(spacing: 20) {
            Image(systemName: icon)
                .scaledFont(size: 60)
                .foregroundColor(AppTheme.Colors.secondaryText)

            Text(title)
                .font(AppTheme.Typography.headingFont)
                .foregroundColor(AppTheme.Colors.primaryText)

            Text(message)
                .font(AppTheme.Typography.bodyFont)
                .foregroundColor(AppTheme.Colors.secondaryText)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 40)

            if let actionTitle = actionTitle, let action = action {
                Button(action: action) {
                    Text(actionTitle)
                }
                .buttonStyle(PrimaryButtonStyle())
                .padding(.top, 10)
            }
        }
        .padding(.vertical, 60)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}




