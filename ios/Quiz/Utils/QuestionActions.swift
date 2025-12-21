import Foundation
import Combine

protocol QuestionActions: BaseViewModel {
    var isSubmittingAction: Bool { get set }
    var showReportModal: Bool { get set }
    var showMarkKnownModal: Bool { get set }
    var isReported: Bool { get set }
}

extension QuestionActions {
    func reportQuestion(id: Int, reason: String?) {
        isSubmittingAction = true
        let request = ReportQuestionRequest(reportReason: reason)
        apiService.reportQuestion(id: id, request: request)
            .handleErrorOnly(on: self)
            .sink(receiveCompletion: { [weak self] completion in
                guard let self = self else { return }
                self.isSubmittingAction = false
                if case .finished = completion {
                    self.isReported = true
                    self.showReportModal = false
                }
            }, receiveValue: { _ in })
            .store(in: &cancellables)
    }

    func markQuestionKnown(id: Int, confidence: Int) {
        isSubmittingAction = true
        let request = MarkQuestionKnownRequest(confidenceLevel: confidence)
        apiService.markQuestionKnown(id: id, request: request)
            .handleErrorOnly(on: self)
            .sink(receiveCompletion: { [weak self] completion in
                guard let self = self else { return }
                self.isSubmittingAction = false
                if case .finished = completion {
                    self.showMarkKnownModal = false
                }
            }, receiveValue: { _ in })
            .store(in: &cancellables)
    }
}

