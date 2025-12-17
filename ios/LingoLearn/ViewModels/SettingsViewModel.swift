import Foundation
import Combine

class SettingsViewModel: ObservableObject {
    @Published var user: User?
    @Published var error: APIService.APIError?

    var apiService: APIService
    private var cancellables = Set<AnyCancellable>()

    init(apiService: APIService = APIService.shared) {
        self.apiService = apiService
    }

    func updateUser(username: String?, email: String?, language: Language?, level: Level?) {
        let request = UserUpdateRequest(username: username, email: email, language: language, level: level)
        apiService.updateUser(request: request)
            .sink(receiveCompletion: { completion in
                switch completion {
                case .failure(let error):
                    self.error = error
                case .finished:
                    break
                }
            }, receiveValue: { user in
                self.user = user
            })
            .store(in: &cancellables)
    }
}
