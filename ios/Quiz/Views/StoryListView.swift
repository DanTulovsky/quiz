import SwiftUI

struct StoryListView: View {
    @StateObject private var viewModel = StoryViewModel()

    var body: some View {
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
        .onAppear {
            viewModel.getStories()
        }
        .navigationTitle("Stories")
    }
}
