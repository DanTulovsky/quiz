import Combine
import Foundation
import SwiftUI

class QuestionViewState: ObservableObject {
    @Published var reportReason = ""
    @Published var selectedConfidence: Int?
    @Published var selectedText: String?
    @Published var showTranslationPopup = false
    @Published var translationSentence: String?
    @Published var showingSnippet: Snippet?
}
