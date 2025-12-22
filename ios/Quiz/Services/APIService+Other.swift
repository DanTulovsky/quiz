import Combine
import Foundation

extension APIService {
    func getVerbConjugations(language: String) -> AnyPublisher<VerbConjugationsData, APIError> {
        return get(path: "verb-conjugations/\(language)", responseType: VerbConjugationsData.self)
    }

    func getVerbConjugation(language: String, verb: String) -> AnyPublisher<
        VerbConjugationDetail, APIError
    > {
        return get(
            path: "verb-conjugations/\(language)/\(verb)", responseType: VerbConjugationDetail.self)
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

    func getLanguages() -> AnyPublisher<[LanguageInfo], APIError> {
        return get(path: "settings/languages", responseType: [LanguageInfo].self)
    }
}
