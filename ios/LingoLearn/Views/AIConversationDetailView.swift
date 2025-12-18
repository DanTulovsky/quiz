import SwiftUI

struct AIConversationDetailView: View {
    let conversationId: String
    @StateObject private var viewModel = AIHistoryViewModel()

    var body: some View {
        ScrollView {
            VStack(spacing: 12) {
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
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(isUser ? "YOU" : "AI")
                    .font(.caption)
                    .fontWeight(.bold)
                    .foregroundColor(.white)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                    .background(isUser ? Color.blue : Color.green)
                    .cornerRadius(8)

                Text(message.createdAt, style: .date)
                    .font(.subheadline)
                    .foregroundColor(.secondary)

                Text(message.createdAt, style: .time)
                    .font(.subheadline)
                    .foregroundColor(.secondary)

                Spacer()

                if !isUser {
                    Button(action: {
                        // TODO: Implement bookmark functionality
                    }) {
                        Label("Bookmark", systemImage: "bookmark")
                            .font(.caption)
                            .foregroundColor(.blue)
                    }
                }
            }

            Text(message.content.text)
                .font(.body)
                .foregroundColor(.primary)
                .fixedSize(horizontal: false, vertical: true)
        }
        .padding()
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(isUser ? Color.blue.opacity(0.1) : Color(.systemBackground))
        .cornerRadius(12)
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .stroke(Color.gray.opacity(0.2), lineWidth: 1)
        )
    }
}

