import SwiftUI

struct StoryDetailView: View {
    @StateObject private var viewModel = StoryViewModel()
    let storyId: Int

    var body: some View {
        VStack {
            if let story = viewModel.selectedStory {
                Text(story.title)
                    .font(.largeTitle)
                    .padding()
                ScrollView {
                    ForEach(story.sections, id: \.id) { section in
                        Text(section.content)
                            .padding()
                    }
                }
            } else {
                Text("Loading Story...")
            }
        }
        .onAppear {
            viewModel.getStory(id: storyId)
        }
    }
}
