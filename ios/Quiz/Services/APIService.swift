import Combine
import Foundation

class APIService {
    static let shared = APIService()
    private let baseURL: URL = {
        #if targetEnvironment(simulator)
            return URL(string: "http://localhost:3000/v1")!
        #else
            return URL(string: "https://quiz.wetsnow.com/v1")!
        #endif
    }()

    enum APIError: Error, LocalizedError {
        case invalidURL
        case requestFailed(Error)
        case invalidResponse
        case decodingFailed(Error)
        case backendError(code: String?, message: String, details: String?)

        var errorDescription: String? {
            switch self {
            case .invalidURL: return "Invalid URL"
            case .requestFailed(let error): return error.localizedDescription
            case .invalidResponse: return "Invalid response from server"
            case .decodingFailed(let error):
                return "Failed to decode response: \(error.localizedDescription)"
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

    init() {}

    private var decoder: JSONDecoder {
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
    }

    private func authenticatedRequest(for url: URL, method: String = "GET") -> URLRequest {
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = method
        urlRequest.httpShouldHandleCookies = true
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
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
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = "POST"
        urlRequest.httpShouldHandleCookies = true
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        urlRequest.httpBody = try? JSONEncoder().encode(request)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func initiateGoogleLogin() -> AnyPublisher<GoogleOAuthLoginResponse, APIError> {
        var components = URLComponents(
            url: baseURL.appendingPathComponent("auth/google/login"),
            resolvingAgainstBaseURL: false)!
        // Add platform parameter to help backend detect iOS and use iOS client ID
        components.queryItems = [URLQueryItem(name: "platform", value: "ios")]

        let urlRequest = URLRequest(url: components.url!)
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

        var components = URLComponents(
            url: baseURL.appendingPathComponent("auth/google/callback"),
            resolvingAgainstBaseURL: false)!
        var queryItems = [URLQueryItem(name: "code", value: code)]
        if let state = state {
            queryItems.append(URLQueryItem(name: "state", value: state))
        }
        components.queryItems = queryItems

        var urlRequest = URLRequest(url: components.url!)
        urlRequest.httpShouldHandleCookies = true
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
                        let cookies = HTTPCookie.cookies(
                            withResponseHeaderFields: httpResponse.allHeaderFields
                                as! [String: String], for: url)
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
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = "POST"
        urlRequest.httpShouldHandleCookies = true
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        urlRequest.httpBody = try? JSONEncoder().encode(request)
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
        var urlComponents = URLComponents(
            url: baseURL.appendingPathComponent("quiz/question"), resolvingAgainstBaseURL: false)!
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
        urlComponents.queryItems = queryItems
        let urlRequest = authenticatedRequest(for: urlComponents.url!)
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
        var urlRequest = authenticatedRequest(for: url, method: "POST")
        urlRequest.httpBody = try? JSONEncoder().encode(request)
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

    func getSnippets(sourceLang: String?, targetLang: String?, storyId: Int? = nil)
        -> AnyPublisher<SnippetList, APIError>
    {
        var urlComponents = URLComponents(
            url: baseURL.appendingPathComponent("snippets"), resolvingAgainstBaseURL: false)!
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
        urlComponents.queryItems = queryItems
        let urlRequest = authenticatedRequest(for: urlComponents.url!)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func createSnippet(request: CreateSnippetRequest) -> AnyPublisher<Snippet, APIError> {
        let url = baseURL.appendingPathComponent("snippets")
        var urlRequest = authenticatedRequest(for: url, method: "POST")
        urlRequest.httpBody = try? JSONEncoder().encode(request)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func updateSnippet(id: Int, request: UpdateSnippetRequest) -> AnyPublisher<Snippet, APIError> {
        let url = baseURL.appendingPathComponent("snippets/\(id)")
        var urlRequest = authenticatedRequest(for: url, method: "PUT")
        urlRequest.httpBody = try? JSONEncoder().encode(request)
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
        var urlRequest = authenticatedRequest(for: url, method: "POST")
        urlRequest.httpBody = try? JSONSerialization.data(withJSONObject: [
            "user_answer_index": userAnswerIndex
        ])
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getExistingTranslationSentence(language: String, level: String, direction: String)
        -> AnyPublisher<TranslationPracticeSentenceResponse, APIError>
    {
        var urlComponents = URLComponents(
            url: baseURL.appendingPathComponent("translation-practice/sentence"),
            resolvingAgainstBaseURL: false)!
        urlComponents.queryItems = [
            URLQueryItem(name: "language", value: language),
            URLQueryItem(name: "level", value: level),
            URLQueryItem(name: "direction", value: direction),
        ]
        let urlRequest = authenticatedRequest(for: urlComponents.url!)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func generateTranslationSentence(request: TranslationPracticeGenerateRequest) -> AnyPublisher<
        TranslationPracticeSentenceResponse, APIError
    > {
        let url = baseURL.appendingPathComponent("translation-practice/generate")
        var urlRequest = authenticatedRequest(for: url, method: "POST")
        urlRequest.httpBody = try? JSONEncoder().encode(request)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func submitTranslation(request: TranslationPracticeSubmitRequest) -> AnyPublisher<
        TranslationPracticeSessionResponse, APIError
    > {
        let url = baseURL.appendingPathComponent("translation-practice/submit")
        var urlRequest = authenticatedRequest(for: url, method: "POST")
        urlRequest.httpBody = try? JSONEncoder().encode(request)
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
        var urlRequest = authenticatedRequest(for: url, method: "PUT")
        urlRequest.httpBody = try? JSONEncoder().encode(request)
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
        var urlRequest = authenticatedRequest(for: url, method: "PUT")
        let body = ["conversation_id": conversationId, "message_id": messageId]
        urlRequest.httpBody = try? JSONSerialization.data(withJSONObject: body)
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
        var urlRequest = authenticatedRequest(for: url, method: "PUT")
        urlRequest.httpBody = try? JSONSerialization.data(withJSONObject: ["title": title])
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
        var urlRequest = authenticatedRequest(for: url, method: "POST")
        urlRequest.httpBody = try? JSONEncoder().encode(request)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func markQuestionKnown(id: Int, request: MarkQuestionKnownRequest) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        let url = baseURL.appendingPathComponent("quiz/question/\(id)/mark-known")
        var urlRequest = authenticatedRequest(for: url, method: "POST")
        urlRequest.httpBody = try? JSONEncoder().encode(request)
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
        var urlRequest = authenticatedRequest(for: url, method: "PUT")
        urlRequest.httpBody = try? JSONEncoder().encode(prefs)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func getTranslationPracticeHistory(limit: Int = 10, offset: Int = 0) -> AnyPublisher<
        TranslationPracticeHistoryResponse, APIError
    > {
        var urlComponents = URLComponents(
            url: baseURL.appendingPathComponent("translation-practice/history"),
            resolvingAgainstBaseURL: false)!
        urlComponents.queryItems = [
            URLQueryItem(name: "limit", value: String(limit)),
            URLQueryItem(name: "offset", value: String(offset)),
        ]
        let urlRequest = authenticatedRequest(for: urlComponents.url!)
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
        var urlComponents = URLComponents(
            url: baseURL.appendingPathComponent("settings/levels"), resolvingAgainstBaseURL: false)!
        if let language = language {
            urlComponents.queryItems = [URLQueryItem(name: "language", value: language)]
        }
        let urlRequest = authenticatedRequest(for: urlComponents.url!)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func updateWordOfDayEmailPreference(enabled: Bool) -> AnyPublisher<SuccessResponse, APIError> {
        let url = baseURL.appendingPathComponent("settings/word-of-day-email")
        var urlRequest = authenticatedRequest(for: url, method: "PUT")
        urlRequest.httpBody = try? JSONEncoder().encode(["enabled": enabled])
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    func testAIConnection(provider: String, model: String, apiKey: String?) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        let url = baseURL.appendingPathComponent("settings/test-ai")
        var urlRequest = authenticatedRequest(for: url, method: "POST")
        var body: [String: String] = ["provider": provider, "model": model]
        if let apiKey = apiKey { body["api_key"] = apiKey }
        urlRequest.httpBody = try? JSONEncoder().encode(body)
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
        var urlRequest = authenticatedRequest(for: url, method: "POST")
        if let body = try? JSONEncoder().encode(request) {
            urlRequest.httpBody = body
        }
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

        var components = URLComponents(url: streamPath, resolvingAgainstBaseURL: false)!
        if let token = token {
            components.queryItems = [URLQueryItem(name: "token", value: token)]
        }
        return components.url!
    }

    func getVoices(language: String) -> AnyPublisher<[EdgeTTSVoiceInfo], APIError> {
        let url = baseURL.appendingPathComponent("voices")
        var components = URLComponents(url: url, resolvingAgainstBaseURL: false)!
        components.queryItems = [URLQueryItem(name: "language", value: language)]

        let urlRequest = authenticatedRequest(for: components.url!)
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
        var urlRequest = authenticatedRequest(for: url, method: "POST")
        urlRequest.httpBody = try? JSONEncoder().encode(request)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }
}
