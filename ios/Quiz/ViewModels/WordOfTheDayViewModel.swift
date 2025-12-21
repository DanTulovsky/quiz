import Foundation
import Combine

class WordOfTheDayViewModel: BaseViewModel {
    @Published var wordOfTheDay: WordOfTheDayDisplay?
    @Published var currentDate = Date()

    init(apiService: APIService = APIService.shared) {
        super.init(apiService: apiService)
    }

    func fetchWordOfTheDay() {
        let dateStr = currentDate.iso8601String
        fetchWord(for: dateStr)
    }

    func fetchToday() {
        currentDate = Date()
        fetchWord(for: nil)
    }

    func nextDay() {
        if let next = Calendar.current.date(byAdding: .day, value: 1, to: currentDate), next <= Date() {
            currentDate = next
            fetchWordOfTheDay()
        }
    }

    func previousDay() {
        if let prev = Calendar.current.date(byAdding: .day, value: -1, to: currentDate) {
            currentDate = prev
            fetchWordOfTheDay()
        }
    }

    func selectDate(_ date: Date) {
        let today = Date()
        let calendar = Calendar.current
        let dateOnly = calendar.startOfDay(for: date)
        let todayOnly = calendar.startOfDay(for: today)

        if dateOnly <= todayOnly {
            currentDate = dateOnly
            fetchWordOfTheDay()
        }
    }

    private func fetchWord(for date: String?) {
        isLoading = true
        clearError()

        apiService.getWordOfTheDay(date: date)
            .handleLoadingAndError(on: self)
            .sink(receiveValue: { [weak self] word in
                self?.wordOfTheDay = word
            })
            .store(in: &cancellables)
    }
}
