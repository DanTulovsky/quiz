import Foundation
import Combine

protocol CRUDOperations: BaseViewModel {
    associatedtype Item: Identifiable
    var crudItems: [Item] { get set }
}

extension CRUDOperations {
    func createItem<U: Decodable>(
        publisher: AnyPublisher<U, APIService.APIError>,
        transform: @escaping (U) -> Item,
        insertAt: Int = 0,
        completion: @escaping (Result<U, APIService.APIError>) -> Void
    ) -> AnyCancellable {
        return publisher
            .executeWithCompletion(
                on: self,
                receiveValue: { [weak self] newItem in
                    let item = transform(newItem)
                    if insertAt == 0 {
                        self?.crudItems.insert(item, at: 0)
                    } else {
                        self?.crudItems.append(item)
                    }
                },
                completion: completion
            )
    }

    func updateItem<U: Decodable>(
        id: Item.ID,
        publisher: AnyPublisher<U, APIService.APIError>,
        transform: @escaping (U) -> Item,
        completion: @escaping (Result<U, APIService.APIError>) -> Void
    ) -> AnyCancellable {
        return publisher
            .executeWithCompletion(
                on: self,
                receiveValue: { [weak self] updatedItem in
                    let item = transform(updatedItem)
                    if let index = self?.crudItems.firstIndex(where: { $0.id == id }) {
                        self?.crudItems[index] = item
                    }
                },
                completion: completion
            )
    }

    func deleteItem(
        id: Item.ID,
        publisher: AnyPublisher<Void, APIService.APIError>,
        completion: @escaping (Result<Void, APIService.APIError>) -> Void
    ) -> AnyCancellable {
        return publisher
            .executeWithCompletion(
                on: self,
                receiveValue: { [weak self] _ in
                    self?.crudItems.removeAll { $0.id == id }
                },
                completion: completion
            )
    }
}

