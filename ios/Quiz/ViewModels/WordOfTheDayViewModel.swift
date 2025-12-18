import Foundation
import Combine

class WordOfTheDayViewModel: ObservableObject {
    @Published var wordOfTheDay: WordOfTheDayDisplay?
    @Published var currentDate = Date()
    @Published var isLoading = false
    @Published var error: APIService.APIError?
    
    private var apiService: APIService
    private var cancellables = Set<AnyCancellable>()
    
    private var dateFormatter: DateFormatter {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter
    }
    
    init(apiService: APIService = APIService.shared) {
        self.apiService = apiService
    }
    
    func fetchWordOfTheDay() {
        let dateStr = dateFormatter.string(from: currentDate)
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
    
    private func fetchWord(for date: String?) {
        isLoading = true
        error = nil
        
        apiService.getWordOfTheDay(date: date)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                self?.isLoading = false
                if case .failure(let error) = completion {
                    self?.error = error
                }
            }, receiveValue: { [weak self] word in
                self?.wordOfTheDay = word
            })
            .store(in: &cancellables)
    }
}
