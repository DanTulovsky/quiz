import Foundation
import Combine

class StoryViewModel: BaseViewModel, SnippetLoading, Refreshable, ListFetching, SectionNavigable {
    typealias Item = StorySummary
    typealias Section = StorySection

    @Published var stories = [StorySummary]()

    var items: [StorySummary] {
        get { stories }
        set { stories = newValue }
    }
    @Published var selectedStory: StoryContent?
    @Published var currentSection: StorySectionWithQuestions?
    @Published var currentSectionIndex = 0
    @Published var snippets = [Snippet]()
    @Published var mode: StoryMode = .section

    enum StoryMode {
        case section
        case reading
    }

    var sections: [StorySection] {
        selectedStory?.sections ?? []
    }

    override init(apiService: APIServiceProtocol = APIService.shared) {
        super.init(apiService: apiService)
    }

    func fetchItemsPublisher() -> AnyPublisher<[StorySummary], APIService.APIError> {
        return apiService.getStories()
    }

    func getStory(id: Int) {
        apiService.getStory(id: id)
            .handleLoadingAndError(on: self)
            .sinkValue(on: self) { [weak self] storyContent in
                guard let self = self else { return }
                self.selectedStory = storyContent
                self.loadSnippets(storyId: id)
                if !storyContent.sections.isEmpty {
                    self.currentSectionIndex = 0
                    self.fetchSection(at: 0)
                } else {
                    self.isLoading = false
                }
            }
            .store(in: &cancellables)
    }

    func fetchSection(at index: Int) {
        guard let story = selectedStory, index >= 0 && index < story.sections.count else { return }
        fetchSection(id: story.sections[index].id)
    }

    private func fetchSection(id: Int) {
        apiService.getStorySection(id: id)
            .handleLoadingAndError(on: self)
            .sinkValue(on: self) { [weak self] section in
                self?.currentSection = section
            }
            .store(in: &cancellables)
    }

    func nextPage() {
        _ = nextSection()
    }

    func previousPage() {
        _ = previousSection()
    }

    func goToBeginning() {
        _ = goToSectionBeginning()
    }

    func goToEnd() {
        _ = goToSectionEnd()
    }

    var fullStoryContent: String {
        guard let story = selectedStory else { return "" }
        return story.sections.map { $0.content }.joined(separator: "\n\n")
    }

    func refreshData() {
        fetchItems()
    }
}
