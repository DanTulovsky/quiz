import Combine
import Foundation

extension APIService {
    func getSnippetsForQuestion(questionId: Int) -> AnyPublisher<SnippetList, APIError> {
        return getSnippetsByQuestion(questionId: questionId)
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
