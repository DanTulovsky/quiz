import Foundation
import Combine

class BaseViewModel: ObservableObject {
    @Published var error: APIService.APIError?
    @Published var isLoading = false

    var apiService: APIService
    var cancellables = Set<AnyCancellable>()

    init(apiService: APIService = APIService.shared) {
        self.apiService = apiService
    }

    func cancelAllRequests() {
        cancellables.removeAll()
    }

    deinit {
        cancelAllRequests()
    }

    func handleError(_ error: APIService.APIError) {
        self.error = error
        self.isLoading = false
    }

    func clearError() {
        self.error = nil
    }
}

