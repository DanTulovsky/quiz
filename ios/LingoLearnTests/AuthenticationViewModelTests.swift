import XCTest
import Combine
@testable import LingoLearn

class AuthenticationViewModelTests: XCTestCase {
    var viewModel: AuthenticationViewModel!
    var mockAPIService: MockAPIService!

    override func setUp() {
        super.setUp()
        mockAPIService = MockAPIService()
        viewModel = AuthenticationViewModel(apiService: mockAPIService)
    }

    override func tearDown() {
        viewModel = nil
        mockAPIService = nil
        super.tearDown()
    }

    func testLoginSuccess() {
        // Given
        let loginResponse = LoginResponse(token: "test_token", user: User(id: 1, username: "test", email: "test@test.com", language: .en, level: .a1))
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

class MockAPIService: APIService {
    var loginResult: Result<LoginResponse, APIError>?
    
    override func login(request: LoginRequest) -> AnyPublisher<LoginResponse, APIError> {
        return loginResult!.publisher.eraseToAnyPublisher()
    }
}

extension Result {
    var publisher: AnyPublisher<Success, Failure> {
        switch self {
        case .success(let value):
            return Just(value)
                .setFailureType(to: Failure.self)
                .eraseToAnyPublisher()
        case .failure(let error):
            return Fail(error: error)
                .eraseToAnyPublisher()
        }
    }
}