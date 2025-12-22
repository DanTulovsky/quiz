import Combine
import Foundation

class QuizViewModel: BaseViewModel, QuestionActions, SnippetLoading, QuestionIDProvider,
                     AnswerSubmittable {
    @Published var question: Question?
    @Published var answerResponse: AnswerResponse?
    @Published var generatingMessage: String?
    @Published var selectedAnswerIndex: Int? {
        didSet {
            saveState()
        }
    }
    @Published var snippets = [Snippet]()

    @Published var isReported = false
    @Published var showReportModal = false
    @Published var showMarkKnownModal = false
    @Published var isSubmittingAction = false

    let questionType: String?
    private let isDaily: Bool
    private var lastLoadedQuestionId: Int?
    private var lastLoadedStoryId: Int?

    init(
        question: Question? = nil, questionType: String? = nil, isDaily: Bool = false,
        apiService: APIServiceProtocol = APIService.shared
    ) {
        self.question = question
        self.questionType = questionType
        self.isDaily = isDaily
        super.init(apiService: apiService)

        if !isDaily, question == nil,
           let savedState = QuizStateManager.shared.getState(for: questionType) {
            self.question = savedState.question
            self.answerResponse = savedState.answerResponse
            self.selectedAnswerIndex = savedState.selectedAnswerIndex
            // Load snippets for the saved question
            if let questionId = savedState.question?.id {
                self.lastLoadedQuestionId = questionId
                self.loadSnippets(questionId: questionId)
            }
        }
    }

    func getQuestion() {
        generatingMessage = nil
        answerResponse = nil
        selectedAnswerIndex = nil
        isReported = false
        lastLoadedQuestionId = nil
        lastLoadedStoryId = nil
        snippets = []

        if !isDaily {
            QuizStateManager.shared.clearState(for: questionType)
        }

        apiService.getQuestion(language: nil, level: nil, type: questionType, excludeType: nil)
            .handleLoadingAndError(on: self)
            .sinkValue(on: self) { [weak self] result in
                guard let self else { return }
                switch result {
                case .question(let question):
                    self.question = question
                    self.generatingMessage = nil
                    self.loadSnippets(questionId: question.id)
                    self.saveState()
                case .generating(let status):
                    self.question = nil
                    self.generatingMessage = status.message
                }
            }
            .store(in: &cancellables)
    }

    func loadSnippets(questionId: Int? = nil, storyId: Int? = nil) {
        let publisher: AnyPublisher<SnippetList, APIService.APIError>

        if let questionId = questionId {
            // Prevent duplicate API calls for the same question
            if questionId == lastLoadedQuestionId {
                return
            }
            lastLoadedQuestionId = questionId
            publisher = apiService.getSnippetsByQuestion(questionId: questionId)
        } else if let storyId = storyId {
            // Prevent duplicate API calls for the same story
            if storyId == lastLoadedStoryId {
                return
            }
            lastLoadedStoryId = storyId
            publisher = apiService.getSnippets(
                sourceLang: nil,
                targetLang: nil,
                storyId: storyId,
                query: nil,
                level: nil
            )
        } else {
            // If neither questionId nor storyId provided, don't make any API call
            return
        }

        publisher
            .catch { _ -> AnyPublisher<SnippetList, APIService.APIError> in
                // Silently handle snippet loading errors - snippets are optional
                // Return empty snippet list instead of propagating error
                return Just(SnippetList(limit: 0, offset: 0, query: nil, snippets: []))
                    .setFailureType(to: APIService.APIError.self)
                    .eraseToAnyPublisher()
            }
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { _ in },
                receiveValue: { [weak self] snippetList in
                    self?.snippets = snippetList.snippets
                }
            )
            .store(in: &cancellables)
    }

    func submitAnswer(userAnswerIndex: Int) {
        guard let question else { return }

        if isDaily {
            submitDailyAnswer(userAnswerIndex: userAnswerIndex)
            return
        }

        let answerRequest = AnswerRequest(
            questionId: question.id, userAnswerIndex: userAnswerIndex, responseTimeMs: nil)
        submitAnswer(
            publisher: apiService.postAnswer(request: answerRequest)
        ) { [weak self] response in
            guard let self else { return }
            self.answerResponse = response
            self.saveState()
        }
        .store(in: &cancellables)
    }

    func submitDailyAnswer(userAnswerIndex: Int) {
        guard let question else { return }
        let today = Date().iso8601String

        apiService.postDailyAnswer(
            date: today, questionId: question.id, userAnswerIndex: userAnswerIndex
        )
        .handleErrorOnly(on: self)
        .sinkValue(on: self) { [weak self] response in
            self?.answerResponse = AnswerResponse(
                isCorrect: response.isCorrect,
                userAnswer: response.userAnswer,
                userAnswerIndex: response.userAnswerIndex,
                explanation: response.explanation,
                correctAnswerIndex: response.correctAnswerIndex)
        }
        .store(in: &cancellables)
    }

    private func saveState() {
        guard !isDaily else { return }
        let state = QuizStateManager.QuizState(
            question: question,
            answerResponse: answerResponse,
            selectedAnswerIndex: selectedAnswerIndex
        )
        QuizStateManager.shared.saveState(for: questionType, state: state)
    }

    var currentQuestionId: Int? {
        return question?.id
    }

    func resetSnippetCache() {
        lastLoadedQuestionId = nil
        lastLoadedStoryId = nil
    }
}
