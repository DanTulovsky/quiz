import Foundation

extension Array where Element == LanguageInfo {
    func find(byCodeOrName codeOrName: String) -> LanguageInfo? {
        let lowercased = codeOrName.lowercased()
        return first(where: {
            $0.name.lowercased() == lowercased || $0.code.lowercased() == lowercased
        })
    }
}


