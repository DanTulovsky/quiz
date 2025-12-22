import Foundation
import Combine

protocol DateNavigable: BaseViewModel {
    var currentDate: Date { get set }
    func fetchData(for date: String?)
}

extension DateNavigable {
    func nextDay(maxDate: Date = Date()) {
        if let next = Calendar.current.date(byAdding: .day, value: 1, to: currentDate), next <= maxDate {
            currentDate = next
            fetchData(for: currentDate.iso8601String)
        }
    }

    func previousDay() {
        if let prev = Calendar.current.date(byAdding: .day, value: -1, to: currentDate) {
            currentDate = prev
            fetchData(for: currentDate.iso8601String)
        }
    }

    func selectDate(_ date: Date, maxDate: Date = Date()) {
        let calendar = Calendar.current
        let dateOnly = calendar.startOfDay(for: date)
        let maxDateOnly = calendar.startOfDay(for: maxDate)

        if dateOnly <= maxDateOnly {
            currentDate = dateOnly
            fetchData(for: currentDate.iso8601String)
        }
    }

    func fetchToday() {
        currentDate = Date()
        fetchData(for: nil)
    }
}
