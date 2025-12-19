import XCTest

@testable import Quiz

class DateFormattersTests: XCTestCase {

    func testISO8601Formatter() {
        // Given
        let date = Date(timeIntervalSince1970: 1_609_459_200)  // 2021-01-01 00:00:00 UTC
        let expected = "2021-01-01"

        // When
        let formatted = DateFormatters.iso8601.string(from: date)

        // Then
        XCTAssertEqual(formatted, expected, "ISO8601 formatter should format date correctly")
    }

    func testISO8601FormatterConsistency() {
        // Given
        let date = Date()

        // When
        let formatted1 = DateFormatters.iso8601.string(from: date)
        let formatted2 = DateFormatters.iso8601.string(from: date)

        // Then
        XCTAssertEqual(formatted1, formatted2, "Formatter should be consistent across calls")
    }

    func testISO8601FormatterParsing() {
        // Given
        let dateString = "2021-12-25"

        // When
        let parsed = DateFormatters.iso8601.date(from: dateString)

        // Then
        XCTAssertNotNil(parsed, "Should be able to parse ISO8601 date string")
        if let parsed = parsed {
            let reformatted = DateFormatters.iso8601.string(from: parsed)
            XCTAssertEqual(reformatted, dateString, "Parsed date should match original string")
        }
    }

    func testDisplayFullFormatter() {
        // Given
        let date = Date(timeIntervalSince1970: 1_609_459_200)

        // When
        let formatted = DateFormatters.displayFull.string(from: date)

        // Then
        XCTAssertFalse(formatted.isEmpty, "Full date formatter should produce non-empty string")
        // Note: The exact format depends on locale, so we just verify it's not empty
        // The formatter uses .full dateStyle which should always produce a meaningful string
    }

    func testDisplayMediumFormatter() {
        // Given
        let date = Date(timeIntervalSince1970: 1_609_459_200)

        // When
        let formatted = DateFormatters.displayMedium.string(from: date)

        // Then
        XCTAssertFalse(formatted.isEmpty, "Medium date formatter should produce non-empty string")
    }

    func testFormatterReusability() {
        // Given
        let dates = [
            Date(timeIntervalSince1970: 1_609_459_200),
            Date(timeIntervalSince1970: 1_640_995_200),
            Date(timeIntervalSince1970: 1_672_531_200),
        ]

        // When
        let formatted = dates.map { DateFormatters.iso8601.string(from: $0) }

        // Then
        XCTAssertEqual(formatted.count, 3, "Should format all dates")
        XCTAssertEqual(formatted[0], "2021-01-01")
        XCTAssertEqual(formatted[1], "2022-01-01")
        XCTAssertEqual(formatted[2], "2023-01-01")
    }
}
