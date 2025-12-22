import Combine
import Foundation

class PhrasebookViewModel: BaseViewModel {
    @Published var categories: [PhrasebookCategoryInfo] = []
    @Published var selectedCategoryData: PhrasebookData?

    func fetchCategories() {
        self.categories = PhrasebookService.shared.loadCategories()
    }

    func fetchCategoryData(id: String) {
        selectedCategoryData = PhrasebookService.shared.loadCategoryData(id: id)
    }
}
