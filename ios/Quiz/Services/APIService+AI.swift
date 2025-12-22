import Combine
import Foundation

extension APIService {

    func toggleBookmark(conversationId: String, messageId: String) -> AnyPublisher<
        BookmarkStatusResponse, APIError
    > {
        return putJSON(
            path: "ai/conversations/bookmark",
            body: ["conversation_id": conversationId, "message_id": messageId],
            responseType: BookmarkStatusResponse.self
        )
    }

    func getAIConversation(id: String) -> AnyPublisher<Conversation, APIError> {
        return get(path: "ai/conversations/\(id)", responseType: Conversation.self)
    }

    func updateAIConversationTitle(id: String, title: String) -> AnyPublisher<
        SuccessResponse, APIError
    > {
        return putJSON(
            path: "ai/conversations/\(id)",
            body: ["title": title],
            responseType: SuccessResponse.self
        )
    }

    func deleteAIConversation(id: String) -> AnyPublisher<SuccessResponse, APIError> {
        return delete(path: "ai/conversations/\(id)", responseType: SuccessResponse.self)
    }
}

