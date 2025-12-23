import Combine
import Foundation

extension APIService {
    func get<T: Decodable>(
        path: String,
        queryItems: [URLQueryItem]? = nil,
        responseType: T.Type
    ) -> AnyPublisher<T, APIError> {
        guard case .success(let url) = buildURL(path: path, queryItems: queryItems) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let request = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: request)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func post<T: Decodable, U: Encodable>(
        path: String,
        body: U,
        responseType: T.Type,
        retry: Bool = false
    ) -> AnyPublisher<T, APIError> {
        guard case .success(let url) = buildURL(path: path) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        guard case .success(let bodyData) = encodeBody(body) else {
            return Fail(error: encodingError(description: "Failed to encode request"))
                .eraseToAnyPublisher()
        }
        let request = authenticatedRequest(for: url, method: "POST", body: bodyData)
        var publisher: AnyPublisher<T, APIError> = URLSession.shared.dataTaskPublisher(for: request)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()

        if retry {
            publisher = publisher.retryOnTransientFailure(maxRetries: 2)
        }
        return publisher
    }

    func put<T: Decodable, U: Encodable>(
        path: String,
        body: U,
        responseType: T.Type
    ) -> AnyPublisher<T, APIError> {
        guard case .success(let url) = buildURL(path: path) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        guard case .success(let bodyData) = encodeBody(body) else {
            return Fail(error: encodingError(description: "Failed to encode request"))
                .eraseToAnyPublisher()
        }
        let request = authenticatedRequest(for: url, method: "PUT", body: bodyData)
        return URLSession.shared.dataTaskPublisher(for: request)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func delete<T: Decodable>(
        path: String,
        responseType: T.Type
    ) -> AnyPublisher<T, APIError> {
        guard case .success(let url) = buildURL(path: path) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let request = authenticatedRequest(for: url, method: "DELETE")
        return URLSession.shared.dataTaskPublisher(for: request)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func deleteVoid(path: String) -> AnyPublisher<Void, APIError> {
        guard case .success(let url) = buildURL(path: path) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let request = authenticatedRequest(for: url, method: "DELETE")
        return URLSession.shared.dataTaskPublisher(for: request)
            .mapError { .requestFailed($0) }
            .flatMap { _, response -> AnyPublisher<Void, APIError> in
                guard let httpResponse = response as? HTTPURLResponse else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }
                if (200...299).contains(httpResponse.statusCode) {
                    return Just(()).setFailureType(to: APIError.self).eraseToAnyPublisher()
                }
                return Fail(error: .invalidResponse).eraseToAnyPublisher()
            }
            .eraseToAnyPublisher()
    }

    private func jsonRequest<T: Decodable>(
        path: String,
        method: String,
        body: [String: Any],
        responseType: T.Type
    ) -> AnyPublisher<T, APIError> {
        guard case .success(let url) = buildURL(path: path) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        guard case .success(let bodyData) = encodeJSONBody(body) else {
            return Fail(error: encodingError(description: "Failed to encode request"))
                .eraseToAnyPublisher()
        }
        let request = authenticatedRequest(for: url, method: method, body: bodyData)
        return URLSession.shared.dataTaskPublisher(for: request)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func postJSON<T: Decodable>(
        path: String,
        body: [String: Any],
        responseType: T.Type
    ) -> AnyPublisher<T, APIError> {
        return jsonRequest(path: path, method: "POST", body: body, responseType: responseType)
    }

    func putJSON<T: Decodable>(
        path: String,
        body: [String: Any],
        responseType: T.Type
    ) -> AnyPublisher<T, APIError> {
        return jsonRequest(path: path, method: "PUT", body: body, responseType: responseType)
    }

    func deleteJSON<T: Decodable>(
        path: String,
        body: [String: Any],
        responseType: T.Type
    ) -> AnyPublisher<T, APIError> {
        return jsonRequest(path: path, method: "DELETE", body: body, responseType: responseType)
    }

    func postVoid(path: String) -> AnyPublisher<SuccessResponse, APIError> {
        guard case .success(let url) = buildURL(path: path) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let request = authenticatedRequest(for: url, method: "POST")
        return URLSession.shared.dataTaskPublisher(for: request)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func clearSessionCookie() {
        guard let cookies = HTTPCookieStorage.shared.cookies else { return }
        for cookie in cookies where cookie.name == "quiz-session" {
            HTTPCookieStorage.shared.deleteCookie(cookie)
        }
    }

    func storeCookies(from response: HTTPURLResponse) {
        guard let url = response.url else { return }
        let headerFields = response.allHeaderFields
        var cookieHeaders: [String: String] = [:]
        for (key, value) in headerFields {
            if let keyString = key as? String, let valueString = value as? String {
                cookieHeaders[keyString] = valueString
            }
        }
        let cookies = HTTPCookie.cookies(withResponseHeaderFields: cookieHeaders, for: url)
        for cookie in cookies {
            HTTPCookieStorage.shared.setCookie(cookie)
        }
    }
}
