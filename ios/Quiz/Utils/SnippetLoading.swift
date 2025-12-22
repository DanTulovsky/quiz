import Foundation
import Combine

protocol SnippetLoading: BaseViewModel {
    var snippets: [Snippet] { get set }
}

extension SnippetLoading {
    func loadSnippets(questionId: Int? = nil, storyId: Int? = nil) {
        print("🔵 [SnippetLoading Protocol Extension] loadSnippets called - questionId: \(questionId?.description ?? "nil"), storyId: \(storyId?.description ?? "nil")")
        print("🔵 [SnippetLoading] Caller: \(String(describing: type(of: self)))")

        let publisher: AnyPublisher<SnippetList, APIService.APIError>

        if let questionId = questionId {
            print("🔵 [SnippetLoading] Using getSnippetsByQuestion(\(questionId))")
            publisher = apiService.getSnippetsByQuestion(questionId: questionId)
        } else {
            print("🔵 [SnippetLoading] Using getSnippets() - GENERAL ENDPOINT")
            publisher = apiService.getSnippets(
                sourceLang: nil,
                targetLang: nil,
                storyId: storyId,
                query: nil,
                level: nil
            )
        }

        publisher
            .catch { error -> AnyPublisher<SnippetList, APIService.APIError> in
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

