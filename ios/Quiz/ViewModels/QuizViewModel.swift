import Foundation
import Combine

class QuizViewModel: ObservableObject {
    @Published var question: Question?
    @Published var answerResponse: AnswerResponse?
    @Published var generatingMessage: String?
    @Published var error: APIService.APIError?
    @Published var isLoading = false
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


    var apiService: APIService
    private var cancellables = Set<AnyCancellable>()
    let questionType: String?
    private let isDaily: Bool

    init(question: Question? = nil, questionType: String? = nil, isDaily: Bool = false, apiService: APIService = APIService.shared) {
        self.question = question
        self.questionType = questionType
        self.isDaily = isDaily
        self.apiService = apiService

        if !isDaily, question == nil, let savedState = QuizStateManager.shared.getState(for: questionType) {
            self.question = savedState.question
            self.answerResponse = savedState.answerResponse
            self.selectedAnswerIndex = savedState.selectedAnswerIndex
        }
    }

    func getQuestion() {
        isLoading = true
        error = nil
        generatingMessage = nil
        answerResponse = nil
        selectedAnswerIndex = nil
        isReported = false

        if !isDaily {
            QuizStateManager.shared.clearState(for: questionType)
        }

        apiService.getQuestion(language: nil, level: nil, type: questionType, excludeType: nil)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                guard let self else { return }
                self.isLoading = false
                switch completion {
                case .failure(let error):
                    self.error = error
                case .finished:
                    break
                }
            }, receiveValue: { [weak self] result in
                guard let self else { return }
                switch result {
                case .question(let question):
                    self.question = question
                    self.generatingMessage = nil
                    self.getSnippets(questionId: question.id)
                    self.saveState()
                case .generating(let status):
                    self.question = nil
                    self.generatingMessage = status.message
                }
            })
            .store(in: &cancellables)
    }

    func getSnippets(questionId: Int) {
        apiService.getSnippetsForQuestion(questionId: questionId)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] snippetList in
                self?.snippets = snippetList.snippets
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
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                guard let self else { return }
                switch completion {
                case .failure(let error):
                    self.error = error
                case .finished:
                    break
                }
            }, receiveValue: { [weak self] response in
                guard let self else { return }
                self.answerResponse = response
                self.saveState()
            })
            .store(in: &cancellables)
    }

    func submitDailyAnswer(userAnswerIndex: Int) {
        guard let question else { return }
        let today = DateFormatters.iso8601.string(from: Date())

        apiService.postDailyAnswer(date: today, questionId: question.id, userAnswerIndex: userAnswerIndex)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] response in
                self?.answerResponse = AnswerResponse(isCorrect: response.isCorrect,
                                                      userAnswer: "",
                                                      userAnswerIndex: userAnswerIndex,
                                                      explanation: response.explanation,
                                                      correctAnswerIndex: 0) // Placeholder
            })
            .store(in: &cancellables)
    }


    func reportQuestion(reason: String?) {
        guard let question = question else { return }
        isSubmittingAction = true
        let request = ReportQuestionRequest(reportReason: reason)
        apiService.reportQuestion(id: question.id, request: request)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isSubmittingAction = false
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] _ in
                self?.isReported = true
                self?.showReportModal = false
            })
            .store(in: &cancellables)
    }

    func markQuestionKnown(confidence: Int) {
        guard let question = question else { return }
        isSubmittingAction = true
        let request = MarkQuestionKnownRequest(confidenceLevel: confidence)
        apiService.markQuestionKnown(id: question.id, request: request)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isSubmittingAction = false
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] _ in
                self?.showMarkKnownModal = false
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
