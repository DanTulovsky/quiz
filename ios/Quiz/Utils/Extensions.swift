import Foundation
import Combine

extension Array where Element == LanguageInfo {
    func find(byCodeOrName codeOrName: String) -> LanguageInfo? {
        let lowercased = codeOrName.lowercased()
        return first(where: {
            $0.name.lowercased() == lowercased || $0.code.lowercased() == lowercased
        })
    }
}

extension AnyPublisher {
    func handleError<T: ObservableObject>(
        on object: T,
        errorPath: ReferenceWritableKeyPath<T, APIService.APIError?>,
        isLoadingPath: ReferenceWritableKeyPath<T, Bool>? = nil
    ) -> AnyPublisher<Output, Failure> where Failure == APIService.APIError {
        return self
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
    func retryOnTransientFailure(maxRetries: Int = 3, delay: TimeInterval = 1.0) -> AnyPublisher<Output, Failure> {
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
            let isTransient = nsError.domain == NSURLErrorDomain && (
                nsError.code == NSURLErrorTimedOut ||
                nsError.code == NSURLErrorNetworkConnectionLost ||
                nsError.code == NSURLErrorNotConnectedToInternet ||
                nsError.code == NSURLErrorCannotConnectToHost ||
                nsError.code == NSURLErrorDNSLookupFailed
            )

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
        on viewModel: T
    ) -> AnyPublisher<Output, Failure> {
        return self
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

    func handleErrorOnly<T: BaseViewModel>(
        on viewModel: T
    ) -> AnyPublisher<Output, Failure> {
        return self
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
        return self
            .sink(receiveCompletion: { _ in }, receiveValue: receiveValue)
    }
}

extension APIService {
    func getSnippetsForQuestion(questionId: Int) -> AnyPublisher<SnippetList, APIError> {
        return getSnippets(sourceLang: nil, targetLang: nil, storyId: nil, query: nil, level: nil)
    }

    func getSnippetsForStory(storyId: Int) -> AnyPublisher<SnippetList, APIError> {
        return getSnippets(sourceLang: nil, targetLang: nil, storyId: storyId, query: nil, level: nil)
    }
}

