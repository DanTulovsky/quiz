import Combine
import Foundation

extension APIService {
    func logout() -> AnyPublisher<SuccessResponse, APIError> {
        return postVoid(path: "auth/logout")
            .handleEvents(receiveOutput: { _ in
                self.clearSessionCookie()
            })
            .eraseToAnyPublisher()
    }
}
