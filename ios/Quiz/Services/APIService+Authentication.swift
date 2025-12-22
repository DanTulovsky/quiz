import Combine
import Foundation

extension APIService {
    func login(request: LoginRequest) -> AnyPublisher<LoginResponse, APIError> {
        return post(
            path: "auth/login", body: request, responseType: LoginResponse.self, retry: true)
    }

    func signup(request: UserCreateRequest) -> AnyPublisher<SuccessResponse, APIError> {
        return post(path: "auth/signup", body: request, responseType: SuccessResponse.self)
    }

    func authStatus() -> AnyPublisher<AuthStatusResponse, APIError> {
        return get(path: "auth/status", responseType: AuthStatusResponse.self)
            .retryOnTransientFailure(maxRetries: 2)
            .eraseToAnyPublisher()
    }

    func initiateGoogleLogin() -> AnyPublisher<GoogleOAuthLoginResponse, APIError> {
        var params = QueryParameters()
        params.add("platform", value: "ios")
        return get(
            path: "auth/google/login", queryItems: params.build(),
            responseType: GoogleOAuthLoginResponse.self)
    }

    func handleGoogleCallback(code: String, state: String?) -> AnyPublisher<LoginResponse, APIError> {
        print(
            "🌐 Making callback API request to: \(baseURL.appendingPathComponent("auth/google/callback"))"
        )
        print("📝 Code: \(code.prefix(10))..., State: \(state ?? "nil")")

        var params = QueryParameters()
        params.add("code", value: code)
        params.add("state", value: state)
        guard
            case .success(let url) = buildURL(
                path: "auth/google/callback", queryItems: params.build())
        else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        var urlRequest = authenticatedRequest(for: url)
        urlRequest.setValue("application/json", forHTTPHeaderField: "Accept")

        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { data, response -> AnyPublisher<LoginResponse, APIError> in
                self.handleGoogleCallbackResponse(data: data, response: response)
            }
            .eraseToAnyPublisher()
    }

    private func handleGoogleCallbackResponse(
        data: Data, response: URLResponse
    ) -> AnyPublisher<LoginResponse, APIError> {
        if let httpResponse = response as? HTTPURLResponse {
            self.storeCookies(from: httpResponse)
        }
        guard let httpResponse = response as? HTTPURLResponse else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        if (200...299).contains(httpResponse.statusCode) {
            return decodeLoginResponse(data: data)
        } else {
            return decodeErrorResponse(data: data)
        }
    }

    private func decodeLoginResponse(data: Data) -> AnyPublisher<LoginResponse, APIError> {
        let decoder = createDateDecoder()
        return Just(data)
            .decode(type: LoginResponse.self, decoder: decoder)
            .mapError { .decodingFailed($0) }
            .eraseToAnyPublisher()
    }

    private func decodeErrorResponse(data: Data) -> AnyPublisher<LoginResponse, APIError> {
        let decoder = JSONDecoder()
        if let errorResp = try? decoder.decode(ErrorResponse.self, from: data),
           let msg = errorResp.message ?? errorResp.error {
            return Fail(
                error: .backendError(
                    code: errorResp.code, message: msg, details: errorResp.details)
            ).eraseToAnyPublisher()
        }
        return Fail(error: .invalidResponse).eraseToAnyPublisher()
    }

    private func createDateDecoder() -> JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            let dateStr = try container.decode(String.self)
            let iso8601WithFractional = ISO8601DateFormatter()
            iso8601WithFractional.formatOptions = [
                .withInternetDateTime, .withFractionalSeconds
            ]
            let iso8601Standard = ISO8601DateFormatter()
            iso8601Standard.formatOptions = [.withInternetDateTime]
            if let date = iso8601WithFractional.date(from: dateStr)
                ?? iso8601Standard.date(from: dateStr) {
                return date
            }
            throw DecodingError.dataCorruptedError(
                in: container, debugDescription: "Invalid date format: \(dateStr)")
        }
        return decoder
    }

    func updateUser(request: UserUpdateRequest) -> AnyPublisher<User, APIError> {
        return put(
            path: "userz/profile", body: request, responseType: UserProfileMessageResponse.self
        )
        .map { $0.user }
        .eraseToAnyPublisher()
    }
}
