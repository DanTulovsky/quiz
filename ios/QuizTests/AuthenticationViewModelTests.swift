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
        // Cancel any pending operations before deallocating
        viewModel?.cancelAllRequests()
        // Small delay to allow any async operations to complete
        let expectation = XCTestExpectation(description: "Cleanup")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.15) {
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 0.2)

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

    func testGoogleCallbackRaceConditionPrevention() {
        // Given - simulate multiple rapid callbacks with the same code
        let user = User(
            id: 1, username: "test", email: "test@test.com", timezone: nil,
            preferredLanguage: "it", currentLevel: "A1", aiEnabled: nil, isPaused: nil,
            wordOfDayEmailEnabled: nil, aiProvider: nil, aiModel: nil, hasApiKey: nil)
        let callbackResponse = LoginResponse(success: true, message: "OK", user: user)
        mockAPIService.handleGoogleCallbackResult = .success(callbackResponse)
        mockAPIService.authStatusResult = .success(
            AuthStatusResponse(authenticated: true, user: user))

        let code = "test-auth-code-123"
        let state = "test-state"

        // When - simulate concurrent callbacks
        let expectation = XCTestExpectation(description: "Only one callback processed")

        // Simulate multiple threads calling handleGoogleCallback simultaneously
        DispatchQueue.concurrentPerform(iterations: 5) { _ in
            self.viewModel.handleGoogleCallback(code: code, state: state)
        }

        // Then - wait for async operation
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
            // Should only process callback once despite multiple calls
            XCTAssertTrue(self.viewModel.isAuthenticated, "Should be authenticated")
            XCTAssertNotNil(self.viewModel.user, "Should have user")
            // Verify processedCodes prevents duplicate processing
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }

    func testGoogleCallbackIgnoresDuplicateCodes() {
        // Given
        let user = User(
            id: 1, username: "test", email: "test@test.com", timezone: nil,
            preferredLanguage: "it", currentLevel: "A1", aiEnabled: nil, isPaused: nil,
            wordOfDayEmailEnabled: nil, aiProvider: nil, aiModel: nil, hasApiKey: nil)
        let callbackResponse = LoginResponse(success: true, message: "OK", user: user)
        mockAPIService.handleGoogleCallbackResult = .success(callbackResponse)
        mockAPIService.authStatusResult = .success(
            AuthStatusResponse(authenticated: true, user: user))

        let code = "duplicate-code-456"
        let state = "test-state"

        // When - call callback twice with same code
        viewModel.handleGoogleCallback(code: code, state: state)

        // Wait a bit for first callback to process
        let firstExpectation = XCTestExpectation(description: "First callback processed")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            // Try to process same code again
            self.viewModel.handleGoogleCallback(code: code, state: state)
            firstExpectation.fulfill()
        }
        wait(for: [firstExpectation], timeout: 1.0)

        // Then - second call should be ignored
        let finalExpectation = XCTestExpectation(description: "Duplicate callback ignored")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            XCTAssertTrue(self.viewModel.isAuthenticated, "Should be authenticated")
            // The second callback should have been ignored due to processedCodes
            finalExpectation.fulfill()
        }
        wait(for: [finalExpectation], timeout: 1.0)
    }
}
