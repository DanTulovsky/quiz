import SwiftUI

struct AppTheme {
    struct Colors {
        static let primaryBlue = Color.blue
        static let accentIndigo = Color.indigo
        static let successGreen = Color.green
        static let errorRed = Color.red

        static let cardBackground = Color(.systemBackground)
        static let secondaryBackground = Color(.secondarySystemBackground)

        static let primaryText = Color.primary
        static let secondaryText = Color.secondary

        static let borderGray = Color.gray.opacity(0.1)
        static let borderBlue = Color.blue.opacity(0.2)
    }

    struct Spacing {
        static let cardPadding: CGFloat = 20
        static let innerPadding: CGFloat = 12
        static let sectionSpacing: CGFloat = 20
        static let itemSpacing: CGFloat = 12
        static let buttonVerticalPadding: CGFloat = 12
    }

    struct CornerRadius {
        static let card: CGFloat = 16
        static let button: CGFloat = 12
        static let badge: CGFloat = 8
        static let innerCard: CGFloat = 12
    }

    struct Shadow {
        static let card = (color: Color.black.opacity(0.05), radius: CGFloat(8), x: CGFloat(0), y: CGFloat(4))
    }

    struct Typography {
        static let badgeFont = Font.caption2.bold()
        static let headingFont = Font.title3.weight(.medium)
        static let bodyFont = Font.body
        static let subheadlineFont = Font.subheadline
        static let captionFont = Font.caption
        static let buttonFont = Font.headline
    }
}

struct CardModifier: ViewModifier {
    func body(content: Content) -> some View {
        content
            .padding(AppTheme.Spacing.cardPadding)
            .background(AppTheme.Colors.cardBackground)
            .cornerRadius(AppTheme.CornerRadius.card)
            .shadow(
                color: AppTheme.Shadow.card.color,
                radius: AppTheme.Shadow.card.radius,
                x: AppTheme.Shadow.card.x,
                y: AppTheme.Shadow.card.y
            )
            .overlay(
                RoundedRectangle(cornerRadius: AppTheme.CornerRadius.card)
                    .stroke(AppTheme.Colors.borderGray, lineWidth: 1)
            )
    }
}

struct InnerCardModifier: ViewModifier {
    func body(content: Content) -> some View {
        content
            .padding(AppTheme.Spacing.innerPadding)
            .background(AppTheme.Colors.cardBackground)
            .cornerRadius(AppTheme.CornerRadius.innerCard)
            .overlay(
                RoundedRectangle(cornerRadius: AppTheme.CornerRadius.innerCard)
                    .stroke(AppTheme.Colors.borderGray, lineWidth: 1)
            )
    }
}

struct PrimaryButtonStyle: ButtonStyle {
    var isDisabled: Bool = false

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(AppTheme.Typography.buttonFont)
            .foregroundColor(.white)
            .frame(maxWidth: .infinity)
            .padding(.vertical, AppTheme.Spacing.buttonVerticalPadding)
            .background(isDisabled ? Color.gray : AppTheme.Colors.primaryBlue)
            .cornerRadius(AppTheme.CornerRadius.button)
            .opacity(configuration.isPressed ? 0.8 : 1.0)
    }
}

struct SecondaryButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(AppTheme.Typography.subheadlineFont.weight(.medium))
            .foregroundColor(AppTheme.Colors.primaryBlue)
            .frame(maxWidth: .infinity)
            .padding(.vertical, AppTheme.Spacing.buttonVerticalPadding)
            .background(AppTheme.Colors.primaryBlue.opacity(0.1))
            .cornerRadius(AppTheme.CornerRadius.button)
            .overlay(
                RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button)
                    .stroke(AppTheme.Colors.borderBlue, lineWidth: 1)
            )
            .opacity(configuration.isPressed ? 0.8 : 1.0)
    }
}

struct OptionButtonStyle: ButtonStyle {
    var isSelected: Bool
    var isCorrect: Bool = false
    var isIncorrect: Bool = false

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(AppTheme.Typography.bodyFont)
            .foregroundColor(
                isIncorrect ? AppTheme.Colors.errorRed :
                (isCorrect ? AppTheme.Colors.successGreen :
                (isSelected ? .white : AppTheme.Colors.primaryText))
            )
            .frame(maxWidth: .infinity)
            .padding(AppTheme.Spacing.innerPadding)
            .background(
                isIncorrect ? AppTheme.Colors.errorRed.opacity(0.1) :
                (isCorrect ? AppTheme.Colors.successGreen.opacity(0.1) :
                (isSelected ? AppTheme.Colors.primaryBlue : AppTheme.Colors.primaryBlue.opacity(0.05)))
            )
            .cornerRadius(AppTheme.CornerRadius.button)
            .overlay(
                RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button)
                    .stroke(
                        isIncorrect ? AppTheme.Colors.errorRed :
                        (isCorrect ? AppTheme.Colors.successGreen :
                        AppTheme.Colors.borderBlue),
                        lineWidth: 1
                    )
            )
    }
}

extension View {
    func appCard() -> some View {
        modifier(CardModifier())
    }

    func appInnerCard() -> some View {
        modifier(InnerCardModifier())
    }
}

