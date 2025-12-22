import Foundation
import Combine

protocol SectionNavigable: BaseViewModel {
    associatedtype Section
    var sections: [Section] { get }
    var currentSectionIndex: Int { get set }
    func fetchSection(at index: Int)
}

extension SectionNavigable {
    var hasNextSection: Bool {
        currentSectionIndex < sections.count - 1
    }

    var hasPreviousSection: Bool {
        currentSectionIndex > 0
    }

    var currentSection: Section? {
        guard currentSectionIndex >= 0 && currentSectionIndex < sections.count else { return nil }
        return sections[currentSectionIndex]
    }

    func nextSection() -> Bool {
        guard hasNextSection else { return false }
        currentSectionIndex += 1
        fetchSection(at: currentSectionIndex)
        return true
    }

    func previousSection() -> Bool {
        guard hasPreviousSection else { return false }
        currentSectionIndex -= 1
        fetchSection(at: currentSectionIndex)
        return true
    }

    func goToSectionBeginning() -> Bool {
        guard !sections.isEmpty && currentSectionIndex != 0 else { return false }
        currentSectionIndex = 0
        fetchSection(at: 0)
        return true
    }

    func goToSectionEnd() -> Bool {
        guard !sections.isEmpty else { return false }
        let lastIndex = sections.count - 1
        guard currentSectionIndex != lastIndex else { return false }
        currentSectionIndex = lastIndex
        fetchSection(at: lastIndex)
        return true
    }
}
