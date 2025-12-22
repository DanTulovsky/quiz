import Combine
import Foundation

protocol SubmittingState: BaseViewModel {
    var isSubmitting: Bool { get set }
}

extension SubmittingState {
    func executeWithSubmittingState<T>(
        publisher: AnyPublisher<T, APIService.APIError>,
        onSuccess: @escaping (T) -> Void
    ) -> AnyCancellable {
        isSubmitting = true
        return
            publisher
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
        return
            publisher
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

protocol AnswerSubmittable: BaseViewModel {
    associatedtype AnswerResponseType
    var answerResponse: AnswerResponseType? { get set }
    var selectedAnswerIndex: Int? { get set }
}

extension AnswerSubmittable {
    func submitAnswer<T>(
        publisher: AnyPublisher<T, APIService.APIError>,
        onSuccess: @escaping (T) -> Void
    ) -> AnyCancellable {
        clearError()
        return
            publisher
            .handleErrorOnly(on: self)
            .sinkValue(on: self) { response in
                onSuccess(response)
            }
    }
}
