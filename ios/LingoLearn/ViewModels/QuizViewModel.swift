import Foundation
import Combine

class QuizViewModel: ObservableObject {
    @Published var question: Question?
    @Published var answer = ""
    @Published var answerResponse: AnswerResponse?
    @Published var error: APIService.APIError?

    var apiService: APIService
    private var cancellables = Set<AnyCancellable>()
    private let questionType: String?
    
    init(questionType: String? = nil, apiService: APIService = APIService.shared) {
        self.questionType = questionType
        self.apiService = apiService
    }

    func getQuestion() {
        apiService.getQuestion(language: nil, level: nil, type: questionType, excludeType: nil)
            .sink(receiveCompletion: { completion in
                switch completion {
                case .failure(let error):
                    self.error = error
                case .finished:
                    break
                }
            }, receiveValue: { question in
                self.question = question
                self.answerResponse = nil
                self.answer = ""
            })
            .store(in: &cancellables)
    }

    func submitAnswer() {
        guard let question = question else { return }

        let answerRequest = AnswerRequest(questionId: question.id, answer: answer)
        apiService.postAnswer(request: answerRequest)
            .sink(receiveCompletion: { completion in
                switch completion {
                case .failure(let error):
                    self.error = error
                case .finished:
                    break
                }
            }, receiveValue: { response in
                self.answerResponse = response
            })
            .store(in: &cancellables)
    }
}
