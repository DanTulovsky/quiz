import Foundation

protocol Navigable: BaseViewModel {
    associatedtype Item
    var items: [Item] { get }
    var currentIndex: Int { get set }
}

extension Navigable {
    var hasNext: Bool {
        currentIndex < items.count - 1
    }

    var hasPrevious: Bool {
        currentIndex > 0
    }

    var currentItem: Item? {
        guard currentIndex >= 0 && currentIndex < items.count else { return nil }
        return items[currentIndex]
    }

    func goToNext() -> Bool {
        guard hasNext else { return false }
        currentIndex += 1
        return true
    }

    func goToPrevious() -> Bool {
        guard hasPrevious else { return false }
        currentIndex -= 1
        return true
    }

    func goToBeginning() -> Bool {
        guard !items.isEmpty && currentIndex != 0 else { return false }
        currentIndex = 0
        return true
    }

    func goToEnd() -> Bool {
        guard !items.isEmpty else { return false }
        let lastIndex = items.count - 1
        guard currentIndex != lastIndex else { return false }
        currentIndex = lastIndex
        return true
    }
}
