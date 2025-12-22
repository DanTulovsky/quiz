import Combine
import Foundation

extension APIService {

    func reportQuestion(id: Int, request: ReportQuestionRequest) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        return post(
            path: "quiz/question/\(id)/report", body: request, responseType: SuccessResponse.self)
    }

    func markQuestionKnown(id: Int, request: MarkQuestionKnownRequest) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        return post(
            path: "quiz/question/\(id)/mark-known", body: request,
            responseType: SuccessResponse.self)
    }
}
