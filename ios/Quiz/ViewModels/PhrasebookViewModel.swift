import Foundation
import Combine

class PhrasebookViewModel: BaseViewModel {
    @Published var categories: [PhrasebookCategoryInfo] = []
    @Published var selectedCategoryData: PhrasebookData?

    func fetchCategories() {
        self.categories = PhrasebookService.shared.loadCategories()
    }

    func fetchCategoryData(id: String) {
        self.isLoading = true
        self.selectedCategoryData = PhrasebookService.shared.loadCategoryData(id: id)
        self.isLoading = false
    }
}
