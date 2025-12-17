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
    
    var currentQuestion: DailyQuestionWithDetails? {
        guard currentQuestionIndex < dailyQuestions.count else { return nil }
        return dailyQuestions[currentQuestionIndex]
    }
    
    var progress: Double {
        guard !dailyQuestions.isEmpty else { return 0 }
        return Double(currentQuestionIndex + 1) / Double(dailyQuestions.count)
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
        if currentQuestionIndex < dailyQuestions.count - 1 {
            currentQuestionIndex += 1
        }
    }
}
