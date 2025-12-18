import XCTest
import Combine
@testable import LingoLearn

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
        let user = User(id: 1, username: "test", email: "test@test.com", language: .en, level: .a1)
        mockAPIService.updateUserResult = .success(user)

        // When
        viewModel.updateUser(username: "test", email: "test@test.com", language: .en, level: .a1)

        // Then
        XCTAssertNotNil(viewModel.user)
        XCTAssertEqual(viewModel.user?.username, "test")
        XCTAssertNil(viewModel.error)
    }

    func testUpdateUserFailure() {
        // Given
        mockAPIService.updateUserResult = .failure(.invalidResponse)

        // When
        viewModel.updateUser(username: "test", email: "test@test.com", language: .en, level: .a1)

        // Then
        XCTAssertNil(viewModel.user)
        XCTAssertNotNil(viewModel.error)
    }
}

extension MockAPIService {
    var updateUserResult: Result<User, APIError>?
    
    override func updateUser(request: UserUpdateRequest) -> AnyPublisher<User, APIError> {
        return updateUserResult!.publisher.eraseToAnyPublisher()
    }
}
