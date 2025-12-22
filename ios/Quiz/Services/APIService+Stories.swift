import Combine
import Foundation

extension APIService {
    func getStories() -> AnyPublisher<[StorySummary], APIError> {
        return get(path: "story", responseType: [StorySummary].self)
    }

    func getStory(id: Int) -> AnyPublisher<StoryContent, APIError> {
        return get(path: "story/\(id)", responseType: StoryContent.self)
    }

    func getStorySection(id: Int) -> AnyPublisher<StorySectionWithQuestions, APIError> {
        return get(path: "story/section/\(id)", responseType: StorySectionWithQuestions.self)
    }
}
