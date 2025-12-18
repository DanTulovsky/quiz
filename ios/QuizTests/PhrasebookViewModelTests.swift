import Combine
import XCTest

@testable import Quiz

class PhrasebookViewModelTests: XCTestCase {
    var viewModel: PhrasebookViewModel!

    override func setUp() {
        super.setUp()
        viewModel = PhrasebookViewModel()
    }

    override func tearDown() {
        viewModel = nil
        super.tearDown()
    }

    func testFetchCategories() {
        // When
        viewModel.fetchCategories()

        // Then
        // Should have at least the sample fallback categories if local files not found
        XCTAssertFalse(viewModel.categories.isEmpty)
    }
}
