import Foundation
import Combine

protocol LevelFetching: BaseViewModel {
    var availableLevels: [String] { get set }
    var levelDescriptions: [String: String] { get set }
}

extension LevelFetching {
    func fetchLevels(language: String?) {
        apiService.getLevels(language: language)
            .handleErrorOnly(on: self)
            .sinkValue(on: self) { [weak self] response in
                self?.availableLevels = response.levels
                self?.levelDescriptions = response.levelDescriptions
            }
            .store(in: &cancellables)
    }
}
