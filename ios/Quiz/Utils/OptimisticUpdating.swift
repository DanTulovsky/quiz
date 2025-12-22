import Foundation
import Combine

protocol OptimisticUpdating: BaseViewModel {
    associatedtype Item: Identifiable
    var items: [Item] { get set }
}

extension OptimisticUpdating {
    func applyOptimisticUpdate(id: Item.ID, update: @escaping (Item) -> Item) {
        if let index = items.firstIndex(where: { $0.id == id }) {
            let oldItem = items[index]
            let updatedItem = update(oldItem)
            var updatedItems = items
            updatedItems[index] = updatedItem
            items = updatedItems
        }
    }
}
