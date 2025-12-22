import Foundation
import Combine

protocol Filterable: BaseViewModel {
    func performFilter()
}

extension Filterable {
    func setupFilterDebounce<P1: Publisher, P2: Publisher, P3: Publisher>(
        _ publisher1: P1,
        _ publisher2: P2,
        _ publisher3: P3,
        delay: TimeInterval = 0.1
    ) -> AnyCancellable where P1.Failure == P2.Failure, P2.Failure == P3.Failure, P1.Failure == Never {
        return Publishers.CombineLatest3(publisher1, publisher2, publisher3)
            .dropFirst()
            .debounce(for: .milliseconds(Int(delay * 1000)), scheduler: RunLoop.main)
            .sink { [weak self] _, _, _ in
                self?.performFilter()
            }
    }

    func setupFilterDebounce<P1: Publisher, P2: Publisher>(
        _ publisher1: P1,
        _ publisher2: P2,
        delay: TimeInterval = 0.1
    ) -> AnyCancellable where P1.Failure == P2.Failure, P1.Failure == Never {
        return Publishers.CombineLatest(publisher1, publisher2)
            .dropFirst()
            .debounce(for: .milliseconds(Int(delay * 1000)), scheduler: RunLoop.main)
            .sink { [weak self] _, _ in
                self?.performFilter()
            }
    }
}

