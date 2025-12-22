import Combine
import Foundation

protocol APIServiceProtocol {
    func login(request: LoginRequest) -> AnyPublisher<LoginResponse, APIService.APIError>
    func signup(request: UserCreateRequest) -> AnyPublisher<SuccessResponse, APIService.APIError>
    func authStatus() -> AnyPublisher<AuthStatusResponse, APIService.APIError>
    func initiateGoogleLogin() -> AnyPublisher<GoogleOAuthLoginResponse, APIService.APIError>
    func handleGoogleCallback(code: String, state: String?) -> AnyPublisher<
        LoginResponse, APIService.APIError
    >
    func updateUser(request: UserUpdateRequest) -> AnyPublisher<User, APIService.APIError>

    func getQuestion(
        language: String?, level: String?, type: String?, excludeType: String?
    ) -> AnyPublisher<APIService.QuestionFetchResult, APIService.APIError>
    func postAnswer(request: AnswerRequest) -> AnyPublisher<AnswerResponse, APIService.APIError>
    func getDailyQuestions(date: String) -> AnyPublisher<
        DailyQuestionsResponse, APIService.APIError
    >
    func postDailyAnswer(date: String, questionId: Int, userAnswerIndex: Int) -> AnyPublisher<
        DailyAnswerResponse, APIService.APIError
    >

    func getStories() -> AnyPublisher<[StorySummary], APIService.APIError>
    func getStory(id: Int) -> AnyPublisher<StoryContent, APIService.APIError>

    func getSnippets(
        sourceLang: String?, targetLang: String?, storyId: Int?, query: String?,
        level: String?
    ) -> AnyPublisher<SnippetList, APIService.APIError>
    func getSnippetsByQuestion(questionId: Int) -> AnyPublisher<SnippetList, APIService.APIError>

    func generateTranslationSentence(request: TranslationPracticeGenerateRequest) -> AnyPublisher<
        TranslationPracticeSentenceResponse, APIService.APIError
    >
    func submitTranslation(request: TranslationPracticeSubmitRequest) -> AnyPublisher<
        TranslationPracticeSessionResponse, APIService.APIError
    >
    func getExistingTranslationSentence(language: String, level: String, direction: String)
    -> AnyPublisher<TranslationPracticeSentenceResponse, APIService.APIError>

    func getVerbConjugations(language: String) -> AnyPublisher<
        VerbConjugationsData, APIService.APIError
    >
    func getVerbConjugation(language: String, verb: String) -> AnyPublisher<
        VerbConjugationDetail, APIService.APIError
    >
    func getWordOfTheDay(date: String?) -> AnyPublisher<WordOfTheDayDisplay, APIService.APIError>
    func getAIConversations() -> AnyPublisher<ConversationListResponse, APIService.APIError>
    func getBookmarkedMessages() -> AnyPublisher<BookmarkedMessagesResponse, APIService.APIError>
    func getLanguages() -> AnyPublisher<[LanguageInfo], APIService.APIError>

    func getLevels(language: String?) -> AnyPublisher<LevelsResponse, APIService.APIError>
    func reportQuestion(id: Int, request: ReportQuestionRequest) -> AnyPublisher<
        SuccessResponse, APIService.APIError
    >
    func markQuestionKnown(id: Int, request: MarkQuestionKnownRequest) -> AnyPublisher<
        SuccessResponse, APIService.APIError
    >

    func getAIConversation(id: String) -> AnyPublisher<Conversation, APIService.APIError>
    func updateAIConversationTitle(id: String, title: String) -> AnyPublisher<
        SuccessResponse, APIService.APIError
    >
    func deleteAIConversation(id: String) -> AnyPublisher<SuccessResponse, APIService.APIError>
    func toggleBookmark(conversationId: String, messageId: String) -> AnyPublisher<
        BookmarkStatusResponse, APIService.APIError
    >
    func logout() -> AnyPublisher<SuccessResponse, APIService.APIError>

    func getLearningPreferences() -> AnyPublisher<UserLearningPreferences, APIService.APIError>
    func updateLearningPreferences(prefs: UserLearningPreferences) -> AnyPublisher<
        UserLearningPreferences, APIService.APIError
    >
    func getAIProviders() -> AnyPublisher<AIProvidersResponse, APIService.APIError>
    func testAIConnection(provider: String, model: String, apiKey: String?) -> AnyPublisher<
        SuccessResponse, APIService.APIError
    >
    func sendTestEmail() -> AnyPublisher<SuccessResponse, APIService.APIError>
    func clearStories() -> AnyPublisher<SuccessResponse, APIService.APIError>
    func clearAIChats() -> AnyPublisher<SuccessResponse, APIService.APIError>
    func clearTranslationHistory() -> AnyPublisher<SuccessResponse, APIService.APIError>
    func resetAccount() -> AnyPublisher<SuccessResponse, APIService.APIError>
    func getVoices(language: String) -> AnyPublisher<[EdgeTTSVoiceInfo], APIService.APIError>
    func getStorySection(id: Int) -> AnyPublisher<StorySectionWithQuestions, APIService.APIError>

    func getTranslationPracticeHistory(limit: Int, offset: Int) -> AnyPublisher<
        TranslationPracticeHistoryResponse, APIService.APIError
    >
    func createSnippet(request: CreateSnippetRequest) -> AnyPublisher<Snippet, APIService.APIError>
    func updateSnippet(id: Int, request: UpdateSnippetRequest) -> AnyPublisher<
        Snippet, APIService.APIError
    >
    func deleteSnippet(id: Int) -> AnyPublisher<Void, APIService.APIError>
}
