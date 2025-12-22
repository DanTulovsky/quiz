import Foundation
import Combine

protocol CRUDOperations: BaseViewModel {
    associatedtype CRUDItem: Identifiable
    var crudItems: [CRUDItem] { get set }
}

extension CRUDOperations {
    func createItem<U: Decodable>(
        publisher: AnyPublisher<U, APIService.APIError>,
        transform: @escaping (U) -> CRUDItem,
        insertAt: Int = 0,
        completion: @escaping (Result<U, APIService.APIError>) -> Void
    ) -> AnyCancellable {
        return publisher
            .receive(on: DispatchQueue.main)
            .executeWithCompletion(
                on: self,
                receiveValue: { [weak self] newItem in
                    guard let self = self else { return }
                    let item = transform(newItem)
                    if insertAt == 0 {
                        self.crudItems.insert(item, at: 0)
                    } else {
                        self.crudItems.append(item)
                    }
                },
                completion: completion
            )
    }

    func updateItem<U: Decodable>(
        id: CRUDItem.ID,
        publisher: AnyPublisher<U, APIService.APIError>,
        transform: @escaping (U) -> CRUDItem,
        completion: @escaping (Result<U, APIService.APIError>) -> Void
    ) -> AnyCancellable {
        return publisher
            .receive(on: DispatchQueue.main)
            .executeWithCompletion(
                on: self,
                receiveValue: { [weak self] updatedItem in
                    guard let self = self else { return }
                    let item = transform(updatedItem)
                    if let index = self.crudItems.firstIndex(where: { $0.id == id }) {
                        self.crudItems[index] = item
                    }
                },
                completion: completion
            )
    }

    func deleteItem(
        id: CRUDItem.ID,
        publisher: AnyPublisher<Void, APIService.APIError>,
        completion: @escaping (Result<Void, APIService.APIError>) -> Void
    ) -> AnyCancellable {
        return publisher
            .receive(on: DispatchQueue.main)
            .executeWithCompletion(
                on: self,
                receiveValue: { [weak self] _ in
                    guard let self = self else { return }
                    self.crudItems.removeAll { $0.id == id }
                },
                completion: completion
            )
    }
}

