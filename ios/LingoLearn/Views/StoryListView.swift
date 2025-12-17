import SwiftUI

struct StoryListView: View {
    @StateObject private var viewModel = StoryViewModel()

    var body: some View {
        NavigationView {
            List(viewModel.stories, id: \.id) { story in
                NavigationLink(destination: StoryDetailView(storyId: story.id)) {
                    VStack(alignment: .leading) {
                        Text(story.title)
                            .font(.headline)
                        Text("Level: \(story.level.rawValue)")
                            .font(.subheadline)
                    }
                }
            }
            .onAppear {
                viewModel.getStories()
            }
            .navigationTitle("Stories")
        }
    }
}
