import SwiftUI

struct AIConversationListView: View {
    @StateObject private var viewModel = AIHistoryViewModel()

    var body: some View {
        VStack(spacing: 0) {
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
                if viewModel.conversations.isEmpty && !viewModel.isLoading {
                    EmptyStateView(
                        icon: "bubble.left.and.bubble.right",
                        title: "No Conversations Yet",
                        message: "Start a conversation with the AI tutor to get personalized help with your language learning."
                    )
                    .padding()
                } else {
                    LazyVStack(spacing: 16) {
                        ForEach(viewModel.conversations, id: \.id) { conv in
                            NavigationLink(destination: AIConversationDetailView(conversationId: conv.id)) {
                                ConversationCard(conversationId: conv.id, viewModel: viewModel)
                                    .id("\(conv.id)-\(conv.title)")
                            }
                            .buttonStyle(PlainButtonStyle())
                        }
                    }
                    .padding()
                }
            }
        }
        .onAppear {
            viewModel.fetchConversations()
        }
        .toolbar {
            ToolbarItem(placement: .principal) {
                HStack(spacing: 8) {
                    Text("Saved Conversations")
                        .font(.headline)
                        .lineLimit(1)
                        .minimumScaleFactor(0.8)

                    Text("\(viewModel.conversations.count)")
                        .font(.caption)
                        .bold()
                        .padding(.horizontal, 6)
                        .padding(.vertical, 4)
                        .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                        .foregroundColor(AppTheme.Colors.primaryBlue)
                        .clipShape(RoundedRectangle(cornerRadius: 6))
                }
            }
        }
        .navigationBarTitleDisplayMode(.inline)
    }
}

struct ConversationCard: View {
    let conversationId: String
    @ObservedObject var viewModel: AIHistoryViewModel
    @State private var showingEditTitle = false
    @State private var newTitle = ""

    // Get the current conversation from viewModel to ensure we always show the latest data
    private var conversation: Conversation? {
        viewModel.conversations.first(where: { $0.id == conversationId })
    }

    var body: some View {
        if let conversation = conversation {
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
                    .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                    .foregroundColor(AppTheme.Colors.primaryBlue)
                    .cornerRadius(6)

                    Text("\(conversation.messageCount ?? 0) MSGS")
                        .font(.caption)
                        .bold()
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(AppTheme.Colors.successGreen.opacity(0.1))
                        .foregroundColor(AppTheme.Colors.successGreen)
                        .cornerRadius(6)
                }
            }
            .appInnerCard()
            .alert("Edit Title", isPresented: $showingEditTitle) {
                TextField("Title", text: $newTitle)
                Button("Cancel", role: .cancel) { }
                Button("Save") {
                    viewModel.updateTitle(id: conversation.id, newTitle: newTitle)
                    showingEditTitle = false
                }
            } message: {
                Text("Enter a new title for this conversation.")
            }
        } else {
            EmptyView()
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
                    .scaledFont(size: 28, weight: .bold)
                    .lineLimit(1)
                    .minimumScaleFactor(0.5)

                Text("\(viewModel.bookmarks.count)")
                    .scaledFont(size: 18)
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
                            .scaledFont(size: 17, weight: .semibold)
                        Text("Back")
                            .scaledFont(size: 17)
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
    @State private var showDeleteConfirmation = false

    private let maxPreviewLength = 100

    var body: some View {
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
                    showDeleteConfirmation = true
                }) {
                    Image(systemName: "bookmark.slash.fill")
                        .font(.caption)
                        .foregroundColor(.red)
                        .padding(8)
                        .background(Color.red.opacity(0.1))
                        .clipShape(Circle())
                }
                .buttonStyle(.plain)
            }

            if let title = message.conversationTitle {
                Text(title.uppercased())
                    .scaledFont(size: 10, weight: .bold)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color.gray.opacity(0.1))
                    .foregroundColor(.secondary)
                    .cornerRadius(4)
            }

            Button(action: {
                withAnimation {
                    isExpanded.toggle()
                }
            }) {
                VStack(alignment: .leading, spacing: 8) {
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
            }
            .buttonStyle(.plain)
        }
        .padding()
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(.systemBackground))
        .cornerRadius(12)
        .overlay(RoundedRectangle(cornerRadius: 12).stroke(Color.gray.opacity(0.2), lineWidth: 1))
        .alert("Remove Bookmark", isPresented: $showDeleteConfirmation) {
            Button("Cancel", role: .cancel) {}
            Button("Remove", role: .destructive) {
                viewModel.toggleBookmark(conversationId: message.conversationId, messageId: message.id)
            }
        } message: {
            Text("Are you sure you want to remove this bookmark?")
        }
    }
}
