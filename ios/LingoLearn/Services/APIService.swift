import Foundation
import Combine

class APIService {
    static let shared = APIService()
    private let baseURL = URL(string: "http://localhost:8080/v1")!

    enum APIError: Error {
        case invalidURL
        case requestFailed(Error)
        case invalidResponse
        case decodingFailed(Error)
    }

    private init() {}

    private func authenticatedRequest(for url: URL, method: String = "GET") -> URLRequest {
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = method
        if let token = KeychainService.shared.loadToken() {
            urlRequest.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        return urlRequest
    }
    
    func login(request: LoginRequest) -> AnyPublisher<LoginResponse, APIError> {
        let url = baseURL.appendingPathComponent("auth/login")
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = "POST"
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")

        do {
            let encoder = JSONEncoder()
            encoder.keyEncodingStrategy = .convertToSnakeCase
            urlRequest.httpBody = try encoder.encode(request)
        } catch {
            return Fail(error: .decodingFailed(error)).eraseToAnyPublisher()
        }

        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { error in
                .requestFailed(error)
            }
            .flatMap { data, response -> AnyPublisher<LoginResponse, APIError> in
                guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }

                let decoder = JSONDecoder()
                decoder.keyDecodingStrategy = .convertFromSnakeCase
                return Just(data)
                    .decode(type: LoginResponse.self, decoder: decoder)
                    .mapError { error in
                        .decodingFailed(error)
                    }
                    .eraseToAnyPublisher()
            }
            .eraseToAnyPublisher()
    }

    func signup(request: UserCreateRequest) -> AnyPublisher<SuccessResponse, APIError> {
        let url = baseURL.appendingPathComponent("auth/signup")
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = "POST"
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")

        do {
            let encoder = JSONEncoder()
            encoder.keyEncodingStrategy = .convertToSnakeCase
            urlRequest.httpBody = try encoder.encode(request)
        } catch {
            return Fail(error: .decodingFailed(error)).eraseToAnyPublisher()
        }

        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { error in
                .requestFailed(error)
            }
            .flatMap { data, response -> AnyPublisher<SuccessResponse, APIError> in
                guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 201 else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }

                let decoder = JSONDecoder()
                decoder.keyDecodingStrategy = .convertFromSnakeCase
                return Just(data)
                    .decode(type: SuccessResponse.self, decoder: decoder)
                    .mapError { error in
                        .decodingFailed(error)
                    }
                    .eraseToAnyPublisher()
            }
            .eraseToAnyPublisher()
    }
    
    func getQuestion(language: Language?, level: Level?, type: String?, excludeType: String?) -> AnyPublisher<Question, APIError> {
        var urlComponents = URLComponents(url: baseURL.appendingPathComponent("quiz/question"), resolvingAgainstBaseURL: false)!
        var queryItems = [URLQueryItem]()
        if let language = language {
            queryItems.append(URLQueryItem(name: "language", value: language.rawValue))
        }
        if let level = level {
            queryItems.append(URLQueryItem(name: "level", value: level.rawValue))
        }
        if let type = type {
            queryItems.append(URLQueryItem(name: "type", value: type))
        }
        if let excludeType = excludeType {
            queryItems.append(URLQueryItem(name: "exclude_type", value: excludeType))
        }
        urlComponents.queryItems = queryItems
        
        let urlRequest = authenticatedRequest(for: urlComponents.url!)
        
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { error in
                .requestFailed(error)
            }
            .flatMap { data, response -> AnyPublisher<Question, APIError> in
                guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }
                
                let decoder = JSONDecoder()
                decoder.keyDecodingStrategy = .convertFromSnakeCase
                return Just(data)
                    .decode(type: Question.self, decoder: decoder)
                    .mapError { error in
                        .decodingFailed(error)
                    }
                    .eraseToAnyPublisher()
            }
            .eraseToAnyPublisher()
    }
    
    func postAnswer(request: AnswerRequest) -> AnyPublisher<AnswerResponse, APIError> {
        let url = baseURL.appendingPathComponent("quiz/answer")
        var urlRequest = authenticatedRequest(for: url, method: "POST")

        do {
            let encoder = JSONEncoder()
            encoder.keyEncodingStrategy = .convertToSnakeCase
            urlRequest.httpBody = try encoder.encode(request)
        } catch {
            return Fail(error: .decodingFailed(error)).eraseToAnyPublisher()
        }
        
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { error in
                .requestFailed(error)
            }
            .flatMap { data, response -> AnyPublisher<AnswerResponse, APIError> in
                guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }
                
                let decoder = JSONDecoder()
                decoder.keyDecodingStrategy = .convertFromSnakeCase
                return Just(data)
                    .decode(type: AnswerResponse.self, decoder: decoder)
                    .mapError { error in
                        .decodingFailed(error)
                    }
                    .eraseToAnyPublisher()
            }
            .eraseToAnyPublisher()
    }
    
    func getStories(language: Language?, level: Level?) -> AnyPublisher<StoryList, APIError> {
        var urlComponents = URLComponents(url: baseURL.appendingPathComponent("stories"), resolvingAgainstBaseURL: false)!
        var queryItems = [URLQueryItem]()
        if let language = language {
            queryItems.append(URLQueryItem(name: "language", value: language.rawValue))
        }
        if let level = level {
            queryItems.append(URLQueryItem(name: "level", value: level.rawValue))
        }
        urlComponents.queryItems = queryItems
        
        let urlRequest = authenticatedRequest(for: urlComponents.url!)
        
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { error in
                .requestFailed(error)
            }
            .flatMap { data, response -> AnyPublisher<StoryList, APIError> in
                guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }
                
                let decoder = JSONDecoder()
                decoder.keyDecodingStrategy = .convertFromSnakeCase
                return Just(data)
                    .decode(type: StoryList.self, decoder: decoder)
                    .mapError { error in
                        .decodingFailed(error)
                    }
                    .eraseToAnyPublisher()
            }
            .eraseToAnyPublisher()
    }
    
    func getStory(id: Int) -> AnyPublisher<StoryContent, APIError> {
        let url = baseURL.appendingPathComponent("stories/\(id)")
        let urlRequest = authenticatedRequest(for: url)
        
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { error in
                .requestFailed(error)
            }
            .flatMap { data, response -> AnyPublisher<StoryContent, APIError> in
                guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }
                
                let decoder = JSONDecoder()
                decoder.keyDecodingStrategy = .convertFromSnakeCase
                return Just(data)
                    .decode(type: StoryContent.self, decoder: decoder)
                    .mapError { error in
                        .decodingFailed(error)
                    }
                    .eraseToAnyPublisher()
            }
            .eraseToAnyPublisher()
    }
    
    func getSnippets(sourceLang: Language?, targetLang: Language?) -> AnyPublisher<SnippetList, APIError> {
        var urlComponents = URLComponents(url: baseURL.appendingPathComponent("snippets"), resolvingAgainstBaseURL: false)!
        var queryItems = [URLQueryItem]()
        if let sourceLang = sourceLang {
            queryItems.append(URLQueryItem(name: "source_lang", value: sourceLang.rawValue))
        }
        if let targetLang = targetLang {
            queryItems.append(URLQueryItem(name: "target_lang", value: targetLang.rawValue))
        }
        urlComponents.queryItems = queryItems
        
        let urlRequest = authenticatedRequest(for: urlComponents.url!)
        
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { error in
                .requestFailed(error)
            }
            .flatMap { data, response -> AnyPublisher<SnippetList, APIError> in
                guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }
                
                let decoder = JSONDecoder()
                decoder.keyDecodingStrategy = .convertFromSnakeCase
                return Just(data)
                    .decode(type: SnippetList.self, decoder: decoder)
                    .mapError { error in
                        .decodingFailed(error)
                    }
                    .eraseToAnyPublisher()
            }
            .eraseToAnyPublisher()
    }
    
    func createSnippet(request: CreateSnippetRequest) -> AnyPublisher<Snippet, APIError> {
        let url = baseURL.appendingPathComponent("snippets")
        var urlRequest = authenticatedRequest(for: url, method: "POST")

        do {
            let encoder = JSONEncoder()
            encoder.keyEncodingStrategy = .convertToSnakeCase
            urlRequest.httpBody = try encoder.encode(request)
        } catch {
            return Fail(error: .decodingFailed(error)).eraseToAnyPublisher()
        }
        
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { error in
                .requestFailed(error)
            }
            .flatMap { data, response -> AnyPublisher<Snippet, APIError> in
                guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 201 else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }
                
                let decoder = JSONDecoder()
                decoder.keyDecodingStrategy = .convertFromSnakeCase
                return Just(data)
                    .decode(type: Snippet.self, decoder: decoder)
                    .mapError { error in
                        .decodingFailed(error)
                    }
                    .eraseToAnyPublisher()
            }
            .eraseToAnyPublisher()
    }
    
    func updateSnippet(id: Int, request: UpdateSnippetRequest) -> AnyPublisher<Snippet, APIError> {
        let url = baseURL.appendingPathComponent("snippets/\(id)")
        var urlRequest = authenticatedRequest(for: url, method: "PUT")

        do {
            let encoder = JSONEncoder()
            encoder.keyEncodingStrategy = .convertToSnakeCase
            urlRequest.httpBody = try encoder.encode(request)
        } catch {
            return Fail(error: .decodingFailed(error)).eraseToAnyPublisher()
        }
        
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { error in
                .requestFailed(error)
            }
            .flatMap { data, response -> AnyPublisher<Snippet, APIError> in
                guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }
                
                let decoder = JSONDecoder()
                decoder.keyDecodingStrategy = .convertFromSnakeCase
                return Just(data)
                    .decode(type: Snippet.self, decoder: decoder)
                    .mapError { error in
                        .decodingFailed(error)
                    }
                    .eraseToAnyPublisher()
            }
            .eraseToAnyPublisher()
    }
    
    func getPhrasebook(language: Language) -> AnyPublisher<PhrasebookResponse, APIError> {
        var urlComponents = URLComponents(url: baseURL.appendingPathComponent("phrasebook"), resolvingAgainstBaseURL: false)!
        urlComponents.queryItems = [URLQueryItem(name: "language", value: language.rawValue)]
        
        let urlRequest = authenticatedRequest(for: urlComponents.url!)
        
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { error in
                .requestFailed(error)
            }
            .flatMap { data, response -> AnyPublisher<PhrasebookResponse, APIError> in
                guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }
                
                let decoder = JSONDecoder()
                decoder.keyDecodingStrategy = .convertFromSnakeCase
                return Just(data)
                    .decode(type: PhrasebookResponse.self, decoder: decoder)
                    .mapError { error in
                        .decodingFailed(error)
                    }
                    .eraseToAnyPublisher()
            }
            .eraseToAnyPublisher()
    }
    
    func updateUser(request: UserUpdateRequest) -> AnyPublisher<User, APIError> {
        let url = baseURL.appendingPathComponent("userz/profile")
        var urlRequest = authenticatedRequest(for: url, method: "PUT")
        
        do {
            let encoder = JSONEncoder()
            encoder.keyEncodingStrategy = .convertToSnakeCase
            urlRequest.httpBody = try encoder.encode(request)
        } catch {
            return Fail(error: .decodingFailed(error)).eraseToAnyPublisher()
        }
        
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { error in
                .requestFailed(error)
            }
            .flatMap { data, response -> AnyPublisher<User, APIError> in
                guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }
                
                let decoder = JSONDecoder()
                decoder.keyDecodingStrategy = .convertFromSnakeCase
                return Just(data)
                    .decode(type: User.self, decoder: decoder)
                    .mapError { error in
                        .decodingFailed(error)
                    }
                    .eraseToAnyPublisher()
            }
            .eraseToAnyPublisher()
    }
}
