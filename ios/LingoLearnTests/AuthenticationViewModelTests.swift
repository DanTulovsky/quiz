import XCTest
import Combine
@testable import LingoLearn

class AuthenticationViewModelTests: XCTestCase {
    var viewModel: AuthenticationViewModel!
    var mockAPIService: MockAPIService!

    override func setUp() {
        super.setUp()
        mockAPIService = MockAPIService()
        // AuthStatus must be provided for init
        mockAPIService.authStatusResult = .success(AuthStatusResponse(authenticated: false, user: nil))
        viewModel = AuthenticationViewModel(apiService: mockAPIService)
    }

    override func tearDown() {
        viewModel = nil
        mockAPIService = nil
        super.tearDown()
    }

    func testLoginSuccess() {
        // Given
        let user = User(id: 1, username: "test", email: "test@test.com", preferredLanguage: "it", currentLevel: "A1")
        let loginResponse = LoginResponse(success: true, message: "OK", user: user)
        mockAPIService.loginResult = .success(loginResponse)

        // When
        viewModel.login()

        // Then
        XCTAssertTrue(viewModel.isAuthenticated)
        XCTAssertNil(viewModel.error)
    }

    func testLoginFailure() {
        // Given
        mockAPIService.loginResult = .failure(.invalidResponse)

        // When
        viewModel.login()

        // Then
        XCTAssertFalse(viewModel.isAuthenticated)
        XCTAssertNotNil(viewModel.error)
    }
}
