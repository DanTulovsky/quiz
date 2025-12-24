import Combine
import Foundation

class DailyViewModel: BaseViewModel, QuestionActions, SnippetLoading, SubmittingState,
    QuestionIDProvider
{
    @Published var dailyQuestions: [DailyQuestionWithDetails] = []
    @Published var currentQuestionIndex = -1
    @Published var snippets = [Snippet]()
    @Published var isPositioned = false

    @Published var selectedAnswerIndex: Int?
    @Published var answerResponse: DailyAnswerResponse?
    @Published var isSubmitting = false

    @Published var showReportModal = false
    @Published var showMarkKnownModal = false
    @Published var isReported = false
    @Published var isSubmittingAction = false

    private var lastLoadedQuestionId: Int?

    var currentQuestion: DailyQuestionWithDetails? {
        guard currentQuestionIndex >= 0 && currentQuestionIndex < dailyQuestions.count else {
            return nil
        }
        return dailyQuestions[currentQuestionIndex]
    }

    var progress: Double {
        guard !dailyQuestions.isEmpty && currentQuestionIndex >= 0 else { return 0 }
        return Double(currentQuestionIndex + 1) / Double(dailyQuestions.count)
    }

    var isAllCompleted: Bool {
        !dailyQuestions.isEmpty && dailyQuestions.allSatisfy { $0.isCompleted }
    }

    var completedCount: Int {
        dailyQuestions.filter { $0.isCompleted }.count
    }

    var totalCount: Int {
        dailyQuestions.count
    }

    var hasPreviousQuestion: Bool {
        currentQuestionIndex > 0
    }

    var hasNextQuestion: Bool {
        currentQuestionIndex < dailyQuestions.count - 1
    }

    override init(apiService: APIServiceProtocol = APIService.shared) {
        super.init(apiService: apiService)
    }

    func fetchDaily() {
        let today = Date().iso8601String

        // Reset positioning state
        isPositioned = false
        currentQuestionIndex = -1

        apiService.getDailyQuestions(date: today)
            .handleLoadingAndError(on: self)
            .receive(on: DispatchQueue.main)
            .sinkValue(on: self) { [weak self] response in
                guard let self = self else { return }

                self.dailyQuestions = response.questions

                // Always position on the first incomplete question when questions are loaded
                // This ensures users never start on a completed question
                self.positionOnFirstIncomplete()

                // Mark as positioned after setting the index
                self.isPositioned = true

                // Load snippets for the positioned question
                if let questionId = self.currentQuestion?.question.id {
                    self.loadSnippets(questionId: questionId)
                }
            }
            .store(in: &cancellables)
    }

    private func positionOnFirstIncomplete() {
        guard !dailyQuestions.isEmpty else {
            currentQuestionIndex = -1
            return
        }

        // Find the first incomplete question
        if let firstIncomplete = dailyQuestions.firstIndex(where: { !$0.isCompleted }) {
            currentQuestionIndex = firstIncomplete
        } else {
            // All questions are completed, start at the first one
            currentQuestionIndex = 0
        }
    }

    func ensurePositionedOnFirstIncomplete() {
        guard !dailyQuestions.isEmpty else {
            currentQuestionIndex = -1
            isPositioned = false
            return
        }

        // Check if current question is completed or invalid
        let currentIsCompleted =
            currentQuestionIndex >= 0
            && currentQuestionIndex < dailyQuestions.count
            && dailyQuestions[currentQuestionIndex].isCompleted
        let currentIsInvalid =
            currentQuestionIndex < 0 || currentQuestionIndex >= dailyQuestions.count

        // If we're on a completed question or invalid index, find the first incomplete question
        if currentIsCompleted || currentIsInvalid {
            positionOnFirstIncomplete()
            isPositioned = true
        }
        // If current question is valid and not completed, we're already positioned correctly
    }

    func validateCurrentQuestionPosition() {
        guard !dailyQuestions.isEmpty else {
            if currentQuestionIndex != -1 {
                currentQuestionIndex = -1
                isPositioned = false
            }
            return
        }

        // Validate that current index is within bounds
        if currentQuestionIndex < 0 || currentQuestionIndex >= dailyQuestions.count {
            positionOnFirstIncomplete()
            isPositioned = true
            return
        }

        // If current question is completed and not all are completed, reposition
        let currentQuestion = dailyQuestions[currentQuestionIndex]
        if currentQuestion.isCompleted && !isAllCompleted {
            positionOnFirstIncomplete()
            isPositioned = true
        }
    }

    func submitAnswer(index: Int) {
        guard let question = currentQuestion else { return }
        selectedAnswerIndex = index

        let today = Date().iso8601String

        executeWithSubmittingState(
            publisher: apiService.postDailyAnswer(
                date: today, questionId: question.question.id, userAnswerIndex: index)
        ) { [weak self] (response: DailyAnswerResponse) in
            guard let self = self else { return }
            self.answerResponse = response

            // Update the question's completed status in the array
            guard
                self.currentQuestionIndex >= 0
                    && self.currentQuestionIndex < self.dailyQuestions.count
            else {
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
        }
        .store(in: &cancellables)
    }

    func nextQuestion() {
        answerResponse = nil
        selectedAnswerIndex = nil
        lastLoadedQuestionId = nil
        snippets = []

        if isAllCompleted {
            // When all completed, allow sequential navigation
            if currentQuestionIndex < dailyQuestions.count - 1 {
                currentQuestionIndex += 1
            }
        } else {
            // Find next unanswered question
            if let nextIncompleteIndex = dailyQuestions.enumerated().first(where: {
                index, question in
                index > currentQuestionIndex && !question.isCompleted
            })?.offset {
                currentQuestionIndex = nextIncompleteIndex
            }
        }
    }

    func previousQuestion() {
        answerResponse = nil
        selectedAnswerIndex = nil
        lastLoadedQuestionId = nil
        snippets = []

        if currentQuestionIndex > 0 {
            currentQuestionIndex -= 1
        }
    }

    var currentQuestionId: Int? {
        return currentQuestion?.question.id
    }

    func resetSnippetCache() {
        lastLoadedQuestionId = nil
    }

    func loadSnippets(questionId: Int? = nil, storyId: Int? = nil) {
        // For DailyViewModel, we should always have a questionId - don't load all snippets
        guard let questionId = questionId else {
            // If no questionId provided, don't make any API call
            return
        }

        // Prevent duplicate API calls for the same question
        if questionId == lastLoadedQuestionId {
            return
        }

        lastLoadedQuestionId = questionId

        // Always use getSnippetsByQuestion for daily questions
        let publisher = apiService.getSnippetsByQuestion(questionId: questionId)

        publisher
            .catch { _ -> AnyPublisher<SnippetList, APIService.APIError> in
                // Return empty snippet list instead of propagating error
                return Just(SnippetList(limit: 0, offset: 0, query: nil, snippets: []))
                    .setFailureType(to: APIService.APIError.self)
                    .eraseToAnyPublisher()
            }
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { _ in },
                receiveValue: { [weak self] snippetList in
                    guard let self = self else { return }
                    // Filter snippets to only include those for the current question
                    // The API should already filter, but we do it here as a safety measure
                    let filteredSnippets = snippetList.snippets.filter {
                        $0.questionId == questionId
                    }
                    // Create a new array to ensure SwiftUI detects the change
                    self.snippets = Array(filteredSnippets)
                }
            )
            .store(in: &cancellables)
    }
}
