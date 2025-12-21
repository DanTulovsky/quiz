import SwiftUI

struct FontSizeMultiplierKey: EnvironmentKey {
    static let defaultValue: CGFloat = 1.0
}

extension EnvironmentValues {
    var fontSizeMultiplier: CGFloat {
        get { self[FontSizeMultiplierKey.self] }
        set { self[FontSizeMultiplierKey.self] = newValue }
    }
}

struct FontSizeHelper {
    static func multiplier(for size: String) -> CGFloat {
        switch size {
        case "S": return 0.85
        case "M": return 1.0
        case "L": return 1.15
        case "XL": return 1.3
        default: return 1.0
        }
    }

    static func scaledFont(size: CGFloat, weight: Font.Weight = .regular, multiplier: CGFloat) -> Font {
        return .system(size: size * multiplier, weight: weight)
    }
}

extension Font {
    static func scaledSystem(size: CGFloat, weight: Font.Weight = .regular, multiplier: CGFloat) -> Font {
        return .system(size: size * multiplier, weight: weight)
    }
}

extension View {
    func scaledFont(size: CGFloat, weight: Font.Weight = .regular) -> some View {
        self.modifier(ScaledFontModifier(baseSize: size, weight: weight))
    }
}

struct ScaledFontModifier: ViewModifier {
    let baseSize: CGFloat
    let weight: Font.Weight
    @Environment(\.fontSizeMultiplier) var multiplier

    func body(content: Content) -> some View {
        content.font(.system(size: baseSize * multiplier, weight: weight))
    }
}

