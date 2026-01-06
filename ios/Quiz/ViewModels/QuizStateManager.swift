import Foundation
import Combine

class QuizStateManager: ObservableObject {
    static let shared = QuizStateManager()

    @Published private var states: [String?: QuizState] = [:]

    private init() {}

    struct QuizState {
        var question: Question?
        var answerResponse: AnswerResponse?
        var selectedAnswerIndex: Int?
    }

    func getState(for questionType: String?) -> QuizState? {
        return states[questionType]
    }

    func saveState(for questionType: String?, state: QuizState) {
        states[questionType] = state
    }

    func clearState(for questionType: String?) {
        states[questionType] = nil
    }
}
