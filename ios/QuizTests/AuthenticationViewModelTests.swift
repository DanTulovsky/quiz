import Combine
import XCTest

@testable import Quiz

class AuthenticationViewModelTests: XCTestCase {
    var viewModel: AuthenticationViewModel!
    var mockAPIService: MockAPIService!

    override func setUp() {
        super.setUp()
        mockAPIService = MockAPIService()
        // AuthStatus must be provided for init
        mockAPIService.authStatusResult = .success(
            AuthStatusResponse(authenticated: false, user: nil))
        viewModel = AuthenticationViewModel(apiService: mockAPIService)
    }

    override func tearDown() {
        viewModel = nil
        mockAPIService = nil
        super.tearDown()
    }

    func testLoginSuccess() {
        // Given
        let user = User(
            id: 1, username: "test", email: "test@test.com", timezone: nil,
            preferredLanguage: "it", currentLevel: "A1", aiEnabled: nil, isPaused: nil,
            wordOfDayEmailEnabled: nil, aiProvider: nil, aiModel: nil, hasApiKey: nil)
        let loginResponse = LoginResponse(success: true, message: "OK", user: user)
        mockAPIService.loginResult = .success(loginResponse)
        let expectation = XCTestExpectation(description: "Login succeeded")

        // When
        viewModel.login()

        // Then - wait for async operation
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertTrue(self.viewModel.isAuthenticated)
            XCTAssertNil(self.viewModel.error)
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }

    func testLoginFailure() {
        // Given
        mockAPIService.loginResult = .failure(.invalidResponse)
        let expectation = XCTestExpectation(description: "Login failed")

        // When
        viewModel.login()

        // Then - wait for async operation
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertFalse(self.viewModel.isAuthenticated)
            XCTAssertNotNil(self.viewModel.error)
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }
}
