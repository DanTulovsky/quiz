import Combine
import Foundation

class APIService: APIServiceProtocol {
    static let shared = APIService()
    let baseURL: URL

    enum APIError: Error, LocalizedError {
        case invalidURL
        case requestFailed(Error)
        case invalidResponse
        case decodingFailed(Error)
        case backendError(code: String?, message: String, details: String?)
        case encodingFailed(Error)

        var errorDescription: String? {
            switch self {
            case .invalidURL: return "Invalid URL"
            case .requestFailed(let error): return error.localizedDescription
            case .invalidResponse: return "Invalid response from server"
            case .decodingFailed(let error):
                return "Failed to decode response: \(error.localizedDescription)"
            case .encodingFailed(let error):
                return "Failed to encode request: \(error.localizedDescription)"
            case .backendError(let code, let message, _):
                if let code = code {
                    return "\(code): \(message)"
                }
                return message
            }
        }

        var errorCode: String? {
            switch self {
            case .backendError(let code, _, _): return code
            default: return nil
            }
        }

        var errorDetails: String? {
            switch self {
            case .backendError(_, _, let details): return details
            default: return nil
            }
        }
    }

    init() {
        #if targetEnvironment(simulator)
        guard let url = URL(string: "http://localhost:3000/v1") else {
            fatalError("Invalid base URL for simulator")
        }
        self.baseURL = url
        #else
        guard let url = URL(string: "https://quiz.wetsnow.com/v1") else {
            fatalError("Invalid base URL for production")
        }
        self.baseURL = url
        #endif
    }

    private static let decoder: JSONDecoder = {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            let dateStr = try container.decode(String.self)
            let iso8601WithFractional = ISO8601DateFormatter()
            iso8601WithFractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
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
    }()

    private var decoder: JSONDecoder {
        return APIService.decoder
    }

    func validateQueryItem(_ item: URLQueryItem) -> Bool {
        // Filter out empty values and validate non-empty values
        guard let value = item.value, !value.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        else {
            return false
        }
        return true
    }

    func buildURL(path: String, queryItems: [URLQueryItem]? = nil) -> Result<URL, APIError> {
        let pathComponents = path.split(separator: "/").map(String.init)
        var fullPath = baseURL
        for component in pathComponents {
            fullPath = fullPath.appendingPathComponent(component)
        }
        guard var components = URLComponents(url: fullPath, resolvingAgainstBaseURL: false) else {
            return .failure(.invalidURL)
        }
        if let queryItems = queryItems, !queryItems.isEmpty {
            // Filter out invalid query items (empty values)
            let validItems = queryItems.filter { validateQueryItem($0) }
            if !validItems.isEmpty {
                components.queryItems = validItems
            }
        }
        guard let url = components.url else {
            return .failure(.invalidURL)
        }
        return .success(url)
    }

    func encodeBody<T: Encodable>(_ value: T) -> Result<Data, APIError> {
        do {
            let data = try JSONEncoder().encode(value)
            return .success(data)
        } catch {
            return .failure(.encodingFailed(error))
        }
    }

    func encodeJSONBody(_ value: [String: Any]) -> Result<Data, APIError> {
        do {
            let data = try JSONSerialization.data(withJSONObject: value)
            return .success(data)
        } catch {
            return .failure(.encodingFailed(error))
        }
    }

    func encodingError(description: String) -> APIError {
        return .encodingFailed(
            NSError(
                domain: "APIService",
                code: -1,
                userInfo: [NSLocalizedDescriptionKey: description]
            )
        )
    }

    func authenticatedRequest(for url: URL, method: String = "GET", body: Data? = nil)
    -> URLRequest {
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = method
        urlRequest.httpShouldHandleCookies = true
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        urlRequest.httpBody = body
        urlRequest.timeoutInterval = 30.0
        return urlRequest
    }

    func handleResponse<T: Decodable>(_ data: Data, _ response: URLResponse)
    -> AnyPublisher<T, APIError> {
        guard let httpResponse = response as? HTTPURLResponse else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }

        if (200...299).contains(httpResponse.statusCode) {
            // For SnippetList, allow empty responses (204 No Content or empty body)
            if T.self == SnippetList.self {
                if data.isEmpty || httpResponse.statusCode == 204 {
                    let emptyList = SnippetList(limit: 0, offset: 0, query: nil, snippets: [])
                    guard let result = emptyList as? T else {
                        return Fail(error: .invalidResponse).eraseToAnyPublisher()
                    }
                    return Just(result)
                        .setFailureType(to: APIError.self)
                        .eraseToAnyPublisher()
                }
            }
            // Check for empty data before attempting to decode (for other types)
            guard !data.isEmpty else {
                return Fail(
                    error: .decodingFailed(
                        NSError(
                            domain: "APIService",
                            code: -1,
                            userInfo: [
                                NSLocalizedDescriptionKey:
                                    "The data couldn't be read because it is missing."
                            ]
                        )
                    )
                ).eraseToAnyPublisher()
            }
            return Just(data)
                .decode(type: T.self, decoder: self.decoder)
                .mapError { .decodingFailed($0) }
                .eraseToAnyPublisher()
        } else {
            if let errorResp = try? self.decoder.decode(ErrorResponse.self, from: data),
               let msg = errorResp.message ?? errorResp.error {
                return Fail(
                    error: .backendError(
                        code: errorResp.code, message: msg, details: errorResp.details)
                ).eraseToAnyPublisher()
            }
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
    }

    private func handleErrorResponse<T>(data: Data) -> AnyPublisher<T, APIError> {
        if let errorResp = try? self.decoder.decode(ErrorResponse.self, from: data),
           let msg = errorResp.message ?? errorResp.error {
            return Fail(
                error: .backendError(code: errorResp.code, message: msg, details: errorResp.details)
            )
            .eraseToAnyPublisher()
        }
        return Fail(error: .invalidResponse).eraseToAnyPublisher()
    }

    enum QuestionFetchResult {
        case question(Question)
        case generating(GeneratingStatusResponse)
    }

}
