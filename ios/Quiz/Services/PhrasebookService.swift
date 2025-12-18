import Foundation

class PhrasebookService {
    static let shared = PhrasebookService()
    
    private var fullPhrasebook: [FullPhrasebookCategory] = []
    
    private init() {
        loadFullPhrasebook()
    }
    
    private func loadFullPhrasebook() {
        guard let url = Bundle.main.url(forResource: "phrasebook_full", withExtension: "json") else {
            print("phrasebook_full.json not found in bundle")
            return
        }
        
        do {
            let data = try Data(contentsOf: url)
            self.fullPhrasebook = try JSONDecoder().decode([FullPhrasebookCategory].self, from: data)
        } catch {
            print("Failed to decode phrasebook_full.json: \(error)")
        }
    }
    
    func loadCategories() -> [PhrasebookCategoryInfo] {
        return fullPhrasebook.map { $0.info }
    }
    
    func loadCategoryData(id: String) -> PhrasebookData? {
        return fullPhrasebook.first(where: { $0.info.id == id })?.data
    }
}
