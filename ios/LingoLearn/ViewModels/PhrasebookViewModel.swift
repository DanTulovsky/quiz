import Foundation
import Combine

class PhrasebookViewModel: ObservableObject {
    @Published var categories: [PhrasebookCategoryInfo] = []
    @Published var selectedCategoryData: PhrasebookData?
    @Published var isLoading = false

    func fetchCategories() {
        self.categories = PhrasebookService.shared.loadCategories()
    }

    func fetchCategoryData(id: String) {
        self.isLoading = true
        self.selectedCategoryData = PhrasebookService.shared.loadCategoryData(id: id)
        self.isLoading = false
    }
}
