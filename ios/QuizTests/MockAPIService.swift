import Combine
import Foundation

@testable import Quiz

final class MockAPIService: APIService, @unchecked Sendable {
    // Result properties
    var loginResult: Result<LoginResponse, APIError>?
    var signupResult: Result<SuccessResponse, APIError>?
    var authStatusResult: Result<AuthStatusResponse, APIError>?
    var getQuestionResult: Result<QuestionFetchResult, APIError>?
    var postAnswerResult: Result<AnswerResponse, APIError>?
    var getStoriesResult: Result<[StorySummary], APIError>?
    var getStoryResult: Result<StoryContent, APIError>?
    var getSnippetsResult: Result<SnippetList, APIError>?
    var getDailyQuestionsResult: Result<DailyQuestionsResponse, APIError>?
    var postDailyAnswerResult: Result<DailyAnswerResponse, APIError>?
    var generateTranslationSentenceResult: Result<TranslationPracticeSentenceResponse, APIError>?
    var submitTranslationResult: Result<TranslationPracticeSessionResponse, APIError>?
    var getExistingTranslationSentenceResult: Result<TranslationPracticeSentenceResponse, APIError>?
    var getVerbConjugationsResult: Result<VerbConjugationsData, APIError>?
    var getVerbConjugationResult: Result<VerbConjugationDetail, APIError>?
    var updateUserResult: Result<User, APIError>?
    var getWordOfTheDayResult: Result<WordOfTheDayDisplay, APIError>?
    var getAIConversationsResult: Result<ConversationListResponse, APIError>?
    var getBookmarkedMessagesResult: Result<BookmarkedMessagesResponse, APIError>?

    // Method Overrides
    override func login(request: LoginRequest) -> AnyPublisher<LoginResponse, APIError> {
        guard let result = loginResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func signup(request: UserCreateRequest) -> AnyPublisher<SuccessResponse, APIError> {
        guard let result = signupResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func authStatus() -> AnyPublisher<AuthStatusResponse, APIError> {
        guard let result = authStatusResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func getQuestion(
        language: Language?, level: Level?, type: String?, excludeType: String?
    ) -> AnyPublisher<QuestionFetchResult, APIError> {
        guard let result = getQuestionResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func postAnswer(request: AnswerRequest) -> AnyPublisher<AnswerResponse, APIError> {
        guard let result = postAnswerResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func getStories() -> AnyPublisher<[StorySummary], APIError> {
        guard let result = getStoriesResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func getStory(id: Int) -> AnyPublisher<StoryContent, APIError> {
        guard let result = getStoryResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func getSnippets(sourceLang: Language?, targetLang: Language?, storyId: Int? = nil)
        -> AnyPublisher<
            SnippetList, APIError
        >
    {
        guard let result = getSnippetsResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func getDailyQuestions(date: String) -> AnyPublisher<DailyQuestionsResponse, APIError>
    {
        guard let result = getDailyQuestionsResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func postDailyAnswer(date: String, questionId: Int, userAnswerIndex: Int)
        -> AnyPublisher<DailyAnswerResponse, APIError>
    {
        guard let result = postDailyAnswerResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func generateTranslationSentence(request: TranslationPracticeGenerateRequest)
        -> AnyPublisher<TranslationPracticeSentenceResponse, APIError>
    {
        guard let result = generateTranslationSentenceResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func submitTranslation(request: TranslationPracticeSubmitRequest) -> AnyPublisher<
        TranslationPracticeSessionResponse, APIError
    > {
        guard let result = submitTranslationResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func getExistingTranslationSentence(language: String, level: String, direction: String)
        -> AnyPublisher<TranslationPracticeSentenceResponse, APIError>
    {
        guard let result = getExistingTranslationSentenceResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func getVerbConjugations(language: String) -> AnyPublisher<
        VerbConjugationsData, APIError
    > {
        guard let result = getVerbConjugationsResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func getVerbConjugation(language: String, verb: String) -> AnyPublisher<
        VerbConjugationDetail, APIError
    > {
        if let result = getVerbConjugationResult {
            return result.publisher.eraseToAnyPublisher()
        }
        return Fail(error: .invalidResponse).eraseToAnyPublisher()
    }

    override func updateUser(request: UserUpdateRequest) -> AnyPublisher<User, APIError> {
        guard let result = updateUserResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func getWordOfTheDay(date: String? = nil) -> AnyPublisher<
        WordOfTheDayDisplay, APIError
    > {
        guard let result = getWordOfTheDayResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func getAIConversations() -> AnyPublisher<ConversationListResponse, APIError> {
        guard let result = getAIConversationsResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    override func getBookmarkedMessages() -> AnyPublisher<BookmarkedMessagesResponse, APIError> {
        guard let result = getBookmarkedMessagesResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }
}

extension Result {
    var publisher: AnyPublisher<Success, Failure> {
        switch self {
        case .success(let value):
            return Just(value)
                .setFailureType(to: Failure.self)
                .eraseToAnyPublisher()
        case .failure(let error):
            return Fail(error: error)
                .eraseToAnyPublisher()
        }
    }
}
