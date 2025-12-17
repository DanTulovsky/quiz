import Foundation

// MARK: - Authentication
struct LoginRequest: Codable {
    let username: String
    let password: String
}

struct LoginResponse: Codable {
    let token: String
    let user: User
}

struct User: Codable {
    let id: Int
    let username: String
    let email: String
    let language: Language
    let level: Level
}

// MARK: - Quiz
struct Question: Codable {
    let id: Int
    let text: String
    let type: String
    let choices: [String]?
}

struct AnswerRequest: Codable {
    let questionId: Int
    let answer: String
}

struct AnswerResponse: Codable {
    let isCorrect: Bool
    let feedback: String
}

// MARK: - General
struct ErrorResponse: Codable {
    let error: String
}

struct SuccessResponse: Codable {
    let message: String
}

// MARK: - Authentication
struct AuthStatusResponse: Codable {
    let isAuthenticated: Bool
    let user: User?
}

struct UserCreateRequest: Codable {
    let username: String
    let email: String
    let password: String
}

struct SignupStatusResponse: Codable {
    let signupsEnabled: Bool
}

struct GoogleOAuthLoginResponse: Codable {
    let redirectUrl: String
}

enum Language: String, Codable {
    case en, es, fr, de, it
}

enum Level: String, Codable {
    case a1 = "A1"
    case a2 = "A2"
    case b1 = "B1"
    case b2 = "B2"
    case c1 = "C1"
    case c2 = "C2"
}

// MARK: - Stories
struct Story: Codable {
    let id: Int
    let title: String
    let language: Language
    let level: Level
    let sections: [StorySection]
}

struct StorySection: Codable {
    let id: Int
    let content: String
}

struct StoryList: Codable {
    let stories: [Story]
}

struct StoryContent: Codable {
    let id: Int
    let title: String
    let sections: [StorySection]
}

// MARK: - Snippets
struct Snippet: Codable {
    let id: Int
    let text: String
    let translation: String
    let sourceLanguage: Language
    let targetLanguage: Language
}

struct SnippetList: Codable {
    let snippets: [Snippet]
}

struct CreateSnippetRequest: Codable {
    let text: String
    let translation: String
    let sourceLanguage: Language
    let targetLanguage: Language
}

struct UpdateSnippetRequest: Codable {
    let text: String?
    let translation: String?
}

// MARK: - Phrasebook
struct PhrasebookCategory: Codable {
    let name: String
    let phrases: [PhrasebookPhrase]
}

struct PhrasebookPhrase: Codable {
    let phrase: String
    let translation: String
}

struct PhrasebookResponse: Codable {
    let categories: [PhrasebookCategory]
}

// MARK: - User
struct UserUpdateRequest: Codable {
    let username: String?
    let email: String?
    let language: Language?
    let level: Level?
}
