import Combine
import Foundation

extension APIService {

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
}

