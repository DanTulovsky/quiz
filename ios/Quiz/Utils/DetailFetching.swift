import Foundation
import Combine

protocol DetailFetching: BaseViewModel {
    associatedtype DetailID
    associatedtype DetailItem
    var selectedDetail: DetailItem? { get set }
    func fetchDetailPublisher(id: DetailID) -> AnyPublisher<DetailItem, APIService.APIError>
}

extension DetailFetching {
    func fetchDetail(id: DetailID) {
        fetchDetailPublisher(id: id)
            .handleLoadingAndError(on: self)
            .sinkValue(on: self) { [weak self] detail in
                self?.selectedDetail = detail
            }
            .store(in: &cancellables)
    }
}

