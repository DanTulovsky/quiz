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

    func testLanguageCacheUpdate() {
        // Given
        let languages = [
            LanguageInfo(code: "en", name: "English", ttsLocale: "en-US", ttsVoice: "en-US-Standard-A"),
            LanguageInfo(code: "es", name: "Spanish", ttsLocale: "es-ES", ttsVoice: "es-ES-Standard-A"),
            LanguageInfo(code: "it", name: "Italian", ttsLocale: "it-IT", ttsVoice: "it-IT-Standard-A")
        ]
        mockAPIService.getLanguagesResult = .success(languages)

        // When
        viewModel.fetchLanguages()

        // Then - wait for languages to load
        let expectation = XCTestExpectation(description: "Languages loaded and cached")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
            // Verify cache is populated by checking O(1) lookup performance
            let startTime = Date()
            _ = self.viewModel.getDefaultVoice(for: "en")
            let lookupTime = Date().timeIntervalSince(startTime)

            // Dictionary lookup should be very fast (< 1ms)
            XCTAssertLessThan(lookupTime, 0.001, "Language lookup should be O(1) via cache")
            XCTAssertEqual(self.viewModel.getDefaultVoice(for: "en"), "en-US-Standard-A")
            XCTAssertEqual(self.viewModel.getDefaultVoice(for: "Spanish"), "es-ES-Standard-A")
            XCTAssertEqual(self.viewModel.getDefaultVoice(for: "IT"), "it-IT-Standard-A")
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }

    func testLanguageCacheCaseInsensitive() {
        // Given
        let languages = [
            LanguageInfo(code: "en", name: "English", ttsLocale: "en-US", ttsVoice: "en-US-Standard-A")
        ]
        mockAPIService.getLanguagesResult = .success(languages)
        viewModel.fetchLanguages()

        // When - wait for languages to load and cache to update
        // The didSet on availableLanguages uses DispatchQueue.main.async, so we need to wait longer
        let expectation = XCTestExpectation(description: "Case insensitive lookup works")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
            // Then - should find language regardless of case
            XCTAssertNotNil(self.viewModel.getDefaultVoice(for: "EN"), "Should find language with uppercase code")
            XCTAssertNotNil(self.viewModel.getDefaultVoice(for: "en"), "Should find language with lowercase code")
            XCTAssertNotNil(self.viewModel.getDefaultVoice(for: "English"), "Should find language with mixed case name")
            XCTAssertNotNil(self.viewModel.getDefaultVoice(for: "ENGLISH"), "Should find language with uppercase name")
            expectation.fulfill()
        }
        wait(for: [expectation], timeout: 1.0)
    }
}
