import Foundation

struct LoginRequest: Codable {
    let username: String
    let password: String
}

struct LoginResponse: Codable {
    let success: Bool
    let message: String
    let user: User
}

struct User: Codable, Equatable {
    let id: Int
    let username: String
    let email: String
    let timezone: String?
    let preferredLanguage: String?
    let currentLevel: String?
    let aiEnabled: Bool?
    let isPaused: Bool?
    let wordOfDayEmailEnabled: Bool?
    let aiProvider: String?
    let aiModel: String?
    let hasApiKey: Bool?

    enum CodingKeys: String, CodingKey {
        case id, username, email, timezone
        case preferredLanguage = "preferred_language"
        case currentLevel = "current_level"
        case aiEnabled = "ai_enabled"
        case isPaused = "is_paused"
        case wordOfDayEmailEnabled = "word_of_day_email_enabled"
        case aiProvider = "ai_provider"
        case aiModel = "ai_model"
        case hasApiKey = "has_api_key"
    }
}

struct ErrorResponse: Codable {
    let error: String?
    let message: String?
    let details: String?
}

struct SuccessResponse: Codable {
    let message: String
}

struct AuthStatusResponse: Codable {
    let authenticated: Bool
    let user: User?
}

struct UserCreateRequest: Codable {
    let username: String
    let email: String
    let password: String
}

struct SignupStatusResponse: Codable {
    let signupsEnabled: Bool
    enum CodingKeys: String, CodingKey {
        case signupsEnabled = "signups_enabled"
    }
}

struct GoogleOAuthLoginResponse: Codable {
    let redirectUrl: String
    enum CodingKeys: String, CodingKey {
        case redirectUrl = "redirect_url"
    }
}

enum Language: String, Codable, CaseIterable {
    case english, spanish, french, german, italian, en, es, fr, de, it
}

enum Level: String, Codable, CaseIterable {
    case a1 = "A1"
    case a2 = "A2"
    case b1 = "B1"
    case b2 = "B2"
    case c1 = "C1"
    case c2 = "C2"
}

enum JSONValue: Codable, Equatable {
    case string(String)
    case number(Double)
    case bool(Bool)
    case object([String: JSONValue])
    case array([JSONValue])
    case null

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if container.decodeNil() {
            self = .null
        } else if let b = try? container.decode(Bool.self) {
            self = .bool(b)
        } else if let i = try? container.decode(Int.self) {
            self = .number(Double(i))
        } else if let d = try? container.decode(Double.self) {
            self = .number(d)
        } else if let s = try? container.decode(String.self) {
            self = .string(s)
        } else if let a = try? container.decode([JSONValue].self) {
            self = .array(a)
        } else if let o = try? container.decode([String: JSONValue].self) {
            self = .object(o)
        } else {
            throw DecodingError.typeMismatch(
                JSONValue.self,
                DecodingError.Context(codingPath: decoder.codingPath, debugDescription: "Unsupported JSON value")
            )
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch self {
        case .string(let s):
            try container.encode(s)
        case .number(let n):
            try container.encode(n)
        case .bool(let b):
            try container.encode(b)
        case .object(let o):
            try container.encode(o)
        case .array(let a):
            try container.encode(a)
        case .null:
            try container.encodeNil()
        }
    }
}

struct Question: Codable {
    let id: Int
    let type: String
    let language: String
    let level: String
    let content: [String: JSONValue]
    let correctAnswerIndex: Int?

    enum CodingKeys: String, CodingKey {
        case id, type, language, level, content
        case correctAnswerIndex = "correct_answer"
    }
}

struct GeneratingStatusResponse: Codable {
    let message: String
    let status: String
}

struct AnswerRequest: Codable {
    let questionId: Int
    let userAnswerIndex: Int
    let responseTimeMs: Int?

    enum CodingKeys: String, CodingKey {
        case questionId = "question_id"
        case userAnswerIndex = "user_answer_index"
        case responseTimeMs = "response_time_ms"
    }
}

struct AnswerResponse: Codable, Equatable {
    let isCorrect: Bool
    let userAnswer: String
    let userAnswerIndex: Int
    let explanation: String
    let correctAnswerIndex: Int

    enum CodingKeys: String, CodingKey {
        case isCorrect = "is_correct"
        case userAnswer = "user_answer"
        case userAnswerIndex = "user_answer_index"
        case explanation
        case correctAnswerIndex = "correct_answer_index"
    }
}

struct Snippet: Codable, Identifiable {
    let id: Int
    let originalText: String
    let translatedText: String
    let context: String?
    let sourceLanguage: String?
    let targetLanguage: String?
    let difficultyLevel: String?
    let questionId: Int?
    let storyId: Int?
    let sectionId: Int?

    enum CodingKeys: String, CodingKey {
        case id, context
        case originalText = "original_text"
        case translatedText = "translated_text"
        case sourceLanguage = "source_language"
        case targetLanguage = "target_language"
        case difficultyLevel = "difficulty_level"
        case questionId = "question_id"
        case storyId = "story_id"
        case sectionId = "section_id"
    }
}

struct SnippetList: Codable {
    let limit: Int
    let offset: Int
    let query: String?
    let snippets: [Snippet]
}

struct StorySummary: Codable {
    let id: Int
    let title: String
    let language: String
    let status: String
}

struct StorySection: Codable {
    let id: Int
    let sectionNumber: Int
    let content: String

    enum CodingKeys: String, CodingKey {
        case id, content
        case sectionNumber = "section_number"
    }
}

struct StoryContent: Codable {
    let id: Int
    let title: String
    let language: String
    let sections: [StorySection]
}

struct UserUpdateRequest: Codable {
    var username: String?
    var email: String?
    var timezone: String?
    var preferredLanguage: String?
    var currentLevel: String?
    var aiEnabled: Bool?
    var isPaused: Bool?
    var wordOfDayEmailEnabled: Bool?
    var aiProvider: String?
    var aiModel: String?
    var apiKey: String?

    enum CodingKeys: String, CodingKey {
        case username, email, timezone
        case preferredLanguage = "preferred_language"
        case currentLevel = "current_level"
        case aiEnabled = "ai_enabled"
        case isPaused = "is_paused"
        case wordOfDayEmailEnabled = "word_of_day_email_enabled"
        case aiProvider = "ai_provider"
        case aiModel = "ai_model"
        case apiKey = "api_key"
    }
}

struct PhrasebookIndex: Codable {
    let categories: [String]
}

struct PhrasebookCategoryInfo: Codable {
    let id: String
    let name: String
    let emoji: String?
}

struct PhrasebookData: Codable {
    let category: String
    let sections: [PhrasebookSection]
}

struct PhrasebookSection: Codable {
    let title: String
    let words: [PhrasebookWord]
}

struct PhrasebookWord: Codable {
    let term: String
    let icon: String?
    let note: String?
    let translations: [String: String]

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: DynamicKey.self)
        var trans: [String: String] = [:]
        
        let allKeys = container.allKeys
        var termVal = ""
        var iconVal: String? = nil
        var noteVal: String? = nil
        
        for key in allKeys {
            if key.stringValue == "term" {
                termVal = try container.decode(String.self, forKey: key)
            } else if key.stringValue == "icon" {
                iconVal = try? container.decode(String.self, forKey: key)
            } else if key.stringValue == "note" {
                noteVal = try? container.decode(String.self, forKey: key)
            } else {
                if let val = try? container.decode(String.self, forKey: key) {
                    trans[key.stringValue] = val
                }
            }
        }
        self.term = termVal
        self.icon = iconVal
        self.note = noteVal
        self.translations = trans
    }
    
    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: DynamicKey.self)
        try container.encode(term, forKey: DynamicKey(stringValue: "term")!)
        if let icon = icon { try container.encode(icon, forKey: DynamicKey(stringValue: "icon")!) }
        if let note = note { try container.encode(note, forKey: DynamicKey(stringValue: "note")!) }
        for (k, v) in translations {
            try container.encode(v, forKey: DynamicKey(stringValue: k)!)
        }
    }
}

struct DynamicKey: CodingKey {
    var stringValue: String
    var intValue: Int?
    init?(stringValue: String) { self.stringValue = stringValue; self.intValue = nil }
    init?(intValue: Int) { return nil }
}


// Daily Questions
struct DailyQuestionsResponse: Codable {
    let date: String
    let questions: [DailyQuestionWithDetails]
}

struct DailyQuestionWithDetails: Codable {
    let id: Int
    let questionId: Int
    let question: Question
    let isCompleted: Bool

    enum CodingKeys: String, CodingKey {
        case id, question
        case questionId = "question_id"
        case isCompleted = "is_completed"
    }
}

struct DailyAnswerResponse: Codable, Equatable {
    let isCorrect: Bool
    let explanation: String
    let isCompleted: Bool
    let correctAnswerIndex: Int
    let userAnswer: String
    let userAnswerIndex: Int

    enum CodingKeys: String, CodingKey {
        case explanation
        case isCorrect = "is_correct"
        case isCompleted = "is_completed"
        case correctAnswerIndex = "correct_answer_index"
        case userAnswer = "user_answer"
        case userAnswerIndex = "user_answer_index"
    }
}

struct FullPhrasebookCategory: Codable {
    let info: PhrasebookCategoryInfo
    let data: PhrasebookData
}

// Translation Practice
struct TranslationPracticeGenerateRequest: Codable {
    let language: String
    let level: String
    let direction: String
    let topic: String?
}

struct TranslationPracticeSentenceResponse: Codable, Equatable {
    let id: Int
    let sentenceText: String
    let sourceLanguage: String
    let targetLanguage: String
    let languageLevel: String
    let sourceType: String
    let sourceId: Int?
    let topic: String?
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case id, topic
        case sentenceText = "sentence_text"
        case sourceLanguage = "source_language"
        case targetLanguage = "target_language"
        case languageLevel = "language_level"
        case sourceType = "source_type"
        case sourceId = "source_id"
        case createdAt = "created_at"
    }
}

struct TranslationPracticeSubmitRequest: Codable {
    let sentenceId: Int
    let originalSentence: String
    let userTranslation: String
    let translationDirection: String

    enum CodingKeys: String, CodingKey {
        case sentenceId = "sentence_id"
        case originalSentence = "original_sentence"
        case userTranslation = "user_translation"
        case translationDirection = "translation_direction"
    }
}

struct TranslationPracticeSessionResponse: Codable, Identifiable, Equatable {
    let id: Int
    let sentenceId: Int
    let originalSentence: String
    let userTranslation: String
    let translationDirection: String
    let aiFeedback: String
    let aiScore: Float?
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case id, aiFeedback = "ai_feedback", aiScore = "ai_score"
        case sentenceId = "sentence_id"
        case originalSentence = "original_sentence"
        case userTranslation = "user_translation"
        case translationDirection = "translation_direction"
        case createdAt = "created_at"
    }
}

struct TranslationPracticeHistoryResponse: Codable {
    let sessions: [TranslationPracticeSessionResponse]
    let total: Int
    let limit: Int
    let offset: Int
}

// Verb Conjugation Models
struct VerbConjugationsData: Codable {
    let language: String
    let languageName: String
    let verbs: [VerbConjugationSummary]
}

struct VerbConjugationSummary: Codable {
    let infinitive: String
    let infinitiveEn: String
    let slug: String?
    let category: String
}

struct VerbConjugationDetail: Codable {
    let infinitive: String
    let infinitiveEn: String
    let slug: String?
    let category: String
    let tenses: [Tense]
}

struct Tense: Codable {
    let tenseId: String
    let tenseName: String
    let tenseNameEn: String
    let description: String
    let conjugations: [Conjugation]
}

struct Conjugation: Codable {
    let pronoun: String
    let form: String
    let exampleSentence: String
    let exampleSentenceEn: String
}

extension Language {
    var code: String {
        switch self {
        case .english, .en: return "en"
        case .spanish, .es: return "es"
        case .french, .fr: return "fr"
        case .german, .de: return "de"
        case .italian, .it: return "it"
        }
    }
}

// Word of the Day
struct WordOfTheDayDisplay: Codable {
    let date: String
    let word: String
    let translation: String
    let sentence: String
    let sourceType: String
    let sourceId: Int
    let language: String
    let level: String?
    let context: String?
    let explanation: String?
    let topicCategory: String?

    enum CodingKeys: String, CodingKey {
        case date, word, translation, sentence, language, level, context, explanation
        case sourceType = "source_type"
        case sourceId = "source_id"
        case topicCategory = "topic_category"
    }
}

// AI History
struct ChatMessage: Codable, Identifiable {
    let id: String
    let conversationId: String
    let questionId: Int?
    let role: String
    let content: ChatMessageContent
    let bookmarked: Bool?
    let createdAt: Date
    let updatedAt: Date
    let conversationTitle: String?

    enum CodingKeys: String, CodingKey {
        case id, role, content, bookmarked
        case conversationId = "conversation_id"
        case questionId = "question_id"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
        case conversationTitle = "conversation_title"
    }
}

struct ChatMessageContent: Codable {
    let text: String
}

struct Conversation: Codable, Identifiable {
    let id: String
    let userId: Int
    let title: String
    let createdAt: Date
    let updatedAt: Date
    let messageCount: Int?
    let messages: [ChatMessage]?

    enum CodingKeys: String, CodingKey {
        case id, title
        case userId = "user_id"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
        case messageCount = "message_count"
        case messages
    }
}

struct ConversationListResponse: Codable {
    let conversations: [Conversation]
    let total: Int
}

struct BookmarkedMessagesResponse: Codable {
    let messages: [ChatMessage]
    let total: Int
}

struct ReportQuestionRequest: Codable {
    let reportReason: String?

    enum CodingKeys: String, CodingKey {
        case reportReason = "report_reason"
    }
}

struct MarkQuestionKnownRequest: Codable {
    let confidenceLevel: Int?

    enum CodingKeys: String, CodingKey {
        case confidenceLevel = "confidence_level"
    }
}

struct StorySectionQuestion: Codable, Identifiable {
    let id: Int
    let sectionId: Int
    let questionText: String
    let options: [String]
    let correctAnswerIndex: Int
    let explanation: String?
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case id, options, explanation
        case sectionId = "section_id"
        case questionText = "question_text"
        case correctAnswerIndex = "correct_answer_index"
        case createdAt = "created_at"
    }
}

struct StorySectionWithQuestions: Codable {
    let id: Int
    let sectionNumber: Int
    let content: String
    let questions: [StorySectionQuestion]

    enum CodingKeys: String, CodingKey {
        case id, content, questions
        case sectionNumber = "section_number"
    }
}

struct UserLearningPreferences: Codable, Equatable {
    var focusOnWeakAreas: Bool
    var freshQuestionRatio: Float
    var knownQuestionPenalty: Float
    var reviewIntervalDays: Int
    var weakAreaBoost: Float
    var dailyReminderEnabled: Bool
    var ttsVoice: String?
    var dailyGoal: Int?

    enum CodingKeys: String, CodingKey {
        case focusOnWeakAreas = "focus_on_weak_areas"
        case freshQuestionRatio = "fresh_question_ratio"
        case knownQuestionPenalty = "known_question_penalty"
        case reviewIntervalDays = "review_interval_days"
        case weakAreaBoost = "weak_area_boost"
        case dailyReminderEnabled = "daily_reminder_enabled"
        case ttsVoice = "tts_voice"
        case dailyGoal = "daily_goal"
    }
}

struct AIModelInfo: Codable {
    let code: String
    let name: String
}

struct AIProviderInfo: Codable, Identifiable {
    let code: String
    let name: String
    let url: String?
    let usageSupported: Bool?
    let models: [AIModelInfo]
    
    var id: String { code }

    enum CodingKeys: String, CodingKey {
        case name, code, url, models
        case usageSupported = "usage_supported"
    }
}

struct AIProvidersResponse: Codable {
    let providers: [AIProviderInfo]
    let levels: [String]
}

struct EdgeTTSVoiceInfo: Codable, Identifiable {
    var id: String { shortName ?? name ?? UUID().uuidString }

    init(shortName: String) {
        self.shortName = shortName
        self.name = nil
        self.displayName = nil
        self.locale = nil
        self.gender = nil
    }

    let name: String?
    let shortName: String?
    let displayName: String?
    let locale: String?
    let gender: String?

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: DynamicKey.self)
        self.name = (try? container.decode(String.self, forKey: DynamicKey(stringValue: "name")!))
        self.shortName = (try? container.decode(String.self, forKey: DynamicKey(stringValue: "short_name")!))
        self.displayName = (try? container.decode(String.self, forKey: DynamicKey(stringValue: "display_name")!))
        self.locale = (try? container.decode(String.self, forKey: DynamicKey(stringValue: "locale")!)) ?? (try? container.decode(String.self, forKey: DynamicKey(stringValue: "Locale")!))
        self.gender = (try? container.decode(String.self, forKey: DynamicKey(stringValue: "gender")!)) ?? (try? container.decode(String.self, forKey: DynamicKey(stringValue: "Gender")!))
    }
}

struct TTSRequest: Codable {
    let input: String
    var voice: String? = "echo"
    var model: String? = "tts-1"
    var streamFormat: String? = "mp3" // Use mp3 for iOS streaming

    enum CodingKeys: String, CodingKey {
        case input
        case voice
        case model
        case streamFormat = "stream_format"
    }
}

struct TTSStreamInitResponse: Codable {
    let streamId: String
    let token: String?

    enum CodingKeys: String, CodingKey {
        case streamId = "stream_id"
        case token
    }
}
