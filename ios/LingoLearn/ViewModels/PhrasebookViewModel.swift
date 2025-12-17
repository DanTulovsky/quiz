import Foundation
import Combine

class PhrasebookViewModel: ObservableObject {
    @Published var phrasebook: PhrasebookResponse?
    @Published var error: APIService.APIError?
    
    var apiService: APIService
    private var cancellables = Set<AnyCancellable>()
    
    init(apiService: APIService = APIService.shared) {
        self.apiService = apiService
    }
    
    func getPhrasebook(language: Language) {
        apiService.getPhrasebook(language: language)
            .sink(receiveCompletion: { completion in
                switch completion {
                case .failure(let error):
                    self.error = error
                case .finished:
                    break
                }
            }, receiveValue: { phrasebook in
                self.phrasebook = phrasebook
            })
            .store(in: &cancellables)
    }
}
