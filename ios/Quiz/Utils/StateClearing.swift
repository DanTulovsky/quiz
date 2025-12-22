import Combine
import Foundation

protocol StateClearing: BaseViewModel {
    func clearStateBeforeFetch()
}

extension StateClearing {
    func clearStateBeforeFetch() {
        clearError()
    }
}
