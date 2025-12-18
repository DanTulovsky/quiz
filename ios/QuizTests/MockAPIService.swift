import Foundation
import Combine
@testable import LingoLearn

class MockAPIService: APIService {
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
        return loginResult!.publisher.eraseToAnyPublisher()
    }
    
    override func signup(request: UserCreateRequest) -> AnyPublisher<SuccessResponse, APIError> {
        return signupResult!.publisher.eraseToAnyPublisher()
    }
    
    override func authStatus() -> AnyPublisher<AuthStatusResponse, APIError> {
        return authStatusResult!.publisher.eraseToAnyPublisher()
    }
    
    override func getQuestion(language: Language?, level: Level?, type: String?, excludeType: String?) -> AnyPublisher<QuestionFetchResult, APIError> {
        return getQuestionResult!.publisher.eraseToAnyPublisher()
    }
    
    override func postAnswer(request: AnswerRequest) -> AnyPublisher<AnswerResponse, APIError> {
        return postAnswerResult!.publisher.eraseToAnyPublisher()
    }
    
    override func getStories() -> AnyPublisher<[StorySummary], APIError> {
        return getStoriesResult!.publisher.eraseToAnyPublisher()
    }
    
    override func getStory(id: Int) -> AnyPublisher<StoryContent, APIError> {
        return getStoryResult!.publisher.eraseToAnyPublisher()
    }
    
    override func getSnippets(sourceLang: Language?, targetLang: Language?) -> AnyPublisher<SnippetList, APIError> {
        return getSnippetsResult!.publisher.eraseToAnyPublisher()
    }
    
    override func getDailyQuestions(date: String) -> AnyPublisher<DailyQuestionsResponse, APIError> {
        return getDailyQuestionsResult!.publisher.eraseToAnyPublisher()
    }
    
    override func postDailyAnswer(date: String, questionId: Int, userAnswerIndex: Int) -> AnyPublisher<DailyAnswerResponse, APIError> {
        return postDailyAnswerResult!.publisher.eraseToAnyPublisher()
    }
    
    override func generateTranslationSentence(request: TranslationPracticeGenerateRequest) -> AnyPublisher<TranslationPracticeSentenceResponse, APIError> {
        return generateTranslationSentenceResult!.publisher.eraseToAnyPublisher()
    }
    
    override func submitTranslation(request: TranslationPracticeSubmitRequest) -> AnyPublisher<TranslationPracticeSessionResponse, APIError> {
        return submitTranslationResult!.publisher.eraseToAnyPublisher()
    }
    
    override func getExistingTranslationSentence(language: String, level: String, direction: String) -> AnyPublisher<TranslationPracticeSentenceResponse, APIError> {
        return getExistingTranslationSentenceResult!.publisher.eraseToAnyPublisher()
    }
    
    override func getVerbConjugations(language: String) -> AnyPublisher<VerbConjugationsData, APIError> {
        return getVerbConjugationsResult!.publisher.eraseToAnyPublisher()
    }
    
    override func getVerbConjugation(language: String, verb: String) -> AnyPublisher<VerbConjugationDetail, APIError> {
        return getVerbConjugationResult!.publisher.eraseToAnyPublisher()
    }
    
    override func updateUser(request: UserUpdateRequest) -> AnyPublisher<User, APIError> {
        return updateUserResult!.publisher.eraseToAnyPublisher()
    }
    
    override func getWordOfTheDay() -> AnyPublisher<WordOfTheDayDisplay, APIError> {
        return getWordOfTheDayResult!.publisher.eraseToAnyPublisher()
    }
    
    override func getAIConversations() -> AnyPublisher<ConversationListResponse, APIError> {
        return getAIConversationsResult!.publisher.eraseToAnyPublisher()
    }
    
    override func getBookmarkedMessages() -> AnyPublisher<BookmarkedMessagesResponse, APIError> {
        return getBookmarkedMessagesResult!.publisher.eraseToAnyPublisher()
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
