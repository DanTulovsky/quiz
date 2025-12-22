import Combine
import Foundation

protocol LanguageFetching: BaseViewModel {
    var availableLanguages: [LanguageInfo] { get set }
}

extension LanguageFetching {
    func fetchLanguages() {
        apiService.getLanguages()
            .handleErrorOnly(on: self)
            .sinkValue(on: self) { [weak self] languages in
                self?.availableLanguages = languages
            }
            .store(in: &cancellables)
    }
}
