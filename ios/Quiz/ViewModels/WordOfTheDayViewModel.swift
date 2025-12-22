import Foundation
import Combine

class WordOfTheDayViewModel: BaseViewModel, Refreshable, DateNavigable {
    @Published var wordOfTheDay: WordOfTheDayDisplay?
    @Published var currentDate = Date()

    override init(apiService: APIServiceProtocol = APIService.shared) {
        super.init(apiService: apiService)
    }

    func fetchData(for date: String?) {
        apiService.getWordOfTheDay(date: date)
            .handleLoadingAndError(on: self)
            .sinkValue(on: self) { [weak self] word in
                self?.wordOfTheDay = word
            }
            .store(in: &cancellables)
    }

    func fetchWordOfTheDay() {
        fetchData(for: currentDate.iso8601String)
    }

    func refreshData() {
        fetchWordOfTheDay()
    }
}
