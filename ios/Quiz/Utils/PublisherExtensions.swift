import Combine
import Foundation

extension AnyPublisher {
    func handleError<T: ObservableObject>(
        on object: T,
        errorPath: ReferenceWritableKeyPath<T, APIService.APIError?>,
        isLoadingPath: ReferenceWritableKeyPath<T, Bool>? = nil
    ) -> AnyPublisher<Output, Failure> where Failure == APIService.APIError {
        return
            self
            .receive(on: DispatchQueue.main)
            .handleEvents(receiveCompletion: { completion in
                if case .failure(let error) = completion {
                    object[keyPath: errorPath] = error
                    if let isLoadingPath = isLoadingPath {
                        object[keyPath: isLoadingPath] = false
                    }
                }
            })
            .eraseToAnyPublisher()
    }
}

extension Publisher where Failure == APIService.APIError {
    func retryOnTransientFailure(maxRetries: Int = 3, delay: TimeInterval = 1.0) -> AnyPublisher<
        Output, Failure
    > {
        guard maxRetries > 0 else {
            return self.eraseToAnyPublisher()
        }

        return self.catch { error -> AnyPublisher<Output, Failure> in
            // Only retry on network-related errors
            guard case .requestFailed(let underlyingError) = error else {
                return Fail(error: error).eraseToAnyPublisher()
            }

            // Check if it's a transient network error
            let nsError = underlyingError as NSError
            let isTransient =
                nsError.domain == NSURLErrorDomain
                && (nsError.code == NSURLErrorTimedOut
                    || nsError.code == NSURLErrorNetworkConnectionLost
                    || nsError.code == NSURLErrorNotConnectedToInternet
                    || nsError.code == NSURLErrorCannotConnectToHost
                    || nsError.code == NSURLErrorDNSLookupFailed)

            guard isTransient else {
                return Fail(error: error).eraseToAnyPublisher()
            }

            // Retry with exponential backoff
            // Use Deferred to ensure the publisher is recreated on each retry
            return Deferred {
                Just(())
            }
            .delay(for: .seconds(delay), scheduler: DispatchQueue.global())
            .flatMap { _ in
                self.retryOnTransientFailure(maxRetries: maxRetries - 1, delay: delay * 1.5)
            }
            .eraseToAnyPublisher()
        }
        .eraseToAnyPublisher()
    }
}

extension Publisher where Failure == APIService.APIError {
    func handleLoadingAndError<T: BaseViewModel>(
        on viewModel: T,
        clearErrorOnStart: Bool = true
    ) -> AnyPublisher<Output, Failure> {
        return
            self
            .receive(on: DispatchQueue.main)
            .handleEvents(
                receiveSubscription: { _ in
                    if clearErrorOnStart {
                        viewModel.clearError()
                    }
                    viewModel.isLoading = true
                },
                receiveCompletion: { completion in
                    viewModel.isLoading = false
                    if case .failure(let error) = completion {
                        viewModel.error = error
                    }
                }
            )
            .eraseToAnyPublisher()
    }

    func handleErrorOnly<T: BaseViewModel>(
        on viewModel: T
    ) -> AnyPublisher<Output, Failure> {
        return
            self
            .receive(on: DispatchQueue.main)
            .handleEvents(receiveCompletion: { completion in
                if case .failure(let error) = completion {
                    viewModel.error = error
                }
            })
            .eraseToAnyPublisher()
    }

    func sinkValue<T: BaseViewModel>(
        on viewModel: T,
        receiveValue: @escaping (Output) -> Void
    ) -> AnyCancellable {
        return
            self
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { _ in }, receiveValue: receiveValue)
    }
}

extension Publisher where Output == String {
    func debouncedSearch<T: BaseViewModel>(
        on viewModel: T,
        delay: TimeInterval = 0.5,
        action: @escaping () -> Void
    ) -> AnyCancellable {
        return
            self
            .debounce(for: .milliseconds(Int(delay * 1000)), scheduler: RunLoop.main)
            .removeDuplicates()
            .sink(receiveCompletion: { _ in }, receiveValue: { _ in action() })
    }
}

extension Publishers.Zip where A.Failure == APIService.APIError, B.Failure == APIService.APIError {
    func handleLoadingAndError<T: BaseViewModel>(
        on viewModel: T
    ) -> AnyPublisher<(A.Output, B.Output), APIService.APIError> {
        return
            self
            .receive(on: DispatchQueue.main)
            .handleEvents(
                receiveSubscription: { _ in
                    viewModel.isLoading = true
                },
                receiveCompletion: { completion in
                    viewModel.isLoading = false
                    if case .failure(let error) = completion {
                        viewModel.error = error
                    }
                }
            )
            .eraseToAnyPublisher()
    }

    func sinkValue<T: BaseViewModel>(
        on viewModel: T,
        receiveValue: @escaping ((A.Output, B.Output)) -> Void
    ) -> AnyCancellable {
        return
            self
            .sink(receiveCompletion: { _ in }, receiveValue: receiveValue)
    }
}

extension Publisher where Failure == APIService.APIError {
    func sinkVoid<T: BaseViewModel>(
        on viewModel: T,
        receiveValue: @escaping () -> Void
    ) -> AnyCancellable {
        return
            self
            .handleErrorOnly(on: viewModel)
            .sink(receiveCompletion: { _ in }, receiveValue: { _ in receiveValue() })
    }
}

extension Publisher where Failure == APIService.APIError {
    func executeWithCompletion<T: BaseViewModel>(
        on viewModel: T,
        receiveValue: @escaping (Output) -> Void,
        completion: @escaping (Result<Output, APIService.APIError>) -> Void
    ) -> AnyCancellable {
        return
            self
            .handleErrorOnly(on: viewModel)
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { result in
                    if case .failure(let error) = result {
                        completion(.failure(error))
                    }
                },
                receiveValue: { value in
                    receiveValue(value)
                    completion(.success(value))
                }
            )
    }
}

