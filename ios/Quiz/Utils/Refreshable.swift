import Combine
import Foundation

protocol Refreshable: BaseViewModel {
    func refresh()
}

extension Refreshable {
    func refresh() {
        clearError()
        refreshData()
    }

    func refreshData() {
        // Subclasses implement this
    }
}
