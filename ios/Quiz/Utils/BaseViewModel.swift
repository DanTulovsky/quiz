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
        // Cancel all subscriptions synchronously on the main queue to avoid deadlocks
        // when deallocating while publishers are still processing on the main queue
        if Thread.isMainThread {
            cancellables.removeAll()
        } else {
            DispatchQueue.main.sync {
                cancellables.removeAll()
            }
        }
    }

    func handleError(_ error: APIService.APIError) {
        self.error = error
        self.isLoading = false
    }

    func clearError() {
        self.error = nil
    }
}

