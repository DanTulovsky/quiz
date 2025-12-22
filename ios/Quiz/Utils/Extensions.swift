import Combine
import Foundation

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

extension APIService {
    func getSnippetsForQuestion(questionId: Int) -> AnyPublisher<SnippetList, APIError> {
        return getSnippets(sourceLang: nil, targetLang: nil, storyId: nil, query: nil, level: nil)
    }

    func getSnippetsForStory(storyId: Int) -> AnyPublisher<SnippetList, APIError> {
        return getSnippets(
            sourceLang: nil, targetLang: nil, storyId: storyId, query: nil, level: nil)
    }
}

protocol ListFetching: BaseViewModel {
    associatedtype Item
    var items: [Item] { get set }
    func fetchItemsPublisher() -> AnyPublisher<[Item], APIService.APIError>
}

extension ListFetching {
    func fetchItems() {
        fetchItemsPublisher()
            .handleLoadingAndError(on: self)
            .sinkValue(on: self) { [weak self] items in
                self?.items = items
            }
            .store(in: &cancellables)
    }
}

protocol ListFetchingWithName: BaseViewModel {
    associatedtype Item
    var items: [Item] { get set }
    func fetchItemsPublisher() -> AnyPublisher<[Item], APIService.APIError>
    func updateItems(_ items: [Item])
}

extension ListFetchingWithName {
    func fetchItems() {
        fetchItemsPublisher()
            .handleLoadingAndError(on: self)
            .sinkValue(on: self) { [weak self] items in
                self?.updateItems(items)
            }
            .store(in: &cancellables)
    }
}
