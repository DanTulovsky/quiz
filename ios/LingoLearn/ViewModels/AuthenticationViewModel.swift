import Foundation
import Combine

class AuthenticationViewModel: ObservableObject {
    @Published var username = ""
    @Published var password = ""
    @Published var email = ""
    @Published var isAuthenticated = false
    @Published var user: User? = nil
    @Published var error: APIService.APIError?

    var apiService: APIService
    private var cancellables = Set<AnyCancellable>()

    init(apiService: APIService = APIService.shared) {
        self.apiService = apiService
        apiService.authStatus()
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                guard let self else { return }
                switch completion {
                case .failure(let error):
                    self.error = error
                case .finished:
                    break
                }
            }, receiveValue: { [weak self] response in
                guard let self else { return }
                self.error = nil
                self.isAuthenticated = response.authenticated
                self.user = response.user
            })
            .store(in: &cancellables)
    }

    func login() {
        let loginRequest = LoginRequest(username: username, password: password)
        apiService.login(request: loginRequest)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                guard let self else { return }
                switch completion {
                case .failure(let error):
                    self.error = error
                    self.isAuthenticated = false
                case .finished:
                    break
                }
            }, receiveValue: { [weak self] response in
                guard let self else { return }
                self.error = nil
                self.isAuthenticated = response.success
                self.user = response.user
            })
            .store(in: &cancellables)
    }

    func signup() {
        let signupRequest = UserCreateRequest(username: username, email: email, password: password)
        apiService.signup(request: signupRequest)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                guard let self else { return }
                switch completion {
                case .failure(let error):
                    self.error = error
                case .finished:
                    break
                }
            }, receiveValue: { _ in
                // For simplicity, we'll just consider the signup successful
                // and the user can now login.
                // A better approach would be to automatically log the user in.
            })
            .store(in: &cancellables)
    }

    func logout() {
        apiService.logout()
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                guard let self else { return }
                switch completion {
                case .failure(let error):
                    self.error = error
                case .finished:
                    break
                }
            }, receiveValue: { [weak self] _ in
                guard let self else { return }
                self.error = nil
                self.isAuthenticated = false
            })
            .store(in: &cancellables)
    }
}
