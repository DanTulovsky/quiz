import XCTest
@testable import Quiz

class ModelsDynamicKeyTests: XCTestCase {

    func testDynamicKeyCreation() {
        // Given
        let keyString = "test_key"

        // When
        let key = DynamicKey(stringValue: keyString)

        // Then
        XCTAssertNotNil(key, "DynamicKey should be created successfully")
        XCTAssertEqual(key?.stringValue, keyString, "Key string value should match")
    }

    func testDynamicKeyWithEmptyString() {
        // Given
        let emptyString = ""

        // When
        let key = DynamicKey(stringValue: emptyString)

        // Then
        XCTAssertNotNil(key, "DynamicKey should handle empty string")
    }

    func testDynamicKeyWithSpecialCharacters() {
        // Given
        let specialString = "key-with-special_chars.123"

        // When
        let key = DynamicKey(stringValue: specialString)

        // Then
        XCTAssertNotNil(key, "DynamicKey should handle special characters")
        XCTAssertEqual(key?.stringValue, specialString, "Key should preserve special characters")
    }

    nonisolated func testPhrasebookWordEncoding() {
        // Given - create a PhrasebookWord by decoding from JSON
        let jsonString = """
        {
            "term": "test",
            "en": "test",
            "es": "prueba"
        }
        """
        let data = jsonString.data(using: .utf8)!

        // When/Then - should not crash when decoding
        XCTAssertNoThrow(try {
            let decoder = JSONDecoder()
            let word = try decoder.decode(PhrasebookWord.self, from: data)
            let encoder = JSONEncoder()
            _ = try encoder.encode(word)
        }(), "Encoding/Decoding PhrasebookWord should not throw")
    }

    nonisolated func testEdgeTTSVoiceInfoDecoding() {
        // Given - test various decoding scenarios
        let jsonString = """
        {
            "name": "Test Voice",
            "short_name": "test-voice",
            "display_name": "Test Voice Display",
            "locale": "en-US",
            "gender": "Female"
        }
        """
        let data = jsonString.data(using: .utf8)!

        // When/Then - should not crash when decoding
        XCTAssertNoThrow(try {
            let decoder = JSONDecoder()
            _ = try decoder.decode(EdgeTTSVoiceInfo.self, from: data)
        }(), "Decoding EdgeTTSVoiceInfo should not throw")
    }

    nonisolated func testEdgeTTSVoiceInfoDecodingWithMissingFields() {
        // Given - test with missing optional fields
        let jsonString = """
        {
            "short_name": "test-voice"
        }
        """
        let data = jsonString.data(using: .utf8)!

        // When/Then - should not crash when decoding
        XCTAssertNoThrow(try {
            let decoder = JSONDecoder()
            _ = try decoder.decode(EdgeTTSVoiceInfo.self, from: data)
        }(), "Decoding EdgeTTSVoiceInfo with missing fields should not throw")
    }
}

