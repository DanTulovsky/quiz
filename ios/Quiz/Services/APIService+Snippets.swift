import Combine
import Foundation

extension APIService {
    func getSnippets(
        sourceLang: String?, targetLang: String?, storyId: Int? = nil, query: String? = nil,
        level: String? = nil
    )
    -> AnyPublisher<SnippetList, APIError> {
        var params = QueryParameters()
        params.add("source_lang", value: sourceLang)
        params.add("target_lang", value: targetLang)
        params.add("story_id", value: storyId)
        params.add("q", value: query)
        params.add("level", value: level)
        return get(
            path: "snippets", queryItems: params.build(),
            responseType: SnippetList.self)
    }

    func getSnippetsByQuestion(questionId: Int) -> AnyPublisher<SnippetList, APIError> {
        return get(
            path: "snippets/by-question/\(questionId)", queryItems: nil,
            responseType: SnippetList.self)
    }

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
