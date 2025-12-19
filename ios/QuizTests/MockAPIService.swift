import Combine
import Foundation

@testable import Quiz

final class MockAPIService: APIService, @unchecked Sendable {
    private let lock = NSLock()

    // Result properties - marked nonisolated(unsafe) because access is protected by lock
    nonisolated(unsafe) private var _loginResult: Result<LoginResponse, APIError>?
    nonisolated(unsafe) private var _signupResult: Result<SuccessResponse, APIError>?
    nonisolated(unsafe) private var _authStatusResult: Result<AuthStatusResponse, APIError>?
    nonisolated(unsafe) private var _getQuestionResult: Result<QuestionFetchResult, APIError>?
    nonisolated(unsafe) private var _postAnswerResult: Result<AnswerResponse, APIError>?
    nonisolated(unsafe) private var _getStoriesResult: Result<[StorySummary], APIError>?
    nonisolated(unsafe) private var _getStoryResult: Result<StoryContent, APIError>?
    nonisolated(unsafe) private var _getSnippetsResult: Result<SnippetList, APIError>?
    nonisolated(unsafe) private var _getDailyQuestionsResult:
        Result<DailyQuestionsResponse, APIError>?
    nonisolated(unsafe) private var _postDailyAnswerResult: Result<DailyAnswerResponse, APIError>?
    nonisolated(unsafe) private var _generateTranslationSentenceResult:
        Result<TranslationPracticeSentenceResponse, APIError>?
    nonisolated(unsafe) private var _submitTranslationResult:
        Result<TranslationPracticeSessionResponse, APIError>?
    nonisolated(unsafe) private var _getExistingTranslationSentenceResult:
        Result<TranslationPracticeSentenceResponse, APIError>?
    nonisolated(unsafe) private var _getVerbConjugationsResult:
        Result<VerbConjugationsData, APIError>?
    nonisolated(unsafe) private var _getVerbConjugationResult:
        Result<VerbConjugationDetail, APIError>?
    nonisolated(unsafe) private var _updateUserResult: Result<User, APIError>?
    nonisolated(unsafe) private var _getWordOfTheDayResult: Result<WordOfTheDayDisplay, APIError>?
    nonisolated(unsafe) private var _getAIConversationsResult:
        Result<ConversationListResponse, APIError>?
    nonisolated(unsafe) private var _getBookmarkedMessagesResult:
        Result<BookmarkedMessagesResponse, APIError>?

    // Thread-safe property accessors
    var loginResult: Result<LoginResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _loginResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _loginResult = newValue
        }
    }
    var signupResult: Result<SuccessResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _signupResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _signupResult = newValue
        }
    }
    var authStatusResult: Result<AuthStatusResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _authStatusResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _authStatusResult = newValue
        }
    }
    var getQuestionResult: Result<QuestionFetchResult, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getQuestionResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getQuestionResult = newValue
        }
    }
    var postAnswerResult: Result<AnswerResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _postAnswerResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _postAnswerResult = newValue
        }
    }
    var getStoriesResult: Result<[StorySummary], APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getStoriesResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getStoriesResult = newValue
        }
    }
    var getStoryResult: Result<StoryContent, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getStoryResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getStoryResult = newValue
        }
    }
    var getSnippetsResult: Result<SnippetList, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getSnippetsResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getSnippetsResult = newValue
        }
    }
    var getDailyQuestionsResult: Result<DailyQuestionsResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getDailyQuestionsResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getDailyQuestionsResult = newValue
        }
    }
    var postDailyAnswerResult: Result<DailyAnswerResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _postDailyAnswerResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _postDailyAnswerResult = newValue
        }
    }
    var generateTranslationSentenceResult: Result<TranslationPracticeSentenceResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _generateTranslationSentenceResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _generateTranslationSentenceResult = newValue
        }
    }
    var submitTranslationResult: Result<TranslationPracticeSessionResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _submitTranslationResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _submitTranslationResult = newValue
        }
    }
    var getExistingTranslationSentenceResult: Result<TranslationPracticeSentenceResponse, APIError>?
    {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getExistingTranslationSentenceResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getExistingTranslationSentenceResult = newValue
        }
    }
    var getVerbConjugationsResult: Result<VerbConjugationsData, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getVerbConjugationsResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getVerbConjugationsResult = newValue
        }
    }
    var getVerbConjugationResult: Result<VerbConjugationDetail, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getVerbConjugationResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getVerbConjugationResult = newValue
        }
    }
    var updateUserResult: Result<User, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _updateUserResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _updateUserResult = newValue
        }
    }
    var getWordOfTheDayResult: Result<WordOfTheDayDisplay, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getWordOfTheDayResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getWordOfTheDayResult = newValue
        }
    }
    var getAIConversationsResult: Result<ConversationListResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getAIConversationsResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getAIConversationsResult = newValue
        }
    }
    var getBookmarkedMessagesResult: Result<BookmarkedMessagesResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getBookmarkedMessagesResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getBookmarkedMessagesResult = newValue
        }
    }

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

    override func getSnippets(sourceLang: String?, targetLang: String?, storyId: Int? = nil)
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
