import Combine
import Foundation

protocol SnippetLoading: BaseViewModel {
    var snippets: [Snippet] { get set }
}

extension SnippetLoading {
    func loadSnippets(questionId: Int? = nil, storyId: Int? = nil) {
        let publisher: AnyPublisher<SnippetList, APIService.APIError>

        if let questionId = questionId {
            publisher = apiService.getSnippetsByQuestion(questionId: questionId)
        } else {
            publisher = apiService.getSnippets(
                sourceLang: nil,
                targetLang: nil,
                storyId: storyId,
                query: nil,
                level: nil
            )
        }

        publisher
            .catch { _ -> AnyPublisher<SnippetList, APIService.APIError> in
                // Silently handle snippet loading errors - snippets are optional
                // Return empty snippet list instead of propagating error
                return Just(SnippetList(limit: 0, offset: 0, query: nil, snippets: []))
                    .setFailureType(to: APIService.APIError.self)
                    .eraseToAnyPublisher()
            }
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { _ in },
                receiveValue: { [weak self] snippetList in
                    self?.snippets = snippetList.snippets
                }
            )
            .store(in: &cancellables)
    }
}
