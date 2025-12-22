import Combine
import Foundation

extension APIService {
    func getAIProviders() -> AnyPublisher<AIProvidersResponse, APIError> {
        return get(path: "settings/ai-providers", responseType: AIProvidersResponse.self)
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
}

