import Combine
import Foundation

extension APIService {
    func translateText(request: TranslateRequest) -> AnyPublisher<TranslateResponse, APIError> {
        return post(path: "translate", body: request, responseType: TranslateResponse.self)
    }
}

