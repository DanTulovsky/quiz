import Combine
import XCTest

@testable import Quiz

class VerbViewModelTests: XCTestCase {
    var viewModel: VerbViewModel!
    var mockAPIService: MockAPIService!

    override func setUp() {
        super.setUp()
        mockAPIService = MockAPIService()
        viewModel = VerbViewModel(apiService: mockAPIService)
    }

    override func tearDown() {
        viewModel = nil
        mockAPIService = nil
        super.tearDown()
    }

    func testFetchVerbsSuccess() {
        // Given
        let verb = VerbConjugationSummary(
            infinitive: "andare", infinitiveEn: "to go", slug: nil, category: "test")
        let data = VerbConjugationsData(language: "it", languageName: "Italian", verbs: [verb])
        mockAPIService.getVerbConjugationsResult = .success(data)
        let expectation = XCTestExpectation(description: "Verbs fetched")

        // When
        viewModel.fetchVerbs(language: "it")

        // Then - wait for async operation
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertEqual(self.viewModel.verbs.count, 1)
            XCTAssertEqual(self.viewModel.verbs.first?.infinitive, "andare")
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }
}
