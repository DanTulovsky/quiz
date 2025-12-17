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
                        ConversationCard(conversation: conv)
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
    
    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text(conversation.title)
                    .font(.headline)
                    .lineLimit(1)
                Spacer()
                Button(action: {}) {
                    Image(systemName: "square.and.pencil")
                        .foregroundColor(.secondary)
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
    }
}

struct BookmarkedMessagesView: View {
    @StateObject private var viewModel = AIHistoryViewModel()
    
    var body: some View {
        VStack(spacing: 0) {
            // Header Stats
            HStack {
                Text("\(viewModel.bookmarks.count)")
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
                        BookmarkCard(message: msg)
                    }
                }
                .padding()
            }
        }
        .onAppear {
            viewModel.fetchBookmarks()
        }
        .navigationTitle("Bookmarked Messages")
    }
}

struct BookmarkCard: View {
    let message: ChatMessage
    
    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text(message.role.uppercased())
                    .font(.caption)
                    .bold()
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(Color.blue)
                    .foregroundColor(.white)
                    .cornerRadius(4)
                
                Text(message.createdAt, style: .date)
                    .font(.caption)
                    .foregroundColor(.secondary)
                Text(message.createdAt, style: .time)
                    .font(.caption)
                    .foregroundColor(.secondary)
                
                Spacer()
                
                Button(action: {}) {
                    Image(systemName: "bookmark.fill")
                        .foregroundColor(.red.opacity(0.7))
                }
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
            
            Text(message.content.text)
                .font(.body)
            
            HStack {
                Spacer()
                Image(systemName: "arrow.up.left.and.arrow.down.right")
                    .font(.caption)
                    .foregroundColor(.blue)
            }
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(12)
        .shadow(color: Color.black.opacity(0.05), radius: 5, x: 0, y: 2)
    }
}
