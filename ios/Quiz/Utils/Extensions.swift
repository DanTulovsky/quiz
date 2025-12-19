import Foundation
import Combine

extension Array where Element == LanguageInfo {
    func find(byCodeOrName codeOrName: String) -> LanguageInfo? {
        let lowercased = codeOrName.lowercased()
        return first(where: {
            $0.name.lowercased() == lowercased || $0.code.lowercased() == lowercased
        })
    }
}

extension AnyPublisher {
    func handleError<T: ObservableObject>(
        on object: T,
        errorPath: ReferenceWritableKeyPath<T, APIService.APIError?>,
        isLoadingPath: ReferenceWritableKeyPath<T, Bool>? = nil
    ) -> AnyPublisher<Output, Failure> where Failure == APIService.APIError {
        return self
            .receive(on: DispatchQueue.main)
            .handleEvents(receiveCompletion: { completion in
                if case .failure(let error) = completion {
                    object[keyPath: errorPath] = error
                    if let isLoadingPath = isLoadingPath {
                        object[keyPath: isLoadingPath] = false
                    }
                }
            })
            .eraseToAnyPublisher()
    }
}

extension APIService {
    func getSnippetsForQuestion(questionId: Int) -> AnyPublisher<SnippetList, APIError> {
        return getSnippets(sourceLang: nil, targetLang: nil, storyId: nil, query: nil, level: nil)
    }

    func getSnippetsForStory(storyId: Int) -> AnyPublisher<SnippetList, APIError> {
        return getSnippets(sourceLang: nil, targetLang: nil, storyId: storyId, query: nil, level: nil)
    }
}

