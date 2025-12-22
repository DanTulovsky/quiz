import Foundation

struct QueryParameters {
    private var items: [URLQueryItem] = []

    init() {}

    mutating func add(_ name: String, value: String?) {
        if let value = value, !value.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            items.append(URLQueryItem(name: name, value: value))
        }
    }

    mutating func add(_ name: String, value: Int?) {
        if let value = value {
            items.append(URLQueryItem(name: name, value: String(value)))
        }
    }

    mutating func add(_ name: String, value: Bool?) {
        if let value = value {
            items.append(URLQueryItem(name: name, value: String(value)))
        }
    }

    func build() -> [URLQueryItem]? {
        return items.isEmpty ? nil : items
    }
}
