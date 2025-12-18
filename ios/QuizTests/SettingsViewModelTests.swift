import Combine
import XCTest

@testable import Quiz

class SettingsViewModelTests: XCTestCase {
    var viewModel: SettingsViewModel!
    var mockAPIService: MockAPIService!

    override func setUp() {
        super.setUp()
        mockAPIService = MockAPIService()
        viewModel = SettingsViewModel(apiService: mockAPIService)
    }

    override func tearDown() {
        viewModel = nil
        mockAPIService = nil
        super.tearDown()
    }

    func testUpdateUserSuccess() {
        // Given
        let user = User(
            id: 1, username: "test", email: "test@test.com", timezone: nil,
            preferredLanguage: "en", currentLevel: "A1", aiEnabled: nil, isPaused: nil,
            wordOfDayEmailEnabled: nil, aiProvider: nil, aiModel: nil, hasApiKey: nil)
        mockAPIService.updateUserResult = .success(user)
        let userUpdate = UserUpdateRequest(
            username: "test", email: "test@test.com", preferredLanguage: "en", currentLevel: "A1")

        // When
        viewModel.saveChanges(userUpdate: userUpdate, prefs: nil)

        // Then
        // Wait a bit for async operation
        let expectation = XCTestExpectation(description: "User updated")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertNotNil(self.viewModel.user)
            XCTAssertEqual(self.viewModel.user?.username, "test")
            XCTAssertNil(self.viewModel.error)
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }

    func testUpdateUserFailure() {
        // Given
        mockAPIService.updateUserResult = .failure(.invalidResponse)
        let userUpdate = UserUpdateRequest(
            username: "test", email: "test@test.com", preferredLanguage: "en", currentLevel: "A1")

        // When
        viewModel.saveChanges(userUpdate: userUpdate, prefs: nil)

        // Then
        // Wait a bit for async operation
        let expectation = XCTestExpectation(description: "User update failed")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertNotNil(self.viewModel.error)
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }
}
