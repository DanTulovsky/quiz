import Foundation
import Combine

protocol SubmittingState: BaseViewModel {
    var isSubmitting: Bool { get set }
}

extension SubmittingState {
    func executeWithSubmittingState<T>(
        publisher: AnyPublisher<T, APIService.APIError>,
        onSuccess: @escaping (T) -> Void
    ) -> AnyCancellable {
        isSubmitting = true
        return publisher
            .handleErrorOnly(on: self)
            .sink(
                receiveCompletion: { [weak self] _ in
                    self?.isSubmitting = false
                },
                receiveValue: { value in
                    onSuccess(value)
                }
            )
    }

    func executeVoidWithSubmittingState(
        publisher: AnyPublisher<SuccessResponse, APIService.APIError>,
        onSuccess: @escaping () -> Void
    ) -> AnyCancellable {
        isSubmitting = true
        return publisher
            .handleErrorOnly(on: self)
            .sink(
                receiveCompletion: { [weak self] _ in
                    self?.isSubmitting = false
                },
                receiveValue: { _ in
                    onSuccess()
                }
            )
    }
}

