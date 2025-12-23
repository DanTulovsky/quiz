import Combine
import Foundation

@testable import Quiz

// swiftlint:disable file_length
// swiftlint:disable:next type_body_length
final class MockAPIService: APIServiceProtocol, @unchecked Sendable {
    private let lock = NSLock()
    let baseURL: URL

    typealias APIError = APIService.APIError
    typealias QuestionFetchResult = APIService.QuestionFetchResult

    init() {
        guard let url = URL(string: "http://localhost:3000/v1") else {
            fatalError("Invalid base URL for mock")
        }
        self.baseURL = url
    }

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
    nonisolated(unsafe) private var _handleGoogleCallbackResult: Result<LoginResponse, APIError>?
    nonisolated(unsafe) private var _getLanguagesResult: Result<[LanguageInfo], APIError>?
    nonisolated(unsafe) private var _getLevelsResult: Result<LevelsResponse, APIError>?
    nonisolated(unsafe) private var _reportQuestionResult: Result<SuccessResponse, APIError>?
    nonisolated(unsafe) private var _markQuestionKnownResult: Result<SuccessResponse, APIError>?
    nonisolated(unsafe) private var _getAIConversationResult: Result<Conversation, APIError>?
    nonisolated(unsafe) private var _updateAIConversationTitleResult:
        Result<SuccessResponse, APIError>?
    nonisolated(unsafe) private var _deleteAIConversationResult: Result<SuccessResponse, APIError>?
    nonisolated(unsafe) private var _toggleBookmarkResult: Result<BookmarkStatusResponse, APIError>?
    nonisolated(unsafe) private var _logoutResult: Result<SuccessResponse, APIError>?
    nonisolated(unsafe) private var _getLearningPreferencesResult:
        Result<UserLearningPreferences, APIError>?
    nonisolated(unsafe) private var _updateLearningPreferencesResult:
        Result<UserLearningPreferences, APIError>?
    nonisolated(unsafe) private var _getAIProvidersResult: Result<AIProvidersResponse, APIError>?
    nonisolated(unsafe) private var _testAIConnectionResult: Result<SuccessResponse, APIError>?
    nonisolated(unsafe) private var _sendTestEmailResult: Result<SuccessResponse, APIError>?
    nonisolated(unsafe) private var _sendTestIOSNotificationResult:
        Result<SuccessResponse, APIError>?
    nonisolated(unsafe) private var _clearStoriesResult: Result<SuccessResponse, APIError>?
    nonisolated(unsafe) private var _clearAIChatsResult: Result<SuccessResponse, APIError>?
    nonisolated(unsafe) private var _clearTranslationHistoryResult:
        Result<SuccessResponse, APIError>?
    nonisolated(unsafe) private var _resetAccountResult: Result<SuccessResponse, APIError>?
    nonisolated(unsafe) private var _getVoicesResult: Result<[EdgeTTSVoiceInfo], APIError>?
    nonisolated(unsafe) private var _getStorySectionResult:
        Result<StorySectionWithQuestions, APIError>?
    nonisolated(unsafe) private var _getTranslationPracticeHistoryResult:
        Result<TranslationPracticeHistoryResponse, APIError>?
    nonisolated(unsafe) private var _createSnippetResult: Result<Snippet, APIError>?
    nonisolated(unsafe) private var _updateSnippetResult: Result<Snippet, APIError>?
    nonisolated(unsafe) private var _deleteSnippetResult: Result<Void, APIError>?
    nonisolated(unsafe) private var _googleOAuthResponse: GoogleOAuthLoginResponse?
    nonisolated(unsafe) private var _dailyAnswerResponse: DailyAnswerResponse?

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
    var handleGoogleCallbackResult: Result<LoginResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _handleGoogleCallbackResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _handleGoogleCallbackResult = newValue
        }
    }
    var getLanguagesResult: Result<[LanguageInfo], APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getLanguagesResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getLanguagesResult = newValue
        }
    }
    var getLevelsResult: Result<LevelsResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getLevelsResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getLevelsResult = newValue
        }
    }
    var reportQuestionResult: Result<SuccessResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _reportQuestionResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _reportQuestionResult = newValue
        }
    }
    var markQuestionKnownResult: Result<SuccessResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _markQuestionKnownResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _markQuestionKnownResult = newValue
        }
    }
    var getAIConversationResult: Result<Conversation, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getAIConversationResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getAIConversationResult = newValue
        }
    }
    var updateAIConversationTitleResult: Result<SuccessResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _updateAIConversationTitleResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _updateAIConversationTitleResult = newValue
        }
    }
    var deleteAIConversationResult: Result<SuccessResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _deleteAIConversationResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _deleteAIConversationResult = newValue
        }
    }
    var toggleBookmarkResult: Result<BookmarkStatusResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _toggleBookmarkResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _toggleBookmarkResult = newValue
        }
    }
    var logoutResult: Result<SuccessResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _logoutResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _logoutResult = newValue
        }
    }
    var getLearningPreferencesResult: Result<UserLearningPreferences, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getLearningPreferencesResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getLearningPreferencesResult = newValue
        }
    }
    var updateLearningPreferencesResult: Result<UserLearningPreferences, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _updateLearningPreferencesResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _updateLearningPreferencesResult = newValue
        }
    }
    var getAIProvidersResult: Result<AIProvidersResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getAIProvidersResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getAIProvidersResult = newValue
        }
    }
    var testAIConnectionResult: Result<SuccessResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _testAIConnectionResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _testAIConnectionResult = newValue
        }
    }
    var sendTestEmailResult: Result<SuccessResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _sendTestEmailResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _sendTestEmailResult = newValue
        }
    }
    var sendTestIOSNotificationResult: Result<SuccessResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _sendTestIOSNotificationResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _sendTestIOSNotificationResult = newValue
        }
    }
    var clearStoriesResult: Result<SuccessResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _clearStoriesResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _clearStoriesResult = newValue
        }
    }
    var clearAIChatsResult: Result<SuccessResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _clearAIChatsResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _clearAIChatsResult = newValue
        }
    }
    var clearTranslationHistoryResult: Result<SuccessResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _clearTranslationHistoryResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _clearTranslationHistoryResult = newValue
        }
    }
    var resetAccountResult: Result<SuccessResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _resetAccountResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _resetAccountResult = newValue
        }
    }
    var getVoicesResult: Result<[EdgeTTSVoiceInfo], APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getVoicesResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getVoicesResult = newValue
        }
    }
    var getStorySectionResult: Result<StorySectionWithQuestions, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getStorySectionResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getStorySectionResult = newValue
        }
    }
    var getTranslationPracticeHistoryResult: Result<TranslationPracticeHistoryResponse, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _getTranslationPracticeHistoryResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _getTranslationPracticeHistoryResult = newValue
        }
    }
    var createSnippetResult: Result<Snippet, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _createSnippetResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _createSnippetResult = newValue
        }
    }
    var updateSnippetResult: Result<Snippet, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _updateSnippetResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _updateSnippetResult = newValue
        }
    }
    var deleteSnippetResult: Result<Void, APIError>? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _deleteSnippetResult
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _deleteSnippetResult = newValue
        }
    }
    var googleOAuthResponse: GoogleOAuthLoginResponse? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _googleOAuthResponse
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _googleOAuthResponse = newValue
        }
    }
    var dailyAnswerResponse: DailyAnswerResponse? {
        get {
            lock.lock()
            defer { lock.unlock() }
            return _dailyAnswerResponse
        }
        set {
            lock.lock()
            defer { lock.unlock() }
            _dailyAnswerResponse = newValue
        }
    }

    // Method implementations (cannot override extension methods,
    // but these will be used when called on MockAPIService instances)
    func login(request: LoginRequest) -> AnyPublisher<LoginResponse, APIError> {
        guard let result = loginResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func signup(request: UserCreateRequest) -> AnyPublisher<SuccessResponse, APIError> {
        guard let result = signupResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func authStatus() -> AnyPublisher<AuthStatusResponse, APIError> {
        guard let result = authStatusResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getQuestion(
        language: String?, level: String?, type: String?, excludeType: String?
    ) -> AnyPublisher<QuestionFetchResult, APIError> {
        guard let result = getQuestionResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func postAnswer(request: AnswerRequest) -> AnyPublisher<AnswerResponse, APIError> {
        guard let result = postAnswerResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getStories() -> AnyPublisher<[StorySummary], APIError> {
        guard let result = getStoriesResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getStory(id: Int) -> AnyPublisher<StoryContent, APIError> {
        guard let result = getStoryResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getSnippets(
        sourceLang: String?, targetLang: String?, storyId: Int?,
        query: String?, level: String?
    )
        -> AnyPublisher<
            SnippetList, APIError
        >
    {
        guard let result = getSnippetsResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getSnippetsByQuestion(questionId: Int) -> AnyPublisher<SnippetList, APIError> {
        guard let result = getSnippetsResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getDailyQuestions(date: String) -> AnyPublisher<DailyQuestionsResponse, APIError> {
        guard let result = getDailyQuestionsResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func postDailyAnswer(date: String, questionId: Int, userAnswerIndex: Int)
        -> AnyPublisher<DailyAnswerResponse, APIError>
    {
        if let response = dailyAnswerResponse {
            return Just(response)
                .setFailureType(to: APIError.self)
                .eraseToAnyPublisher()
        }
        guard let result = postDailyAnswerResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func generateTranslationSentence(request: TranslationPracticeGenerateRequest)
        -> AnyPublisher<TranslationPracticeSentenceResponse, APIError>
    {
        guard let result = generateTranslationSentenceResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func submitTranslation(request: TranslationPracticeSubmitRequest) -> AnyPublisher<
        TranslationPracticeSessionResponse, APIError
    > {
        guard let result = submitTranslationResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getExistingTranslationSentence(language: String, level: String, direction: String)
        -> AnyPublisher<TranslationPracticeSentenceResponse, APIError>
    {
        guard let result = getExistingTranslationSentenceResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getVerbConjugations(language: String) -> AnyPublisher<
        VerbConjugationsData, APIError
    > {
        guard let result = getVerbConjugationsResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getVerbConjugation(language: String, verb: String) -> AnyPublisher<
        VerbConjugationDetail, APIError
    > {
        if let result = getVerbConjugationResult {
            return result.publisher.eraseToAnyPublisher()
        }
        return Fail(error: .invalidResponse).eraseToAnyPublisher()
    }

    func updateUser(request: UserUpdateRequest) -> AnyPublisher<User, APIError> {
        guard let result = updateUserResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getWordOfTheDay(date: String?) -> AnyPublisher<
        WordOfTheDayDisplay, APIError
    > {
        guard let result = getWordOfTheDayResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getAIConversations() -> AnyPublisher<ConversationListResponse, APIError> {
        guard let result = getAIConversationsResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getBookmarkedMessages() -> AnyPublisher<BookmarkedMessagesResponse, APIError> {
        guard let result = getBookmarkedMessagesResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func handleGoogleCallback(code: String, state: String?) -> AnyPublisher<LoginResponse, APIError>
    {
        guard let result = handleGoogleCallbackResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getLanguages() -> AnyPublisher<[LanguageInfo], APIError> {
        guard let result = getLanguagesResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getLevels(language: String?) -> AnyPublisher<LevelsResponse, APIError> {
        guard let result = getLevelsResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func reportQuestion(id: Int, request: ReportQuestionRequest) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        guard let result = reportQuestionResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func markQuestionKnown(id: Int, request: MarkQuestionKnownRequest) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        guard let result = markQuestionKnownResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getAIConversation(id: String) -> AnyPublisher<Conversation, APIError> {
        guard let result = getAIConversationResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func updateAIConversationTitle(id: String, title: String) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        guard let result = updateAIConversationTitleResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func deleteAIConversation(id: String) -> AnyPublisher<SuccessResponse, APIError> {
        guard let result = deleteAIConversationResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func toggleBookmark(conversationId: String, messageId: String) -> AnyPublisher<
        BookmarkStatusResponse, APIError
    > {
        guard let result = toggleBookmarkResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func logout() -> AnyPublisher<SuccessResponse, APIError> {
        guard let result = logoutResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getLearningPreferences() -> AnyPublisher<UserLearningPreferences, APIError> {
        guard let result = getLearningPreferencesResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func updateLearningPreferences(prefs: UserLearningPreferences) -> AnyPublisher<
        UserLearningPreferences, APIError
    > {
        guard let result = updateLearningPreferencesResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getAIProviders() -> AnyPublisher<AIProvidersResponse, APIError> {
        guard let result = getAIProvidersResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func testAIConnection(provider: String, model: String, apiKey: String?) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        guard let result = testAIConnectionResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func sendTestEmail() -> AnyPublisher<SuccessResponse, APIError> {
        guard let result = sendTestEmailResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func sendTestIOSNotification(notificationType: String) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        guard let result = sendTestIOSNotificationResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func clearStories() -> AnyPublisher<SuccessResponse, APIError> {
        guard let result = clearStoriesResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func clearAIChats() -> AnyPublisher<SuccessResponse, APIError> {
        guard let result = clearAIChatsResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func clearTranslationHistory() -> AnyPublisher<SuccessResponse, APIError> {
        guard let result = clearTranslationHistoryResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func resetAccount() -> AnyPublisher<SuccessResponse, APIError> {
        guard let result = resetAccountResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getVoices(language: String) -> AnyPublisher<[EdgeTTSVoiceInfo], APIError> {
        guard let result = getVoicesResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getStorySection(id: Int) -> AnyPublisher<StorySectionWithQuestions, APIError> {
        guard let result = getStorySectionResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func getTranslationPracticeHistory(limit: Int, offset: Int) -> AnyPublisher<
        TranslationPracticeHistoryResponse, APIError
    > {
        guard let result = getTranslationPracticeHistoryResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func createSnippet(request: CreateSnippetRequest) -> AnyPublisher<Snippet, APIError> {
        guard let result = createSnippetResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func updateSnippet(id: Int, request: UpdateSnippetRequest) -> AnyPublisher<Snippet, APIError> {
        guard let result = updateSnippetResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func deleteSnippet(id: Int) -> AnyPublisher<Void, APIError> {
        guard let result = deleteSnippetResult else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return result.publisher.eraseToAnyPublisher()
    }

    func initiateGoogleLogin() -> AnyPublisher<GoogleOAuthLoginResponse, APIError> {
        guard let response = googleOAuthResponse else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        return Just(response)
            .setFailureType(to: APIError.self)
            .eraseToAnyPublisher()
    }

    func getSnippetsForQuestion(questionId: Int) -> AnyPublisher<SnippetList, APIError> {
        return getSnippetsByQuestion(questionId: questionId)
    }

    func getSnippetsForStory(storyId: Int) -> AnyPublisher<SnippetList, APIError> {
        return getSnippets(
            sourceLang: nil, targetLang: nil, storyId: storyId, query: nil, level: nil)
    }

}

extension Result {
    var publisher: AnyPublisher<Success, Failure> {
        switch self {
        case .success(let value):
            // Use Deferred + Future with immediate async dispatch to ensure reliable delivery
            // This pattern works consistently in both individual and suite test runs
            return Deferred {
                Future<Success, Failure> { promise in
                    // Schedule on next run loop cycle to ensure async delivery
                    DispatchQueue.main.async {
                        promise(.success(value))
                    }
                }
            }
            .eraseToAnyPublisher()
        case .failure(let error):
            return Fail(error: error)
                .eraseToAnyPublisher()
        }
    }
}
