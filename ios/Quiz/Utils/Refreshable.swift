import Foundation
import Combine

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

