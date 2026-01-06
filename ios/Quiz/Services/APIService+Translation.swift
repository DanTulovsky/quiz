import Combine
import Foundation

extension APIService {
    func getExistingTranslationSentence(language: String, level: String, direction: String)
    -> AnyPublisher<TranslationPracticeSentenceResponse, APIError> {
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
}





