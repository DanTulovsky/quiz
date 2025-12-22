import Combine
import Foundation

class PhrasebookViewModel: BaseViewModel, LanguageFetching {
    @Published var categories: [PhrasebookCategoryInfo] = []
    @Published var selectedCategoryData: PhrasebookData?
    @Published var availableLanguages: [LanguageInfo] = []

    func fetchCategories() {
        isLoading = true
        clearError()

        // PhrasebookService loads from local JSON synchronously
        categories = PhrasebookService.shared.loadCategories()
        isLoading = false
        if categories.isEmpty {
            error = APIService.APIError.decodingFailed(
                NSError(
                    domain: "PhrasebookService", code: -1,
                    userInfo: [
                        NSLocalizedDescriptionKey: "Failed to load phrasebook categories"
                    ])
            )
        }
    }

    func fetchCategoryData(id: String) {
        isLoading = true
        clearError()

        // PhrasebookService loads from local JSON synchronously
        selectedCategoryData = PhrasebookService.shared.loadCategoryData(id: id)
        isLoading = false
        if selectedCategoryData == nil {
            error = APIService.APIError.decodingFailed(
                NSError(
                    domain: "PhrasebookService", code: -1,
                    userInfo: [NSLocalizedDescriptionKey: "Failed to load category data"])
            )
        }
    }
}
