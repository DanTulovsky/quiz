import Foundation
import Combine

class DailyViewModel: ObservableObject {
    @Published var dailyQuestions: [DailyQuestionWithDetails] = []
    @Published var currentQuestionIndex = 0
    @Published var isLoading = false
    @Published var error: APIService.APIError?

    @Published var selectedAnswerIndex: Int? = nil
    @Published var answerResponse: DailyAnswerResponse? = nil
    @Published var isSubmitting = false

    @Published var showReportModal = false
    @Published var showMarkKnownModal = false
    @Published var isReported = false
    @Published var isSubmittingAction = false

    var currentQuestion: DailyQuestionWithDetails? {
        guard currentQuestionIndex < dailyQuestions.count else { return nil }
        return dailyQuestions[currentQuestionIndex]
    }

    var progress: Double {
        guard !dailyQuestions.isEmpty else { return 0 }
        return Double(currentQuestionIndex + 1) / Double(dailyQuestions.count)
    }

    var isAllCompleted: Bool {
        !dailyQuestions.isEmpty && dailyQuestions.allSatisfy { $0.isCompleted }
    }

    var hasPreviousQuestion: Bool {
        currentQuestionIndex > 0
    }

    var hasNextQuestion: Bool {
        currentQuestionIndex < dailyQuestions.count - 1
    }

    private var apiService: APIService
    private var cancellables = Set<AnyCancellable>()

    init(apiService: APIService = APIService.shared) {
        self.apiService = apiService
    }

    func fetchDaily() {
        isLoading = true
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let today = formatter.string(from: Date())

        apiService.getDailyQuestions(date: today)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isLoading = false
                if case .failure(let error) = completion {
                    self?.error = error
                }
            }, receiveValue: { [weak self] response in
                self?.dailyQuestions = response.questions
                // Find first incomplete question
                if let firstIncomplete = response.questions.firstIndex(where: { !$0.isCompleted }) {
                    self?.currentQuestionIndex = firstIncomplete
                }
            })
            .store(in: &cancellables)
    }

    func submitAnswer(index: Int) {
        guard let question = currentQuestion else { return }
        selectedAnswerIndex = index
        isSubmitting = true

        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let today = formatter.string(from: Date())

        apiService.postDailyAnswer(date: today, questionId: question.question.id, userAnswerIndex: index)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isSubmitting = false
                if case .failure(let error) = completion {
                    self?.error = error
                }
            }, receiveValue: { [weak self] response in
                self?.answerResponse = response
            })
            .store(in: &cancellables)
    }

    func nextQuestion() {
        answerResponse = nil
        selectedAnswerIndex = nil

        if isAllCompleted {
            // When all completed, allow sequential navigation
            if currentQuestionIndex < dailyQuestions.count - 1 {
                currentQuestionIndex += 1
            }
        } else {
            // Find next unanswered question
            if let nextIncompleteIndex = dailyQuestions.enumerated().first(where: { index, question in
                index > currentQuestionIndex && !question.isCompleted
            })?.offset {
                currentQuestionIndex = nextIncompleteIndex
            }
        }
    }

    func previousQuestion() {
        answerResponse = nil
        selectedAnswerIndex = nil

        if currentQuestionIndex > 0 {
            currentQuestionIndex -= 1
        }
    }

    func reportQuestion(reason: String) {
        guard let question = currentQuestion else { return }
        isSubmittingAction = true

        let request = ReportQuestionRequest(reportReason: reason)
        apiService.reportQuestion(id: question.question.id, request: request)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isSubmittingAction = false
                if case .failure(let error) = completion {
                    self?.error = error
                } else {
                    self?.isReported = true
                    self?.showReportModal = false
                }
            }, receiveValue: { _ in })
            .store(in: &cancellables)
    }

    func markQuestionKnown(confidence: Int) {
        guard let question = currentQuestion else { return }
        isSubmittingAction = true

        let request = MarkQuestionKnownRequest(confidenceLevel: confidence)
        apiService.markQuestionKnown(id: question.question.id, request: request)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isSubmittingAction = false
                if case .failure(let error) = completion {
                    self?.error = error
                } else {
                    self?.showMarkKnownModal = false
                }
            }, receiveValue: { _ in })
            .store(in: &cancellables)
    }
}
