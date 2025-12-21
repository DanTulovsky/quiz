import Foundation
import Combine

class StoryViewModel: BaseViewModel, SnippetLoading {
    @Published var stories = [StorySummary]()
    @Published var selectedStory: StoryContent?
    @Published var currentSection: StorySectionWithQuestions?
    @Published var currentSectionIndex = 0
    @Published var snippets = [Snippet]()
    @Published var mode: StoryMode = .section

    enum StoryMode {
        case section
        case reading
    }

    init(apiService: APIService = APIService.shared) {
        super.init(apiService: apiService)
    }

    func getStories() {
        apiService.getStories()
            .handleLoadingAndError(on: self)
            .sink(receiveValue: { [weak self] storyList in
                self?.stories = storyList
            })
            .store(in: &cancellables)
    }

    func getStory(id: Int) {
        isLoading = true
        apiService.getStory(id: id)
            .handleLoadingAndError(on: self)
            .sink(receiveValue: { [weak self] storyContent in
                guard let self = self else { return }
                self.selectedStory = storyContent
                self.loadSnippets(storyId: id)
                if !storyContent.sections.isEmpty {
                    self.fetchSection(id: storyContent.sections[0].id)
                } else {
                    self.isLoading = false
                }
            })
            .store(in: &cancellables)
    }

    func fetchSection(id: Int) {
        apiService.getStorySection(id: id)
            .handleLoadingAndError(on: self)
            .sink(receiveValue: { [weak self] section in
                self?.currentSection = section
            })
            .store(in: &cancellables)
    }

    func nextPage() {
        guard let story = selectedStory, currentSectionIndex < story.sections.count - 1 else { return }
        currentSectionIndex += 1
        fetchSection(id: story.sections[currentSectionIndex].id)
    }

    func previousPage() {
        guard let story = selectedStory, currentSectionIndex > 0 else { return }
        currentSectionIndex -= 1
        fetchSection(id: story.sections[currentSectionIndex].id)
    }

    func goToBeginning() {
        guard let story = selectedStory, !story.sections.isEmpty, currentSectionIndex != 0 else { return }
        currentSectionIndex = 0
        fetchSection(id: story.sections[0].id)
    }

    func goToEnd() {
        guard let story = selectedStory, !story.sections.isEmpty else { return }
        let lastIndex = story.sections.count - 1
        guard currentSectionIndex != lastIndex else { return }
        currentSectionIndex = lastIndex
        fetchSection(id: story.sections[lastIndex].id)
    }

    var fullStoryContent: String {
        guard let story = selectedStory else { return "" }
        return story.sections.map { $0.content }.joined(separator: "\n\n")
    }
}
