import Combine
import Foundation

extension APIService {

    func createSnippet(request: CreateSnippetRequest) -> AnyPublisher<Snippet, APIError> {
        return post(path: "snippets", body: request, responseType: Snippet.self)
    }

    func updateSnippet(id: Int, request: UpdateSnippetRequest) -> AnyPublisher<Snippet, APIError> {
        return put(path: "snippets/\(id)", body: request, responseType: Snippet.self)
    }

    func deleteSnippet(id: Int) -> AnyPublisher<Void, APIError> {
        return deleteVoid(path: "snippets/\(id)")
    }
}

