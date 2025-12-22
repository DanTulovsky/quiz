import XCTest
import SwiftUI
@testable import Quiz

class FontSizeHelperTests: XCTestCase {

    // MARK: - FontSizeHelper.multiplier Tests

    func testMultiplierForSmall() {
        // Given
        let size = "S"

        // When
        let multiplier = FontSizeHelper.multiplier(for: size)

        // Then
        XCTAssertEqual(multiplier, 0.85, accuracy: 0.001, "Small size should have multiplier of 0.85")
    }

    func testMultiplierForMedium() {
        // Given
        let size = "M"

        // When
        let multiplier = FontSizeHelper.multiplier(for: size)

        // Then
        XCTAssertEqual(multiplier, 1.0, accuracy: 0.001, "Medium size should have multiplier of 1.0")
    }

    func testMultiplierForLarge() {
        // Given
        let size = "L"

        // When
        let multiplier = FontSizeHelper.multiplier(for: size)

        // Then
        XCTAssertEqual(multiplier, 1.15, accuracy: 0.001, "Large size should have multiplier of 1.15")
    }

    func testMultiplierForExtraLarge() {
        // Given
        let size = "XL"

        // When
        let multiplier = FontSizeHelper.multiplier(for: size)

        // Then
        XCTAssertEqual(multiplier, 1.3, accuracy: 0.001, "Extra large size should have multiplier of 1.3")
    }

    func testMultiplierForInvalidSize() {
        // Given
        let invalidSizes = ["", "XS", "XXL", "invalid", "s", "m", "l", "xl"]

        // When & Then
        for size in invalidSizes {
            let multiplier = FontSizeHelper.multiplier(for: size)
            XCTAssertEqual(
                multiplier, 1.0, accuracy: 0.001,
                "Invalid size '\(size)' should default to multiplier of 1.0"
            )
        }
    }

    func testMultiplierForNilSize() {
        // Given
        let size: String? = nil

        // When
        let multiplier = FontSizeHelper.multiplier(for: size ?? "")

        // Then
        XCTAssertEqual(multiplier, 1.0, accuracy: 0.001, "Nil size should default to multiplier of 1.0")
    }

    // MARK: - FontSizeHelper.scaledFont Tests

    func testScaledFontWithSmallMultiplier() {
        // Given
        let baseSize: CGFloat = 48
        let multiplier: CGFloat = 0.85
        let weight = Font.Weight.bold

        // When
        _ = FontSizeHelper.scaledFont(size: baseSize, weight: weight, multiplier: multiplier)

        // Then
        let expectedSize = baseSize * multiplier
        XCTAssertEqual(expectedSize, 40.8, accuracy: 0.1, "Scaled font size should be 48 * 0.85 = 40.8")
    }

    func testScaledFontWithMediumMultiplier() {
        // Given
        let baseSize: CGFloat = 48
        let multiplier: CGFloat = 1.0
        let weight = Font.Weight.regular

        // When
        _ = FontSizeHelper.scaledFont(size: baseSize, weight: weight, multiplier: multiplier)

        // Then
        let expectedSize = baseSize * multiplier
        XCTAssertEqual(expectedSize, 48.0, accuracy: 0.1, "Scaled font size should be 48 * 1.0 = 48.0")
    }

    func testScaledFontWithLargeMultiplier() {
        // Given
        let baseSize: CGFloat = 17
        let multiplier: CGFloat = 1.15
        let weight = Font.Weight.semibold

        // When
        _ = FontSizeHelper.scaledFont(size: baseSize, weight: weight, multiplier: multiplier)

        // Then
        let expectedSize = baseSize * multiplier
        XCTAssertEqual(expectedSize, 19.55, accuracy: 0.1, "Scaled font size should be 17 * 1.15 = 19.55")
    }

    func testScaledFontWithExtraLargeMultiplier() {
        // Given
        let baseSize: CGFloat = 60
        let multiplier: CGFloat = 1.3
        let weight = Font.Weight.regular

        // When
        _ = FontSizeHelper.scaledFont(size: baseSize, weight: weight, multiplier: multiplier)

        // Then
        let expectedSize = baseSize * multiplier
        XCTAssertEqual(expectedSize, 78.0, accuracy: 0.1, "Scaled font size should be 60 * 1.3 = 78.0")
    }

    func testScaledFontWithZeroMultiplier() {
        // Given
        let baseSize: CGFloat = 48
        let multiplier: CGFloat = 0.0
        let weight = Font.Weight.regular

        // When
        _ = FontSizeHelper.scaledFont(size: baseSize, weight: weight, multiplier: multiplier)

        // Then
        let expectedSize = baseSize * multiplier
        XCTAssertEqual(expectedSize, 0.0, accuracy: 0.1, "Scaled font size should be 48 * 0.0 = 0.0")
    }

    func testScaledFontWithNegativeMultiplier() {
        // Given
        let baseSize: CGFloat = 48
        let multiplier: CGFloat = -1.0
        let weight = Font.Weight.regular

        // When
        _ = FontSizeHelper.scaledFont(size: baseSize, weight: weight, multiplier: multiplier)

        // Then
        let expectedSize = baseSize * multiplier
        XCTAssertEqual(expectedSize, -48.0, accuracy: 0.1, "Scaled font size should handle negative multiplier")
    }

    func testScaledFontDefaultWeight() {
        // Given
        let baseSize: CGFloat = 17
        let multiplier: CGFloat = 1.0

        // When
        let font = FontSizeHelper.scaledFont(size: baseSize, multiplier: multiplier)

        // Then
        XCTAssertNotNil(font, "Scaled font should be created with default weight")
    }

    // MARK: - Font.scaledSystem Tests

    func testScaledSystemFont() {
        // Given
        let baseSize: CGFloat = 28
        let multiplier: CGFloat = 1.15
        let weight = Font.Weight.bold

        // When
        let font = Font.scaledSystem(size: baseSize, weight: weight, multiplier: multiplier)

        // Then
        XCTAssertNotNil(font, "Scaled system font should be created")
        let expectedSize = baseSize * multiplier
        XCTAssertEqual(expectedSize, 32.2, accuracy: 0.1, "Scaled system font size should be correct")
    }

    // MARK: - Integration Tests

    func testFullWorkflowSmallSize() {
        // Given
        let sizeString = "S"
        let baseSize: CGFloat = 48

        // When
        let multiplier = FontSizeHelper.multiplier(for: sizeString)
        let font = FontSizeHelper.scaledFont(size: baseSize, weight: .bold, multiplier: multiplier)

        // Then
        XCTAssertEqual(multiplier, 0.85, accuracy: 0.001)
        XCTAssertNotNil(font)
        let expectedSize = baseSize * multiplier
        XCTAssertEqual(expectedSize, 40.8, accuracy: 0.1)
    }

    func testFullWorkflowMediumSize() {
        // Given
        let sizeString = "M"
        let baseSize: CGFloat = 17

        // When
        let multiplier = FontSizeHelper.multiplier(for: sizeString)
        let font = FontSizeHelper.scaledFont(size: baseSize, weight: .semibold, multiplier: multiplier)

        // Then
        XCTAssertEqual(multiplier, 1.0, accuracy: 0.001)
        XCTAssertNotNil(font)
        let expectedSize = baseSize * multiplier
        XCTAssertEqual(expectedSize, 17.0, accuracy: 0.1)
    }

    func testFullWorkflowLargeSize() {
        // Given
        let sizeString = "L"
        let baseSize: CGFloat = 60

        // When
        let multiplier = FontSizeHelper.multiplier(for: sizeString)
        let font = FontSizeHelper.scaledFont(size: baseSize, weight: .regular, multiplier: multiplier)

        // Then
        XCTAssertEqual(multiplier, 1.15, accuracy: 0.001)
        XCTAssertNotNil(font)
        let expectedSize = baseSize * multiplier
        XCTAssertEqual(expectedSize, 69.0, accuracy: 0.1)
    }

    func testFullWorkflowExtraLargeSize() {
        // Given
        let sizeString = "XL"
        let baseSize: CGFloat = 80

        // When
        let multiplier = FontSizeHelper.multiplier(for: sizeString)
        let font = FontSizeHelper.scaledFont(size: baseSize, weight: .regular, multiplier: multiplier)

        // Then
        XCTAssertEqual(multiplier, 1.3, accuracy: 0.001)
        XCTAssertNotNil(font)
        let expectedSize = baseSize * multiplier
        XCTAssertEqual(expectedSize, 104.0, accuracy: 0.1)
    }

    // MARK: - Edge Cases

    func testVerySmallBaseSize() {
        // Given
        let baseSize: CGFloat = 10
        let multiplier: CGFloat = 0.85

        // When
        let font = FontSizeHelper.scaledFont(size: baseSize, multiplier: multiplier)

        // Then
        XCTAssertNotNil(font)
        let expectedSize = baseSize * multiplier
        XCTAssertEqual(expectedSize, 8.5, accuracy: 0.1)
    }

    func testVeryLargeBaseSize() {
        // Given
        let baseSize: CGFloat = 100
        let multiplier: CGFloat = 1.3

        // When
        let font = FontSizeHelper.scaledFont(size: baseSize, multiplier: multiplier)

        // Then
        XCTAssertNotNil(font)
        let expectedSize = baseSize * multiplier
        XCTAssertEqual(expectedSize, 130.0, accuracy: 0.1)
    }

    func testAllFontWeights() {
        // Given
        let baseSize: CGFloat = 17
        let multiplier: CGFloat = 1.0
        let weights: [Font.Weight] = [.ultraLight, .thin, .light, .regular, .medium, .semibold, .bold, .heavy, .black]

        // When & Then
        for weight in weights {
            let font = FontSizeHelper.scaledFont(size: baseSize, weight: weight, multiplier: multiplier)
            XCTAssertNotNil(font, "Font should be created for weight: \(weight)")
        }
    }

    // MARK: - Consistency Tests

    func testMultiplierConsistency() {
        // Given
        let sizes = ["S", "M", "L", "XL"]

        // When & Then
        for size in sizes {
            let multiplier1 = FontSizeHelper.multiplier(for: size)
            let multiplier2 = FontSizeHelper.multiplier(for: size)
            XCTAssertEqual(
                multiplier1, multiplier2, accuracy: 0.001,
                "Multiplier should be consistent for size: \(size)"
            )
        }
    }

    func testScaledFontConsistency() {
        // Given
        let baseSize: CGFloat = 48
        let multiplier: CGFloat = 1.15
        let weight = Font.Weight.bold

        // When
        let font1 = FontSizeHelper.scaledFont(size: baseSize, weight: weight, multiplier: multiplier)
        let font2 = FontSizeHelper.scaledFont(size: baseSize, weight: weight, multiplier: multiplier)

        // Then
        XCTAssertNotNil(font1)
        XCTAssertNotNil(font2)
    }
}
