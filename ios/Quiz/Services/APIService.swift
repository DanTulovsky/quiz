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
        guard let value = item.value, !value.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        else {
            return false
        }
        return true
    }

    private func buildURL(path: String, queryItems: [URLQueryItem]? = nil) -> Result<URL, APIError>
    {
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

    private func encodingError(description: String) -> APIError {
        return .encodingFailed(
            NSError(
                domain: "APIService",
                code: -1,
                userInfo: [NSLocalizedDescriptionKey: description]
            )
        )
    }

    private func authenticatedRequest(for url: URL, method: String = "GET", body: Data? = nil)
        -> URLRequest
    {
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
                return Fail(
                    error: .backendError(
                        code: errorResp.code, message: msg, details: errorResp.details)
                ).eraseToAnyPublisher()
            }
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
    }

    func login(request: LoginRequest) -> AnyPublisher<LoginResponse, APIError> {
        return post(
            path: "auth/login", body: request, responseType: LoginResponse.self, retry: true)
    }

    func initiateGoogleLogin() -> AnyPublisher<GoogleOAuthLoginResponse, APIError> {
        var params = QueryParameters()
        params.add("platform", value: "ios")
        return get(
            path: "auth/google/login", queryItems: params.build(),
            responseType: GoogleOAuthLoginResponse.self)
    }

    func handleGoogleCallback(code: String, state: String?) -> AnyPublisher<LoginResponse, APIError>
    {
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

        // Use URLSession.shared to ensure cookies are shared with other API calls
        // This ensures the session cookie from OAuth is available for subsequent requests
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { data, response -> AnyPublisher<LoginResponse, APIError> in
                if let httpResponse = response as? HTTPURLResponse {
                    self.storeCookies(from: httpResponse)
                }
                return self.handleResponse(data, response)
            }
            .eraseToAnyPublisher()
    }

    func authStatus() -> AnyPublisher<AuthStatusResponse, APIError> {
        return get(path: "auth/status", responseType: AuthStatusResponse.self)
            .retryOnTransientFailure(maxRetries: 2)
            .eraseToAnyPublisher()
    }

    func logout() -> AnyPublisher<SuccessResponse, APIError> {
        return postVoid(path: "auth/logout")
            .handleEvents(receiveOutput: { _ in
                self.clearSessionCookie()
            })
            .eraseToAnyPublisher()
    }

    func signup(request: UserCreateRequest) -> AnyPublisher<SuccessResponse, APIError> {
        return post(path: "auth/signup", body: request, responseType: SuccessResponse.self)
    }

    enum QuestionFetchResult {
        case question(Question)
        case generating(GeneratingStatusResponse)
    }

    func getQuestion(language: Language?, level: Level?, type: String?, excludeType: String?)
        -> AnyPublisher<QuestionFetchResult, APIError>
    {
        var params = QueryParameters()
        params.add("language", value: language?.rawValue)
        params.add("level", value: level?.rawValue)
        params.add("type", value: type)
        params.add("exclude_type", value: excludeType)
        guard
            case .success(let url) = buildURL(
                path: "quiz/question", queryItems: params.build())
        else {
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
                    return Just(data)
                        .decode(type: Question.self, decoder: self.decoder)
                        .map(QuestionFetchResult.question)
                        .mapError { .decodingFailed($0) }
                        .eraseToAnyPublisher()
                } else if httpResponse.statusCode == 202 {
                    return Just(data)
                        .decode(type: GeneratingStatusResponse.self, decoder: self.decoder)
                        .map(QuestionFetchResult.generating)
                        .mapError { .decodingFailed($0) }
                        .eraseToAnyPublisher()
                } else {
                    return self.handleErrorResponse(data: data)
                }
            }
            .eraseToAnyPublisher()
    }

    private func handleErrorResponse<T>(data: Data) -> AnyPublisher<T, APIError> {
        if let errorResp = try? self.decoder.decode(ErrorResponse.self, from: data),
            let msg = errorResp.message ?? errorResp.error
        {
            return Fail(
                error: .backendError(code: errorResp.code, message: msg, details: errorResp.details)
            )
            .eraseToAnyPublisher()
        }
        return Fail(error: .invalidResponse).eraseToAnyPublisher()
    }

    func postAnswer(request: AnswerRequest) -> AnyPublisher<AnswerResponse, APIError> {
        return post(path: "quiz/answer", body: request, responseType: AnswerResponse.self)
    }

    func getStories() -> AnyPublisher<[StorySummary], APIError> {
        return get(path: "story", responseType: [StorySummary].self)
    }

    func getStory(id: Int) -> AnyPublisher<StoryContent, APIError> {
        return get(path: "story/\(id)", responseType: StoryContent.self)
    }

    func getSnippets(
        sourceLang: String?, targetLang: String?, storyId: Int? = nil, query: String? = nil,
        level: String? = nil
    )
        -> AnyPublisher<SnippetList, APIError>
    {
        var params = QueryParameters()
        params.add("source_lang", value: sourceLang)
        params.add("target_lang", value: targetLang)
        params.add("story_id", value: storyId)
        params.add("q", value: query)
        params.add("level", value: level)
        return get(
            path: "snippets", queryItems: params.build(),
            responseType: SnippetList.self)
    }

    func createSnippet(request: CreateSnippetRequest) -> AnyPublisher<Snippet, APIError> {
        return post(path: "snippets", body: request, responseType: Snippet.self)
    }

    func updateSnippet(id: Int, request: UpdateSnippetRequest) -> AnyPublisher<Snippet, APIError> {
        return put(path: "snippets/\(id)", body: request, responseType: Snippet.self)
    }

    func deleteSnippet(id: Int) -> AnyPublisher<Void, APIError> {
        return deleteVoid(path: "snippets/\(id)")
    }

    func getDailyQuestions(date: String) -> AnyPublisher<DailyQuestionsResponse, APIError> {
        return get(path: "daily/questions/\(date)", responseType: DailyQuestionsResponse.self)
    }

    func postDailyAnswer(date: String, questionId: Int, userAnswerIndex: Int) -> AnyPublisher<
        DailyAnswerResponse, APIError
    > {
        return postJSON(
            path: "daily/questions/\(date)/answer/\(questionId)",
            body: ["user_answer_index": userAnswerIndex],
            responseType: DailyAnswerResponse.self
        )
    }

    func getExistingTranslationSentence(language: String, level: String, direction: String)
        -> AnyPublisher<TranslationPracticeSentenceResponse, APIError>
    {
        var params = QueryParameters()
        params.add("language", value: language)
        params.add("level", value: level)
        params.add("direction", value: direction)
        return get(
            path: "translation-practice/sentence", queryItems: params.build(),
            responseType: TranslationPracticeSentenceResponse.self)
    }

    func generateTranslationSentence(request: TranslationPracticeGenerateRequest) -> AnyPublisher<
        TranslationPracticeSentenceResponse, APIError
    > {
        return post(
            path: "translation-practice/generate", body: request,
            responseType: TranslationPracticeSentenceResponse.self)
    }

    func submitTranslation(request: TranslationPracticeSubmitRequest) -> AnyPublisher<
        TranslationPracticeSessionResponse, APIError
    > {
        return post(
            path: "translation-practice/submit", body: request,
            responseType: TranslationPracticeSessionResponse.self)
    }

    func getVerbConjugations(language: String) -> AnyPublisher<VerbConjugationsData, APIError> {
        return get(path: "verb-conjugations/\(language)", responseType: VerbConjugationsData.self)
    }

    func getVerbConjugation(language: String, verb: String) -> AnyPublisher<
        VerbConjugationDetail, APIError
    > {
        return get(
            path: "verb-conjugations/\(language)/\(verb)", responseType: VerbConjugationDetail.self)
    }

    func updateUser(request: UserUpdateRequest) -> AnyPublisher<User, APIError> {
        return put(
            path: "userz/profile", body: request, responseType: UserProfileMessageResponse.self
        )
        .map { $0.user }
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
        if let date = date {
            return get(path: "word-of-day/\(date)", responseType: WordOfTheDayDisplay.self)
        } else {
            return get(path: "word-of-day", responseType: WordOfTheDayDisplay.self)
        }
    }

    func getAIConversations() -> AnyPublisher<ConversationListResponse, APIError> {
        return get(path: "ai/conversations", responseType: ConversationListResponse.self)
    }

    func getBookmarkedMessages() -> AnyPublisher<BookmarkedMessagesResponse, APIError> {
        return get(path: "ai/bookmarks", responseType: BookmarkedMessagesResponse.self)
    }

    func toggleBookmark(conversationId: String, messageId: String) -> AnyPublisher<
        BookmarkStatusResponse, APIError
    > {
        return putJSON(
            path: "ai/conversations/bookmark",
            body: ["conversation_id": conversationId, "message_id": messageId],
            responseType: BookmarkStatusResponse.self
        )
    }

    func getAIConversation(id: String) -> AnyPublisher<Conversation, APIError> {
        return get(path: "ai/conversations/\(id)", responseType: Conversation.self)
    }

    func updateAIConversationTitle(id: String, title: String) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        return putJSON(
            path: "ai/conversations/\(id)",
            body: ["title": title],
            responseType: SuccessResponse.self
        )
    }

    func deleteAIConversation(id: String) -> AnyPublisher<SuccessResponse, APIError> {
        return delete(path: "ai/conversations/\(id)", responseType: SuccessResponse.self)
    }

    func reportQuestion(id: Int, request: ReportQuestionRequest) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        return post(
            path: "quiz/question/\(id)/report", body: request, responseType: SuccessResponse.self)
    }

    func markQuestionKnown(id: Int, request: MarkQuestionKnownRequest) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        return post(
            path: "quiz/question/\(id)/mark-known", body: request,
            responseType: SuccessResponse.self)
    }

    func getStorySection(id: Int) -> AnyPublisher<StorySectionWithQuestions, APIError> {
        return get(path: "story/section/\(id)", responseType: StorySectionWithQuestions.self)
    }

    func getLearningPreferences() -> AnyPublisher<UserLearningPreferences, APIError> {
        return get(path: "preferences/learning", responseType: UserLearningPreferences.self)
    }

    func updateLearningPreferences(prefs: UserLearningPreferences) -> AnyPublisher<
        UserLearningPreferences, APIError
    > {
        return put(
            path: "preferences/learning", body: prefs, responseType: UserLearningPreferences.self)
    }

    func getTranslationPracticeHistory(limit: Int = 10, offset: Int = 0) -> AnyPublisher<
        TranslationPracticeHistoryResponse, APIError
    > {
        var params = QueryParameters()
        params.add("limit", value: limit)
        params.add("offset", value: offset)
        return get(
            path: "translation-practice/history", queryItems: params.build(),
            responseType: TranslationPracticeHistoryResponse.self)
    }

    func getAIProviders() -> AnyPublisher<AIProvidersResponse, APIError> {
        return get(path: "settings/ai-providers", responseType: AIProvidersResponse.self)
    }

    func getLanguages() -> AnyPublisher<[LanguageInfo], APIError> {
        return get(path: "settings/languages", responseType: [LanguageInfo].self)
    }

    func getLevels(language: String?) -> AnyPublisher<LevelsResponse, APIError> {
        var params = QueryParameters()
        params.add("language", value: language)
        return get(
            path: "settings/levels", queryItems: params.build(), responseType: LevelsResponse.self)
    }

    func updateWordOfDayEmailPreference(enabled: Bool) -> AnyPublisher<SuccessResponse, APIError> {
        return putJSON(
            path: "settings/word-of-day-email",
            body: ["enabled": enabled],
            responseType: SuccessResponse.self
        )
    }

    func testAIConnection(provider: String, model: String, apiKey: String?) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        var body: [String: Any] = ["provider": provider, "model": model]
        if let apiKey = apiKey { body["api_key"] = apiKey }
        return postJSON(
            path: "settings/test-ai",
            body: body,
            responseType: SuccessResponse.self
        )
    }

    func sendTestEmail() -> AnyPublisher<SuccessResponse, APIError> {
        return postVoid(path: "settings/test-email")
    }

    func clearStories() -> AnyPublisher<SuccessResponse, APIError> {
        return postVoid(path: "settings/clear-stories")
    }

    func clearAIChats() -> AnyPublisher<SuccessResponse, APIError> {
        return postVoid(path: "settings/clear-ai-chats")
    }

    func clearTranslationHistory() -> AnyPublisher<SuccessResponse, APIError> {
        return postVoid(path: "settings/clear-translation-practice-history")
    }

    func resetAccount() -> AnyPublisher<SuccessResponse, APIError> {
        return postVoid(path: "settings/reset-account")
    }

    func initializeTTSStream(request: TTSRequest) -> AnyPublisher<TTSStreamInitResponse, APIError> {
        return post(
            path: "audio/speech/init",
            body: request,
            responseType: TTSStreamInitResponse.self
        )
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
        var params = QueryParameters()
        params.add("language", value: language)
        guard case .success(let url) = buildURL(path: "voices", queryItems: params.build()) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { data, response -> AnyPublisher<[EdgeTTSVoiceInfo], APIError> in
                guard let httpResponse = response as? HTTPURLResponse else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }
                guard (200...299).contains(httpResponse.statusCode) else {
                    return self.handleErrorResponse(data: data)
                }
                return self.decodeVoicesWithFallbacks(data: data)
            }
            .eraseToAnyPublisher()
    }

    private func decodeVoicesWithFallbacks(data: Data) -> AnyPublisher<[EdgeTTSVoiceInfo], APIError>
    {
        let decoder = JSONDecoder()

        // Try direct array of objects
        if let voices = try? decoder.decode([EdgeTTSVoiceInfo].self, from: data) {
            return Just(voices).setFailureType(to: APIError.self).eraseToAnyPublisher()
        }

        // Try decoding as a dictionary with "voices" key
        if let wrapper = try? decoder.decode([String: [EdgeTTSVoiceInfo]].self, from: data),
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
                    userInfo: [NSLocalizedDescriptionKey: "Failed to decode voices response"]
                )
            )
        ).eraseToAnyPublisher()
    }

    func translateText(request: TranslateRequest) -> AnyPublisher<TranslateResponse, APIError> {
        return post(path: "translate", body: request, responseType: TranslateResponse.self)
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
