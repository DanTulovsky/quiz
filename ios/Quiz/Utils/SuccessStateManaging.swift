import Foundation
import Combine

protocol SuccessStateManaging: BaseViewModel {
    var isSuccess: Bool { get set }
}

extension SuccessStateManaging {
    func resetSuccessState() {
        isSuccess = false
    }

    func setSuccessState() {
        isSuccess = true
    }

    func executeWithSuccessState<T>(
        publisher: AnyPublisher<T, APIService.APIError>,
        onSuccess: @escaping (T) -> Void
    ) -> AnyCancellable {
        resetSuccessState()
        return publisher
            .handleErrorOnly(on: self)
            .sinkValue(on: self) { [weak self] value in
                onSuccess(value)
                self?.setSuccessState()
            }
    }

    func executeVoidWithSuccessState(
        publisher: AnyPublisher<SuccessResponse, APIService.APIError>
    ) -> AnyCancellable {
        resetSuccessState()
        return publisher
            .handleErrorOnly(on: self)
            .sinkVoid(on: self) { [weak self] in
                self?.setSuccessState()
            }
    }
}

