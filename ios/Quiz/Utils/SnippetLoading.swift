import Foundation
import Combine

protocol SnippetLoading: BaseViewModel {
    var snippets: [Snippet] { get set }
}

extension SnippetLoading {
    func loadSnippets(questionId: Int? = nil, storyId: Int? = nil) {
        apiService.getSnippets(
            sourceLang: nil,
            targetLang: nil,
            storyId: storyId,
            query: nil,
            level: nil
        )
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

