import Foundation
import Combine

class QuizViewModel: BaseViewModel, QuestionActions, SnippetLoading {
    @Published var question: Question?
    @Published var answerResponse: AnswerResponse?
    @Published var generatingMessage: String?
    @Published var selectedAnswerIndex: Int? = nil {
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

    init(question: Question? = nil, questionType: String? = nil, isDaily: Bool = false, apiService: APIService = APIService.shared) {
        self.question = question
        self.questionType = questionType
        self.isDaily = isDaily
        super.init(apiService: apiService)

        if !isDaily, question == nil, let savedState = QuizStateManager.shared.getState(for: questionType) {
            self.question = savedState.question
            self.answerResponse = savedState.answerResponse
            self.selectedAnswerIndex = savedState.selectedAnswerIndex
        }
    }

    func getQuestion() {
        isLoading = true
        clearError()
        generatingMessage = nil
        answerResponse = nil
        selectedAnswerIndex = nil
        isReported = false

        if !isDaily {
            QuizStateManager.shared.clearState(for: questionType)
        }

        apiService.getQuestion(language: nil, level: nil, type: questionType, excludeType: nil)
            .handleLoadingAndError(on: self)
            .sink(receiveValue: { [weak self] result in
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
            })
            .store(in: &cancellables)
    }

    func submitAnswer(userAnswerIndex: Int) {
        guard let question else { return }

        if isDaily {
            submitDailyAnswer(userAnswerIndex: userAnswerIndex)
            return
        }

        let answerRequest = AnswerRequest(questionId: question.id, userAnswerIndex: userAnswerIndex, responseTimeMs: nil)
        apiService.postAnswer(request: answerRequest)
            .handleErrorOnly(on: self)
            .sink(receiveValue: { [weak self] response in
                guard let self else { return }
                self.answerResponse = response
                self.saveState()
            })
            .store(in: &cancellables)
    }

    func submitDailyAnswer(userAnswerIndex: Int) {
        guard let question else { return }
        let today = Date().iso8601String

        apiService.postDailyAnswer(date: today, questionId: question.id, userAnswerIndex: userAnswerIndex)
            .handleErrorOnly(on: self)
            .sink(receiveValue: { [weak self] response in
                self?.answerResponse = AnswerResponse(isCorrect: response.isCorrect,
                                                      userAnswer: response.userAnswer,
                                                      userAnswerIndex: response.userAnswerIndex,
                                                      explanation: response.explanation,
                                                      correctAnswerIndex: response.correctAnswerIndex)
            })
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
}

extension QuizViewModel {
    func reportQuestion(reason: String?) {
        guard let question = question else { return }
        reportQuestion(id: question.id, reason: reason)
    }

    func markQuestionKnown(confidence: Int) {
        guard let question = question else { return }
        markQuestionKnown(id: question.id, confidence: confidence)
    }
}
