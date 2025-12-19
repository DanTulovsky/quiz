import Security
import XCTest

@testable import Quiz

class KeychainServiceTests: XCTestCase {
    var keychainService: KeychainService!

    override func setUp() {
        super.setUp()
        keychainService = KeychainService.shared
        // Clean up any existing tokens before each test
        keychainService.deleteToken()
    }

    override func tearDown() {
        keychainService.deleteToken()
        super.tearDown()
    }

    func testSaveToken() {
        // Given
        let token = "test-token-123"

        // When
        let success = keychainService.save(token: token)

        // Then
        XCTAssertTrue(success, "Saving token should succeed")
        XCTAssertEqual(keychainService.loadToken(), token, "Loaded token should match saved token")
    }

    func testLoadTokenWhenEmpty() {
        // Given - no token saved

        // When
        let token = keychainService.loadToken()

        // Then
        XCTAssertNil(token, "Loading token when empty should return nil")
    }

    func testLoadTokenAfterSave() {
        // Given
        let token = "test-token-456"
        let saveSuccess = keychainService.save(token: token)
        XCTAssertTrue(saveSuccess, "Saving token should succeed")

        // When
        let loadedToken = keychainService.loadToken()

        // Then
        XCTAssertEqual(loadedToken, token, "Loaded token should match saved token")
    }

    func testDeleteToken() {
        // Given
        let token = "test-token-789"
        let saveSuccess = keychainService.save(token: token)
        XCTAssertTrue(saveSuccess, "Saving token should succeed")
        XCTAssertNotNil(keychainService.loadToken(), "Token should exist before deletion")

        // When
        let deleteSuccess = keychainService.deleteToken()

        // Then
        XCTAssertTrue(deleteSuccess, "Deleting token should succeed")
        XCTAssertNil(keychainService.loadToken(), "Token should be nil after deletion")
    }

    func testDeleteTokenWhenEmpty() {
        // Given - no token exists

        // When
        let success = keychainService.deleteToken()

        // Then
        XCTAssertTrue(success, "Deleting non-existent token should still return true")
    }

    func testSaveTokenOverwritesExisting() {
        // Given
        let firstToken = "first-token"
        let secondToken = "second-token"
        let firstSaveSuccess = keychainService.save(token: firstToken)
        XCTAssertTrue(firstSaveSuccess, "Saving first token should succeed")

        // When
        let secondSaveSuccess = keychainService.save(token: secondToken)
        XCTAssertTrue(secondSaveSuccess, "Saving second token should succeed")

        // Then
        XCTAssertEqual(
            keychainService.loadToken(), secondToken, "Second token should overwrite first")
        XCTAssertNotEqual(
            keychainService.loadToken(), firstToken, "First token should no longer exist")
    }

    func testTokenPersistence() {
        // Given
        let token = "persistent-token"
        let saveSuccess = keychainService.save(token: token)
        XCTAssertTrue(saveSuccess, "Saving token should succeed")

        // When - create a new instance (simulating app restart)
        let newKeychainService = KeychainService.shared

        // Then
        XCTAssertEqual(
            newKeychainService.loadToken(), token, "Token should persist across instances")
    }
}
