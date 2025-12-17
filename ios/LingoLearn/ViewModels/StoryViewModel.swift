import Foundation
import Combine

class StoryViewModel: ObservableObject {
    @Published var stories = [Story]()
    @Published var selectedStory: StoryContent?
    @Published var error: APIService.APIError?

    var apiService: APIService
    private var cancellables = Set<AnyCancellable>()

    init(apiService: APIService = APIService.shared) {
        self.apiService = apiService
    }

    func getStories() {
        apiService.getStories(language: nil, level: nil)
            .sink(receiveCompletion: { completion in
                switch completion {
                case .failure(let error):
                    self.error = error
                case .finished:
                    break
                }
            }, receiveValue: { storyList in
                self.stories = storyList.stories
            })
            .store(in: &cancellables)
    }

    func getStory(id: Int) {
        apiService.getStory(id: id)
            .sink(receiveCompletion: { completion in
                switch completion {
                case .failure(let error):
                    self.error = error
                case .finished:
                    break
                }
            }, receiveValue: { storyContent in
                self.selectedStory = storyContent
            })
            .store(in: &cancellables)
    }
}
