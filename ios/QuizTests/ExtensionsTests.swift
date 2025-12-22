import XCTest
import Combine
@testable import Quiz

class ExtensionsTests: XCTestCase {

    // MARK: - LanguageInfo Array Extension Tests

    func testFindLanguageByCode() {
        // Given
        let languages: [LanguageInfo] = [
            LanguageInfo(code: "en", name: "English", ttsLocale: "en-US", ttsVoice: "en-US-JennyNeural"),
            LanguageInfo(code: "es", name: "Spanish", ttsLocale: "es-ES", ttsVoice: "es-ES-ElviraNeural"),
            LanguageInfo(code: "fr", name: "French", ttsLocale: "fr-FR", ttsVoice: "fr-FR-DeniseNeural")
        ]

        // When
        let found = languages.find(byCodeOrName: "es")

        // Then
        XCTAssertNotNil(found, "Should find language by code")
        XCTAssertEqual(found?.code, "es", "Found language should have correct code")
        XCTAssertEqual(found?.name, "Spanish", "Found language should have correct name")
    }

    func testFindLanguageByName() {
        // Given
        let languages: [LanguageInfo] = [
            LanguageInfo(code: "en", name: "English", ttsLocale: "en-US", ttsVoice: "en-US-JennyNeural"),
            LanguageInfo(code: "es", name: "Spanish", ttsLocale: "es-ES", ttsVoice: "es-ES-ElviraNeural")
        ]

        // When
        let found = languages.find(byCodeOrName: "Spanish")

        // Then
        XCTAssertNotNil(found, "Should find language by name")
        XCTAssertEqual(found?.code, "es", "Found language should have correct code")
    }

    func testFindLanguageCaseInsensitive() {
        // Given
        let languages: [LanguageInfo] = [
            LanguageInfo(code: "en", name: "English", ttsLocale: "en-US", ttsVoice: "en-US-JennyNeural"),
            LanguageInfo(code: "es", name: "Spanish", ttsLocale: "es-ES", ttsVoice: "es-ES-ElviraNeural")
        ]

        // When
        let found1 = languages.find(byCodeOrName: "ENGLISH")
        let found2 = languages.find(byCodeOrName: "SPANISH")
        let found3 = languages.find(byCodeOrName: "En")

        // Then
        XCTAssertNotNil(found1, "Should find language with uppercase name")
        XCTAssertNotNil(found2, "Should find language with uppercase name")
        XCTAssertNotNil(found3, "Should find language with mixed case code")
        XCTAssertEqual(found1?.code, "en")
        XCTAssertEqual(found2?.code, "es")
        XCTAssertEqual(found3?.code, "en")
    }

    func testFindLanguageNotFound() {
        // Given
        let languages: [LanguageInfo] = [
            LanguageInfo(code: "en", name: "English", ttsLocale: "en-US", ttsVoice: "en-US-JennyNeural")
        ]

        // When
        let found = languages.find(byCodeOrName: "de")

        // Then
        XCTAssertNil(found, "Should return nil when language not found")
    }

    func testFindLanguageEmptyArray() {
        // Given
        let languages: [LanguageInfo] = []

        // When
        let found = languages.find(byCodeOrName: "en")

        // Then
        XCTAssertNil(found, "Should return nil for empty array")
    }

    // MARK: - APIService Extension Tests

    func testGetSnippetsForQuestion() {
        // Given
        let mockService = MockAPIService()
        var cancellables = Set<AnyCancellable>()
        let expectedSnippets = SnippetList(
            limit: 10,
            offset: 0,
            query: nil,
            snippets: [
                Snippet(
                    id: 1,
                    originalText: "Hello",
                    translatedText: "Hola",
                    context: nil,
                    sourceLanguage: "en",
                    targetLanguage: "es",
                    difficultyLevel: "A1",
                    questionId: 123,
                    storyId: nil,
                    sectionId: nil
                )
            ]
        )
        mockService.getSnippetsResult = .success(expectedSnippets)

        // When
        let expectation = XCTestExpectation(description: "Get snippets for question")
        var receivedSnippets: SnippetList?
        var receivedError: APIService.APIError?

        mockService.getSnippetsForQuestion(questionId: 123)
            .sink(
                receiveCompletion: { completion in
                    if case .failure(let error) = completion {
                        receivedError = error
                    }
                    expectation.fulfill()
                },
                receiveValue: { snippets in
                    receivedSnippets = snippets
                }
            )
            .store(in: &cancellables)

        wait(for: [expectation], timeout: 1.0)

        // Then
        XCTAssertNil(receivedError, "Should not receive error")
        XCTAssertNotNil(receivedSnippets, "Should receive snippets")
        XCTAssertEqual(receivedSnippets?.snippets.count, 1, "Should receive one snippet")
    }

    func testGetSnippetsForStory() {
        // Given
        let mockService = MockAPIService()
        var cancellables = Set<AnyCancellable>()
        let expectedSnippets = SnippetList(
            limit: 10,
            offset: 0,
            query: nil,
            snippets: [
                Snippet(
                    id: 1,
                    originalText: "Story text",
                    translatedText: "Texto de la historia",
                    context: nil,
                    sourceLanguage: "en",
                    targetLanguage: "es",
                    difficultyLevel: "B1",
                    questionId: nil,
                    storyId: 456,
                    sectionId: nil
                )
            ]
        )
        mockService.getSnippetsResult = .success(expectedSnippets)

        // When
        let expectation = XCTestExpectation(description: "Get snippets for story")
        var receivedSnippets: SnippetList?

        mockService.getSnippetsForStory(storyId: 456)
            .sink(
                receiveCompletion: { _ in
                    expectation.fulfill()
                },
                receiveValue: { snippets in
                    receivedSnippets = snippets
                }
            )
            .store(in: &cancellables)

        wait(for: [expectation], timeout: 1.0)

        // Then
        XCTAssertNotNil(receivedSnippets, "Should receive snippets")
        XCTAssertEqual(receivedSnippets?.snippets.first?.storyId, 456, "Snippet should have correct story ID")
    }
}
