import Foundation
import Combine

class DailyViewModel: BaseViewModel, QuestionActions, SnippetLoading {
    @Published var dailyQuestions: [DailyQuestionWithDetails] = []
    @Published var currentQuestionIndex = 0
    @Published var snippets = [Snippet]()

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

    override init(apiService: APIService = APIService.shared) {
        super.init(apiService: apiService)
    }

    func fetchDaily() {
        isLoading = true
        let today = Date().iso8601String

        apiService.getDailyQuestions(date: today)
            .handleLoadingAndError(on: self)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] response in
                guard let self = self else { return }
                self.dailyQuestions = response.questions
                // Always position on the first incomplete question when questions are loaded
                // This ensures users never start on a completed question
                if let firstIncomplete = response.questions.firstIndex(where: { !$0.isCompleted }) {
                    self.currentQuestionIndex = firstIncomplete
                } else if !response.questions.isEmpty {
                    // All questions are completed, start at the first one
                    self.currentQuestionIndex = 0
                }
                self.loadSnippets(questionId: self.currentQuestion?.question.id)
            })
            .store(in: &cancellables)
    }

    func ensurePositionedOnFirstIncomplete() {
        guard !dailyQuestions.isEmpty else { return }

        // Check if current question is completed or invalid
        let currentIsCompleted = currentQuestionIndex < dailyQuestions.count && dailyQuestions[currentQuestionIndex].isCompleted
        let currentIsInvalid = currentQuestionIndex >= dailyQuestions.count

        // If we're on a completed question or invalid index, find the first incomplete question
        if currentIsCompleted || currentIsInvalid {
            if let firstIncomplete = dailyQuestions.firstIndex(where: { !$0.isCompleted }) {
                currentQuestionIndex = firstIncomplete
            } else {
                // All questions are completed, go to first question
                currentQuestionIndex = 0
            }
        }
        // If current question is valid and not completed, we're already positioned correctly
    }

    func submitAnswer(index: Int) {
        guard let question = currentQuestion else { return }
        selectedAnswerIndex = index
        isSubmitting = true

        let today = Date().iso8601String

        apiService.postDailyAnswer(date: today, questionId: question.question.id, userAnswerIndex: index)
            .receive(on: DispatchQueue.main)
            .handleEvents(receiveCompletion: { [weak self] _ in
                self?.isSubmitting = false
            })
            .handleErrorOnly(on: self)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] response in
                guard let self = self else { return }
                self.answerResponse = response

                // Update the question's completed status in the array
                guard self.currentQuestionIndex >= 0 && self.currentQuestionIndex < self.dailyQuestions.count else {
                    return
                }
                let updatedQuestion = DailyQuestionWithDetails(
                    id: self.dailyQuestions[self.currentQuestionIndex].id,
                    questionId: self.dailyQuestions[self.currentQuestionIndex].questionId,
                    question: self.dailyQuestions[self.currentQuestionIndex].question,
                    isCompleted: response.isCompleted,
                    userAnswerIndex: response.userAnswerIndex
                )
                self.dailyQuestions[self.currentQuestionIndex] = updatedQuestion
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

}

extension DailyViewModel {
    func reportQuestion(reason: String) {
        guard let question = currentQuestion else { return }
        reportQuestion(id: question.question.id, reason: reason)
    }

    func markQuestionKnown(confidence: Int) {
        guard let question = currentQuestion else { return }
        markQuestionKnown(id: question.question.id, confidence: confidence)
    }
}
