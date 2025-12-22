import Combine
import Foundation

extension APIService {

    func getStorySection(id: Int) -> AnyPublisher<StorySectionWithQuestions, APIError> {
        return get(path: "story/section/\(id)", responseType: StorySectionWithQuestions.self)
    }
}
