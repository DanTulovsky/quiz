import Combine
import Foundation

class APIService {
    static let shared = APIService()
    private let baseURL: URL

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
                ?? iso8601Standard.date(from: dateStr)
            {
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

    private func validateQueryItem(_ item: URLQueryItem) -> Bool {
        // Filter out empty values and validate non-empty values
        guard let value = item.value, !value.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            return false
        }
        return true
    }

    private func buildURL(path: String, queryItems: [URLQueryItem]? = nil) -> Result<URL, APIError> {
        let fullPath = baseURL.appendingPathComponent(path)
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

    private func encodeBody<T: Encodable>(_ value: T) -> Result<Data, APIError> {
        do {
            let data = try JSONEncoder().encode(value)
            return .success(data)
        } catch {
            return .failure(.encodingFailed(error))
        }
    }

    private func encodeJSONBody(_ value: [String: Any]) -> Result<Data, APIError> {
        do {
            let data = try JSONSerialization.data(withJSONObject: value)
            return .success(data)
        } catch {
            return .failure(.encodingFailed(error))
        }
    }

    private func authenticatedRequest(for url: URL, method: String = "GET", body: Data? = nil) -> URLRequest {
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = method
        urlRequest.httpShouldHandleCookies = true
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        urlRequest.httpBody = body
        urlRequest.timeoutInterval = 30.0
        return urlRequest
    }

    private func handleResponse<T: Decodable>(_ data: Data, _ response: URLResponse)
        -> AnyPublisher<T, APIError>
    {
        guard let httpResponse = response as? HTTPURLResponse else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }

        if (200...299).contains(httpResponse.statusCode) {
            return Just(data)
                .decode(type: T.self, decoder: self.decoder)
                .mapError { .decodingFailed($0) }
                .eraseToAnyPublisher()
        } else {
            if let errorResp = try? self.decoder.decode(ErrorResponse.self, from: data),
                let msg = errorResp.message ?? errorResp.error
            {
                return Fail(error: .backendError(code: errorResp.code, message: msg, details: errorResp.details)).eraseToAnyPublisher()
            }
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
    }

    func login(request: LoginRequest) -> AnyPublisher<LoginResponse, APIError> {
        let url = baseURL.appendingPathComponent("auth/login")
        guard case .success(let body) = encodeBody(request) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode login request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "POST", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .retryOnTransientFailure(maxRetries: 2)
            .eraseToAnyPublisher()
    }

    func initiateGoogleLogin() -> AnyPublisher<GoogleOAuthLoginResponse, APIError> {
        let queryItems = [URLQueryItem(name: "platform", value: "ios")]
        guard case .success(let url) = buildURL(path: "auth/google/login", queryItems: queryItems) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func handleGoogleCallback(code: String, state: String?) -> AnyPublisher<LoginResponse, APIError>
    {
        print(
            "🌐 Making callback API request to: \(baseURL.appendingPathComponent("auth/google/callback"))"
        )
        print("📝 Code: \(code.prefix(10))..., State: \(state ?? "nil")")

        var queryItems = [URLQueryItem(name: "code", value: code)]
        if let state = state {
            queryItems.append(URLQueryItem(name: "state", value: state))
        }
        guard case .success(let url) = buildURL(path: "auth/google/callback", queryItems: queryItems) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        var urlRequest = authenticatedRequest(for: url)
        urlRequest.setValue("application/json", forHTTPHeaderField: "Accept")

        // Use URLSession.shared to ensure cookies are shared with other API calls
        // This ensures the session cookie from OAuth is available for subsequent requests
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { error in
                return .requestFailed(error)
            }
            .flatMap { data, response -> AnyPublisher<LoginResponse, APIError> in
                if let httpResponse = response as? HTTPURLResponse {
                    // Ensure cookies from the OAuth callback are stored
                    if let url = httpResponse.url {
                        let headerFields = httpResponse.allHeaderFields
                        var cookieHeaders: [String: String] = [:]
                        for (key, value) in headerFields {
                            if let keyString = key as? String, let valueString = value as? String {
                                cookieHeaders[keyString] = valueString
                            }
                        }
                        let cookies = HTTPCookie.cookies(
                            withResponseHeaderFields: cookieHeaders, for: url)
                        for cookie in cookies {
                            HTTPCookieStorage.shared.setCookie(cookie)
                        }
                    }
                }
                return self.handleResponse(data, response)
            }
            .eraseToAnyPublisher()
    }

    func authStatus() -> AnyPublisher<AuthStatusResponse, APIError> {
        let url = baseURL.appendingPathComponent("auth/status")
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .retryOnTransientFailure(maxRetries: 2)
            .eraseToAnyPublisher()
    }

    func logout() -> AnyPublisher<SuccessResponse, APIError> {
        let url = baseURL.appendingPathComponent("auth/logout")
        let urlRequest = authenticatedRequest(for: url, method: "POST")
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { (data, response) -> AnyPublisher<SuccessResponse, APIError> in
                self.handleResponse(data, response)
            }
            .handleEvents(receiveOutput: { _ in
                guard let cookies = HTTPCookieStorage.shared.cookies else { return }
                for cookie in cookies where cookie.name == "quiz-session" {
                    HTTPCookieStorage.shared.deleteCookie(cookie)
                }
            })
            .eraseToAnyPublisher()
    }

    func signup(request: UserCreateRequest) -> AnyPublisher<SuccessResponse, APIError> {
        let url = baseURL.appendingPathComponent("auth/signup")
        guard case .success(let body) = encodeBody(request) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode signup request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "POST", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    enum QuestionFetchResult {
        case question(Question)
        case generating(GeneratingStatusResponse)
    }

    func getQuestion(language: Language?, level: Level?, type: String?, excludeType: String?)
        -> AnyPublisher<QuestionFetchResult, APIError>
    {
        var queryItems = [URLQueryItem]()
        if let language = language {
            queryItems.append(URLQueryItem(name: "language", value: language.rawValue))
        }
        if let level = level {
            queryItems.append(URLQueryItem(name: "level", value: level.rawValue))
        }
        if let type = type { queryItems.append(URLQueryItem(name: "type", value: type)) }
        if let excludeType = excludeType {
            queryItems.append(URLQueryItem(name: "exclude_type", value: excludeType))
        }
        guard case .success(let url) = buildURL(path: "quiz/question", queryItems: queryItems.isEmpty ? nil : queryItems) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { data, response -> AnyPublisher<QuestionFetchResult, APIError> in
                guard let httpResponse = response as? HTTPURLResponse else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }
                if httpResponse.statusCode == 200 {
                    return Just(data).decode(type: Question.self, decoder: self.decoder).map(
                        QuestionFetchResult.question
                    ).mapError { .decodingFailed($0) }.eraseToAnyPublisher()
                } else if httpResponse.statusCode == 202 {
                    return Just(data).decode(
                        type: GeneratingStatusResponse.self, decoder: self.decoder
                    ).map(QuestionFetchResult.generating).mapError { .decodingFailed($0) }
                        .eraseToAnyPublisher()
                } else {
                    if let errorResp = try? self.decoder.decode(ErrorResponse.self, from: data),
                        let msg = errorResp.message ?? errorResp.error
                    {
                        return Fail(error: .backendError(code: errorResp.code, message: msg, details: errorResp.details)).eraseToAnyPublisher()
                    }
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }
            }
            .eraseToAnyPublisher()
    }

    func postAnswer(request: AnswerRequest) -> AnyPublisher<AnswerResponse, APIError> {
        let url = baseURL.appendingPathComponent("quiz/answer")
        guard case .success(let body) = encodeBody(request) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode answer request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "POST", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getStories() -> AnyPublisher<[StorySummary], APIError> {
        let url = baseURL.appendingPathComponent("story")
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getStory(id: Int) -> AnyPublisher<StoryContent, APIError> {
        let url = baseURL.appendingPathComponent("story/\(id)")
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getSnippets(sourceLang: String?, targetLang: String?, storyId: Int? = nil, query: String? = nil, level: String? = nil)
        -> AnyPublisher<SnippetList, APIError>
    {
        var queryItems = [URLQueryItem]()
        if let sourceLang = sourceLang {
            queryItems.append(URLQueryItem(name: "source_lang", value: sourceLang))
        }
        if let targetLang = targetLang {
            queryItems.append(URLQueryItem(name: "target_lang", value: targetLang))
        }
        if let storyId = storyId {
            queryItems.append(URLQueryItem(name: "story_id", value: String(storyId)))
        }
        if let query = query, !query.isEmpty {
            queryItems.append(URLQueryItem(name: "q", value: query))
        }
        if let level = level, !level.isEmpty {
            queryItems.append(URLQueryItem(name: "level", value: level))
        }
        guard case .success(let url) = buildURL(path: "snippets", queryItems: queryItems.isEmpty ? nil : queryItems) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func createSnippet(request: CreateSnippetRequest) -> AnyPublisher<Snippet, APIError> {
        let url = baseURL.appendingPathComponent("snippets")
        guard case .success(let body) = encodeBody(request) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode create snippet request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "POST", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func updateSnippet(id: Int, request: UpdateSnippetRequest) -> AnyPublisher<Snippet, APIError> {
        let url = baseURL.appendingPathComponent("snippets/\(id)")
        guard case .success(let body) = encodeBody(request) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode update snippet request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "PUT", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func deleteSnippet(id: Int) -> AnyPublisher<Void, APIError> {
        let url = baseURL.appendingPathComponent("snippets/\(id)")
        let urlRequest = authenticatedRequest(for: url, method: "DELETE")
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { (data, response) -> AnyPublisher<Void, APIError> in
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

    func getDailyQuestions(date: String) -> AnyPublisher<DailyQuestionsResponse, APIError> {
        let url = baseURL.appendingPathComponent("daily/questions/\(date)")
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func postDailyAnswer(date: String, questionId: Int, userAnswerIndex: Int) -> AnyPublisher<
        DailyAnswerResponse, APIError
    > {
        let url = baseURL.appendingPathComponent("daily/questions/\(date)/answer/\(questionId)")
        guard case .success(let body) = encodeJSONBody(["user_answer_index": userAnswerIndex]) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode daily answer request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "POST", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getExistingTranslationSentence(language: String, level: String, direction: String)
        -> AnyPublisher<TranslationPracticeSentenceResponse, APIError>
    {
        let queryItems = [
            URLQueryItem(name: "language", value: language),
            URLQueryItem(name: "level", value: level),
            URLQueryItem(name: "direction", value: direction),
        ]
        guard case .success(let url) = buildURL(path: "translation-practice/sentence", queryItems: queryItems) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func generateTranslationSentence(request: TranslationPracticeGenerateRequest) -> AnyPublisher<
        TranslationPracticeSentenceResponse, APIError
    > {
        let url = baseURL.appendingPathComponent("translation-practice/generate")
        guard case .success(let body) = encodeBody(request) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode translation generate request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "POST", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func submitTranslation(request: TranslationPracticeSubmitRequest) -> AnyPublisher<
        TranslationPracticeSessionResponse, APIError
    > {
        let url = baseURL.appendingPathComponent("translation-practice/submit")
        guard case .success(let body) = encodeBody(request) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode translation submit request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "POST", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getVerbConjugations(language: String) -> AnyPublisher<VerbConjugationsData, APIError> {
        let url = baseURL.appendingPathComponent("verb-conjugations/\(language)")
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getVerbConjugation(language: String, verb: String) -> AnyPublisher<
        VerbConjugationDetail, APIError
    > {
        let url = baseURL.appendingPathComponent("verb-conjugations/\(language)/\(verb)")
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func updateUser(request: UserUpdateRequest) -> AnyPublisher<User, APIError> {
        let url = baseURL.appendingPathComponent("userz/profile")
        guard case .success(let body) = encodeBody(request) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode user update request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "PUT", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { data, response -> AnyPublisher<User, APIError> in
                do {
                    let profileResponse: UserProfileMessageResponse = try self.decodeResponse(
                        data, response)
                    return Just(profileResponse.user)
                        .setFailureType(to: APIError.self)
                        .eraseToAnyPublisher()
                } catch let error as APIError {
                    return Fail(error: error).eraseToAnyPublisher()
                } catch {
                    return Fail(error: .decodingFailed(error)).eraseToAnyPublisher()
                }
            }
            .eraseToAnyPublisher()
    }

    private func decodeResponse<T: Decodable>(_ data: Data, _ response: URLResponse) throws -> T {
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        guard (200...299).contains(httpResponse.statusCode) else {
            throw APIError.invalidResponse
        }
        do {
            let decoder = JSONDecoder()
            return try decoder.decode(T.self, from: data)
        } catch {
            throw APIError.decodingFailed(error)
        }
    }

    func getWordOfTheDay(date: String? = nil) -> AnyPublisher<WordOfTheDayDisplay, APIError> {
        let url: URL
        if let date = date {
            url = baseURL.appendingPathComponent("word-of-day/\(date)")
        } else {
            url = baseURL.appendingPathComponent("word-of-day")
        }
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getAIConversations() -> AnyPublisher<ConversationListResponse, APIError> {
        let url = baseURL.appendingPathComponent("ai/conversations")
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getBookmarkedMessages() -> AnyPublisher<BookmarkedMessagesResponse, APIError> {
        let url = baseURL.appendingPathComponent("ai/bookmarks")
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func toggleBookmark(conversationId: String, messageId: String) -> AnyPublisher<
        BookmarkStatusResponse, APIError
    > {
        let url = baseURL.appendingPathComponent("ai/conversations/bookmark")
        let body = ["conversation_id": conversationId, "message_id": messageId]
        guard case .success(let bodyData) = encodeJSONBody(body) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode bookmark request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "PUT", body: bodyData)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getAIConversation(id: String) -> AnyPublisher<Conversation, APIError> {
        let url = baseURL.appendingPathComponent("ai/conversations/\(id)")
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func updateAIConversationTitle(id: String, title: String) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        let url = baseURL.appendingPathComponent("ai/conversations/\(id)")
        guard case .success(let body) = encodeJSONBody(["title": title]) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode conversation title request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "PUT", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func deleteAIConversation(id: String) -> AnyPublisher<SuccessResponse, APIError> {
        let url = baseURL.appendingPathComponent("ai/conversations/\(id)")
        let urlRequest = authenticatedRequest(for: url, method: "DELETE")
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func reportQuestion(id: Int, request: ReportQuestionRequest) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        let url = baseURL.appendingPathComponent("quiz/question/\(id)/report")
        guard case .success(let body) = encodeBody(request) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode report question request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "POST", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func markQuestionKnown(id: Int, request: MarkQuestionKnownRequest) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        let url = baseURL.appendingPathComponent("quiz/question/\(id)/mark-known")
        guard case .success(let body) = encodeBody(request) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode mark question known request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "POST", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getStorySection(id: Int) -> AnyPublisher<StorySectionWithQuestions, APIError> {
        let url = baseURL.appendingPathComponent("story/section/\(id)")
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getLearningPreferences() -> AnyPublisher<UserLearningPreferences, APIError> {
        let url = baseURL.appendingPathComponent("preferences/learning")
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func updateLearningPreferences(prefs: UserLearningPreferences) -> AnyPublisher<
        UserLearningPreferences, APIError
    > {
        let url = baseURL.appendingPathComponent("preferences/learning")
        guard case .success(let body) = encodeBody(prefs) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode learning preferences request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "PUT", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getTranslationPracticeHistory(limit: Int = 10, offset: Int = 0) -> AnyPublisher<
        TranslationPracticeHistoryResponse, APIError
    > {
        let queryItems = [
            URLQueryItem(name: "limit", value: String(limit)),
            URLQueryItem(name: "offset", value: String(offset)),
        ]
        guard case .success(let url) = buildURL(path: "translation-practice/history", queryItems: queryItems) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getAIProviders() -> AnyPublisher<AIProvidersResponse, APIError> {
        let url = baseURL.appendingPathComponent("settings/ai-providers")
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getLanguages() -> AnyPublisher<[LanguageInfo], APIError> {
        let url = baseURL.appendingPathComponent("settings/languages")
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getLevels(language: String?) -> AnyPublisher<LevelsResponse, APIError> {
        let queryItems = language.map { [URLQueryItem(name: "language", value: $0)] }
        guard case .success(let url) = buildURL(path: "settings/levels", queryItems: queryItems) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func updateWordOfDayEmailPreference(enabled: Bool) -> AnyPublisher<SuccessResponse, APIError> {
        let url = baseURL.appendingPathComponent("settings/word-of-day-email")
        guard case .success(let body) = encodeJSONBody(["enabled": enabled]) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode email preference request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "PUT", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func testAIConnection(provider: String, model: String, apiKey: String?) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        let url = baseURL.appendingPathComponent("settings/test-ai")
        var body: [String: String] = ["provider": provider, "model": model]
        if let apiKey = apiKey { body["api_key"] = apiKey }
        guard case .success(let bodyData) = encodeJSONBody(body) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode AI test request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "POST", body: bodyData)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func sendTestEmail() -> AnyPublisher<SuccessResponse, APIError> {
        let url = baseURL.appendingPathComponent("settings/test-email")
        let urlRequest = authenticatedRequest(for: url, method: "POST")
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func clearStories() -> AnyPublisher<SuccessResponse, APIError> {
        let url = baseURL.appendingPathComponent("settings/clear-stories")
        let urlRequest = authenticatedRequest(for: url, method: "POST")
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func clearAIChats() -> AnyPublisher<SuccessResponse, APIError> {
        let url = baseURL.appendingPathComponent("settings/clear-ai-chats")
        let urlRequest = authenticatedRequest(for: url, method: "POST")
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func clearTranslationHistory() -> AnyPublisher<SuccessResponse, APIError> {
        let url = baseURL.appendingPathComponent("settings/clear-translation-practice-history")
        let urlRequest = authenticatedRequest(for: url, method: "POST")
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func resetAccount() -> AnyPublisher<SuccessResponse, APIError> {
        let url = baseURL.appendingPathComponent("settings/reset-account")
        let urlRequest = authenticatedRequest(for: url, method: "POST")
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func initializeTTSStream(request: TTSRequest) -> AnyPublisher<TTSStreamInitResponse, APIError> {
        let url = baseURL.appendingPathComponent("audio")
            .appendingPathComponent("speech")
            .appendingPathComponent("init")
        guard case .success(let body) = encodeBody(request) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode TTS request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "POST", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func streamURL(for streamId: String, token: String?) -> URL {
        let streamPath = baseURL.appendingPathComponent("audio")
            .appendingPathComponent("speech")
            .appendingPathComponent("stream")
            .appendingPathComponent(streamId)

        guard var components = URLComponents(url: streamPath, resolvingAgainstBaseURL: false) else {
            return streamPath
        }
        if let token = token {
            components.queryItems = [URLQueryItem(name: "token", value: token)]
        }
        return components.url ?? streamPath
    }

    func getVoices(language: String) -> AnyPublisher<[EdgeTTSVoiceInfo], APIError> {
        let queryItems = [URLQueryItem(name: "language", value: language)]
        guard case .success(let url) = buildURL(path: "voices", queryItems: queryItems) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { data, response -> AnyPublisher<[EdgeTTSVoiceInfo], APIError> in
                guard let httpResponse = response as? HTTPURLResponse else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }

                if (200...299).contains(httpResponse.statusCode) {
                    let decoder = JSONDecoder()

                    // Try direct array of objects
                    if let voices = try? decoder.decode([EdgeTTSVoiceInfo].self, from: data) {
                        return Just(voices).setFailureType(to: APIError.self).eraseToAnyPublisher()
                    }

                    // Try decoding as a dictionary with "voices" key
                    if let wrapper = try? decoder.decode(
                        [String: [EdgeTTSVoiceInfo]].self, from: data),
                        let voices = wrapper["voices"]
                    {
                        return Just(voices).setFailureType(to: APIError.self).eraseToAnyPublisher()
                    }

                    // Try array of strings
                    if let strings = try? decoder.decode([String].self, from: data) {
                        let voices = strings.map { EdgeTTSVoiceInfo(shortName: $0) }
                        return Just(voices).setFailureType(to: APIError.self).eraseToAnyPublisher()
                    }

                    // Fallback: try to see if it's a JSON object at all
                    if let json = try? JSONSerialization.jsonObject(with: data, options: []),
                        let voicesArray = json as? [String]
                    {
                        let voices = voicesArray.map { EdgeTTSVoiceInfo(shortName: $0) }
                        return Just(voices).setFailureType(to: APIError.self).eraseToAnyPublisher()
                    }

                    return Fail(
                        error: .decodingFailed(
                            NSError(
                                domain: "", code: 0,
                                userInfo: [
                                    NSLocalizedDescriptionKey: "Failed to decode voices response"
                                ]))
                    ).eraseToAnyPublisher()
                }
                return Fail(error: .invalidResponse).eraseToAnyPublisher()
            }
            .eraseToAnyPublisher()
    }

    func translateText(request: TranslateRequest) -> AnyPublisher<TranslateResponse, APIError> {
        let url = baseURL.appendingPathComponent("translate")
        guard case .success(let body) = encodeBody(request) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode translate request"]))).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url, method: "POST", body: body)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }
}

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
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode request"])))
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
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode request"])))
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
            .flatMap { data, response -> AnyPublisher<Void, APIError> in
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

    func postJSON<T: Decodable>(
        path: String,
        body: [String: Any],
        responseType: T.Type
    ) -> AnyPublisher<T, APIError> {
        guard case .success(let url) = buildURL(path: path) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        guard case .success(let bodyData) = encodeJSONBody(body) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode request"])))
                .eraseToAnyPublisher()
        }
        let request = authenticatedRequest(for: url, method: "POST", body: bodyData)
        return URLSession.shared.dataTaskPublisher(for: request)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
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
}
