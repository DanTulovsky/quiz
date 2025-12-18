import SwiftUI

struct AIConversationDetailView: View {
    let conversationId: String
    @StateObject private var viewModel = AIHistoryViewModel()

    var body: some View {
        ScrollView {
            VStack(spacing: 20) {
                if viewModel.isLoading && viewModel.selectedConversation == nil {
                    ProgressView()
                        .padding(.top, 50)
                } else if let conversation = viewModel.selectedConversation {
                    if let messages = conversation.messages {
                        ForEach(messages) { message in
                            MessageBubble(message: message)
                        }
                    } else {
                        Text("No messages in this conversation.")
                            .foregroundColor(.secondary)
                            .padding(.top, 50)
                    }
                }

                if let error = viewModel.error {
                    Text("Error: \(error.localizedDescription)")
                        .foregroundColor(.red)
                        .padding()
                }
            }
            .padding()
        }
        .navigationTitle(viewModel.selectedConversation?.title ?? "Conversation")
        .navigationBarTitleDisplayMode(.inline)
        .onAppear {
            viewModel.fetchConversation(id: conversationId)
        }
    }
}

struct MessageBubble: View {
    let message: ChatMessage

    var isUser: Bool {
        message.role == "user"
    }

    var body: some View {
        HStack {
            if isUser { Spacer() }

            VStack(alignment: isUser ? .trailing : .leading, spacing: 4) {
                Text(message.content.text)
                    .padding(12)
                    .background(isUser ? Color.blue : Color(.secondarySystemBackground))
                    .foregroundColor(isUser ? .white : .primary)
                    .cornerRadius(16)
                    .fixedSize(horizontal: false, vertical: true)

                Text(message.createdAt, style: .time)
                    .font(.caption2)
                    .foregroundColor(.secondary)
                    .padding(.horizontal, 4)
            }

            if !isUser { Spacer() }
        }
    }
}

