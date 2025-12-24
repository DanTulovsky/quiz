import Foundation

struct DateFormatters {
    static let iso8601: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.locale = Locale(identifier: "en_US_POSIX")
        // Use local timezone instead of UTC to match web behavior
        // This ensures the date matches the user's local calendar day
        formatter.timeZone = TimeZone.current
        return formatter
    }()

    static let iso8601Full: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()

    static let displayFull: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateStyle = .full
        return formatter
    }()

    static let displayMedium: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateStyle = .medium
        return formatter
    }()

    static let displayShort: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateStyle = .short
        return formatter
    }()
}

extension Date {
    var iso8601String: String {
        return DateFormatters.iso8601.string(from: self)
    }

    var displayString: String {
        return DateFormatters.displayMedium.string(from: self)
    }

    static func from(iso8601String: String) -> Date? {
        return DateFormatters.iso8601.date(from: iso8601String)
    }
}
