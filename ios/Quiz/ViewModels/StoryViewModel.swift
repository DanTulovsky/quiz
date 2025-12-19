import Foundation
import Combine

class StoryViewModel: ObservableObject {
    @Published var stories = [StorySummary]()
    @Published var selectedStory: StoryContent?
    @Published var currentSection: StorySectionWithQuestions?
    @Published var currentSectionIndex = 0
    @Published var snippets = [Snippet]()
    @Published var mode: StoryMode = .section
    @Published var isLoading = false
    @Published var error: APIService.APIError?

    enum StoryMode {
        case section
        case reading
    }

    var apiService: APIService
    private var cancellables = Set<AnyCancellable>()

    init(apiService: APIService = APIService.shared) {
        self.apiService = apiService
    }

    func getStories() {
        isLoading = true
        apiService.getStories()
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isLoading = false
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] storyList in
                self?.stories = storyList
            })
            .store(in: &cancellables)
    }

    func getStory(id: Int) {
        isLoading = true
        apiService.getStory(id: id)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                if case .failure(let error) = completion {
                    self?.isLoading = false
                    self?.error = error
                }
            }, receiveValue: { [weak self] storyContent in
                self?.selectedStory = storyContent
                self?.getSnippets(storyId: id)
                if !storyContent.sections.isEmpty {
                    self?.fetchSection(id: storyContent.sections[0].id)
                } else {
                    self?.isLoading = false
                }
            })
            .store(in: &cancellables)
    }

    func fetchSection(id: Int) {
        isLoading = true
        apiService.getStorySection(id: id)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isLoading = false
                if case .failure(let error) = completion { self?.error = error }
            }, receiveValue: { [weak self] section in
                self?.currentSection = section
            })
            .store(in: &cancellables)
    }

    func getSnippets(storyId: Int) {
        apiService.getSnippetsForStory(storyId: storyId)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] snippetList in
                self?.snippets = snippetList.snippets
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
