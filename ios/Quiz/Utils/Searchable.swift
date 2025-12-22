import Combine
import Foundation

protocol Searchable: BaseViewModel {
    var searchQuery: String { get set }
    var searchQueryPublisher: Published<String>.Publisher { get }
    func performSearch()
}

extension Searchable where Self: ObservableObject {
    func setupSearchDebounce(delay: TimeInterval = 0.5) {
        searchQueryPublisher
            .debouncedSearch(on: self, delay: delay) { [weak self] in
                self?.performSearch()
            }
            .store(in: &cancellables)
    }
}
