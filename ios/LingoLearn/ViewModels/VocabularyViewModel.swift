import Foundation
import Combine

class VocabularyViewModel: ObservableObject {
    @Published var snippets = [Snippet]()
    @Published var error: APIService.APIError?
    
    var apiService: APIService
    private var cancellables = Set<AnyCancellable>()
    
    init(apiService: APIService = APIService.shared) {
        self.apiService = apiService
    }
    
    func getSnippets() {
        apiService.getSnippets(sourceLang: nil, targetLang: nil)
            .sink(receiveCompletion: { completion in
                switch completion {
                case .failure(let error):
                    self.error = error
                case .finished:
                    break
                }
            }, receiveValue: { snippetList in
                self.snippets = snippetList.snippets
            })
            .store(in: &cancellables)
    }
}
