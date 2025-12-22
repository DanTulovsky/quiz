import Foundation

protocol LanguageCaching: AnyObject {
    var availableLanguages: [LanguageInfo] { get set }
    var languageCacheByCode: [String: LanguageInfo] { get set }
    var languageCacheByName: [String: LanguageInfo] { get set }
    func updateLanguageCache()
}

extension LanguageCaching {
    func findLanguage(byCodeOrName codeOrName: String) -> LanguageInfo? {
        let lowercased = codeOrName.lowercased()
        return languageCacheByCode[lowercased] ?? languageCacheByName[lowercased]
    }

    func updateLanguageCache() {
        languageCacheByCode.removeAll()
        languageCacheByName.removeAll()
        for lang in availableLanguages {
            languageCacheByCode[lang.code.lowercased()] = lang
            languageCacheByName[lang.name.lowercased()] = lang
        }
    }
}
