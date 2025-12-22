import SwiftUI

struct StoryListView: View {
    @StateObject private var viewModel = StoryViewModel()

    var body: some View {
        Group {
            if viewModel.isLoading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if viewModel.stories.isEmpty {
                EmptyStateView(
                    icon: "book",
                    title: "No Stories Available",
                    message: "Check back later for new stories to practice your reading skills."
                )
            } else {
                List(viewModel.stories, id: \.id) { story in
                    NavigationLink(destination: StoryDetailView(storyId: story.id)) {
                        VStack(alignment: .leading) {
                            Text(story.title)
                                .font(AppTheme.Typography.headingFont)
                            Text("Language: \(story.language)")
                                .font(AppTheme.Typography.subheadlineFont)
                        }
                    }
                }
            }
        }
        .onAppear {
            viewModel.fetchItems()
        }
        .navigationTitle("Stories")
    }
}
