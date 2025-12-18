import SwiftUI

struct AIConversationListView: View {
    @StateObject private var viewModel = AIHistoryViewModel()

    var body: some View {
        VStack(spacing: 0) {
            // Header Stats
            HStack {
                Text("\(viewModel.conversations.count)")
                    .font(.caption)
                    .bold()
                    .padding(6)
                    .background(Color.blue.opacity(0.1))
                    .foregroundColor(.blue)
                    .clipShape(RoundedRectangle(cornerRadius: 6))

                Spacer()
            }
            .padding(.horizontal)
            .padding(.top)

            // Search Bar
            HStack {
                Image(systemName: "magnifyingglass")
                    .foregroundColor(.secondary)
                TextField("Search conversations...", text: .constant(""))
            }
            .padding(10)
            .background(Color(.secondarySystemBackground))
            .cornerRadius(10)
            .padding()

            // Conversations List
            ScrollView {
                LazyVStack(spacing: 16) {
                    ForEach(viewModel.conversations) { conv in
                        NavigationLink(destination: AIConversationDetailView(conversationId: conv.id)) {
                            ConversationCard(conversation: conv, viewModel: viewModel)
                        }
                        .buttonStyle(PlainButtonStyle())
                    }
                }
                .padding()
            }
        }
        .onAppear {
            viewModel.fetchConversations()
        }
        .navigationTitle("Saved Conversations")
    }
}

struct ConversationCard: View {
    let conversation: Conversation
    @ObservedObject var viewModel: AIHistoryViewModel
    @State private var showingEditTitle = false
    @State private var newTitle = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text(conversation.title)
                    .font(.headline)
                    .lineLimit(1)
                Spacer()

                Menu {
                    NavigationLink(destination: AIConversationDetailView(conversationId: conversation.id)) {
                        Label("View", systemImage: "eye")
                    }

                    Button(action: {
                        newTitle = conversation.title
                        showingEditTitle = true
                    }) {
                        Label("Edit Title", systemImage: "pencil")
                    }

                    Button(role: .destructive, action: {
                        viewModel.deleteConversation(id: conversation.id)
                    }) {
                        Label("Delete", systemImage: "trash")
                    }
                } label: {
                    Image(systemName: "ellipsis.circle")
                        .font(.title3)
                        .foregroundColor(.secondary)
                        .padding(4)
                }
            }

            HStack(spacing: 10) {
                HStack(spacing: 4) {
                    Image(systemName: "calendar")
                    Text(conversation.createdAt, style: .date)
                    Text(conversation.createdAt, style: .time)
                }
                .font(.caption)
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(Color.blue.opacity(0.1))
                .foregroundColor(.blue)
                .cornerRadius(6)

                Text("\(conversation.messageCount ?? 0) MSGS")
                    .font(.caption)
                    .bold()
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color.green.opacity(0.1))
                    .foregroundColor(.green)
                    .cornerRadius(6)
            }
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(12)
        .shadow(color: Color.black.opacity(0.05), radius: 5, x: 0, y: 2)
        .alert("Edit Title", isPresented: $showingEditTitle) {
            TextField("Title", text: $newTitle)
            Button("Cancel", role: .cancel) { }
            Button("Save") {
                viewModel.updateTitle(id: conversation.id, newTitle: newTitle)
            }
        } message: {
            Text("Enter a new title for this conversation.")
        }
    }
}

struct BookmarkedMessagesView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel = AIHistoryViewModel()

    var body: some View {
        VStack(spacing: 0) {
            // Header with Title and Count
            HStack(spacing: 8) {
                Text("Bookmarked Messages")
                    .font(.system(size: 28, weight: .bold))
                    .lineLimit(1)
                    .minimumScaleFactor(0.5)

                Text("\(viewModel.bookmarks.count)")
                    .font(.system(size: 18))
                    .bold()
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color.blue.opacity(0.1))
                    .foregroundColor(.blue)
                    .clipShape(RoundedRectangle(cornerRadius: 6))

                Spacer()
            }
            .padding(.horizontal)
            .padding(.top)
            .padding(.bottom, 8)

            // Search Bar
            HStack {
                Image(systemName: "magnifyingglass")
                    .foregroundColor(.secondary)
                TextField("Search bookmarks...", text: .constant(""))
            }
            .padding(10)
            .background(Color(.secondarySystemBackground))
            .cornerRadius(10)
            .padding()

            // Bookmarks List
            ScrollView {
                LazyVStack(spacing: 16) {
                    ForEach(viewModel.bookmarks) { msg in
                        BookmarkCard(message: msg, viewModel: viewModel)
                    }
                }
                .padding()
            }
        }
        .navigationBarTitleDisplayMode(.inline)
        .navigationBarBackButtonHidden(true)
        .toolbar {
            ToolbarItem(placement: .navigationBarLeading) {
                Button(action: { dismiss() }) {
                    HStack(spacing: 4) {
                        Image(systemName: "chevron.left")
                            .font(.system(size: 17, weight: .semibold))
                        Text("Back")
                            .font(.system(size: 17))
                    }
                    .foregroundColor(.blue)
                }
            }
        }
        .onAppear {
            viewModel.fetchBookmarks()
        }
    }
}

struct BookmarkCard: View {
    let message: ChatMessage
    @ObservedObject var viewModel: AIHistoryViewModel
    @State private var isExpanded = false

    private let maxPreviewLength = 100

    var body: some View {
        Button(action: {
            withAnimation {
                isExpanded.toggle()
            }
        }) {
            VStack(alignment: .leading, spacing: 12) {
                HStack {
                    Text(message.role.uppercased())
                        .font(.caption)
                        .bold()
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(message.role == "user" ? Color.blue : Color.green)
                        .foregroundColor(.white)
                        .cornerRadius(6)

                    Text(message.createdAt, style: .date)
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Text(message.createdAt, style: .time)
                        .font(.caption)
                        .foregroundColor(.secondary)

                    Spacer()

                    Button(action: {
                        viewModel.toggleBookmark(conversationId: message.conversationId, messageId: message.id)
                    }) {
                        Image(systemName: "bookmark.fill")
                            .foregroundColor(.red.opacity(0.7))
                            .font(.system(size: 16))
                    }
                    .buttonStyle(.plain)
                }

                if let title = message.conversationTitle {
                    Text(title.uppercased())
                        .font(.system(size: 10, weight: .bold))
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(Color.gray.opacity(0.1))
                        .foregroundColor(.secondary)
                        .cornerRadius(4)
                }

                if isExpanded {
                    Text(message.content.text)
                        .font(.body)
                        .foregroundColor(.primary)
                        .fixedSize(horizontal: false, vertical: true)
                } else {
                    let preview = message.content.text.prefix(maxPreviewLength)
                    Text(preview + (message.content.text.count > maxPreviewLength ? "..." : ""))
                        .font(.body)
                        .foregroundColor(.primary)
                        .lineLimit(2)
                }

                HStack {
                    Spacer()
                    Image(systemName: isExpanded ? "arrow.down.right.and.arrow.up.left" : "arrow.up.left.and.arrow.down.right")
                        .font(.caption)
                        .foregroundColor(.blue)
                }
            }
            .padding()
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Color(.systemBackground))
            .cornerRadius(12)
            .overlay(RoundedRectangle(cornerRadius: 12).stroke(Color.gray.opacity(0.2), lineWidth: 1))
        }
        .buttonStyle(.plain)
    }
}
