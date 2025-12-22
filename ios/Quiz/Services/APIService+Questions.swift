import Combine
import Foundation

extension APIService {
    func getQuestion(
        language: String?, level: String?, type: String?, excludeType: String?
    ) -> AnyPublisher<QuestionFetchResult, APIError> {
        var params = QueryParameters()
        params.add("language", value: language)
        params.add("level", value: level)
        params.add("type", value: type)
        params.add("exclude_type", value: excludeType)
        guard
            case .success(let url) = buildURL(
                path: "quiz/question", queryItems: params.build())
        else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { data, response -> AnyPublisher<QuestionFetchResult, APIError> in
                self.handleQuestionResponse(data: data, response: response)
            }
            .eraseToAnyPublisher()
    }

    private func handleQuestionResponse(
        data: Data, response: URLResponse
    ) -> AnyPublisher<QuestionFetchResult, APIError> {
        guard let httpResponse = response as? HTTPURLResponse else {
            return Fail(error: .invalidResponse).eraseToAnyPublisher()
        }
        let decoder = createDateDecoder()

        if httpResponse.statusCode == 200 {
            return decodeQuestion(data: data, decoder: decoder)
        } else if httpResponse.statusCode == 202 {
            return decodeGeneratingStatus(data: data, decoder: decoder)
        } else {
            return decodeQuestionError(data: data, decoder: decoder)
        }
    }

    private func decodeQuestion(
        data: Data, decoder: JSONDecoder
    ) -> AnyPublisher<QuestionFetchResult, APIError> {
        return Just(data)
            .decode(type: Question.self, decoder: decoder)
            .map(QuestionFetchResult.question)
            .mapError { .decodingFailed($0) }
            .eraseToAnyPublisher()
    }

    private func decodeGeneratingStatus(
        data: Data, decoder: JSONDecoder
    ) -> AnyPublisher<QuestionFetchResult, APIError> {
        return Just(data)
            .decode(type: GeneratingStatusResponse.self, decoder: decoder)
            .map(QuestionFetchResult.generating)
            .mapError { .decodingFailed($0) }
            .eraseToAnyPublisher()
    }

    private func decodeQuestionError(
        data: Data, decoder: JSONDecoder
    ) -> AnyPublisher<QuestionFetchResult, APIError> {
        if let errorResp = try? decoder.decode(ErrorResponse.self, from: data),
            let msg = errorResp.message ?? errorResp.error
        {
            return Fail(
                error: .backendError(
                    code: errorResp.code, message: msg, details: errorResp.details)
            ).eraseToAnyPublisher()
        }
        return Fail(error: .invalidResponse).eraseToAnyPublisher()
    }

    private func createDateDecoder() -> JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            let dateStr = try container.decode(String.self)
            let iso8601WithFractional = ISO8601DateFormatter()
            iso8601WithFractional.formatOptions = [
                .withInternetDateTime, .withFractionalSeconds,
            ]
            let iso8601Standard = ISO8601DateFormatter()
            iso8601Standard.formatOptions = [.withInternetDateTime]
            if let date = iso8601WithFractional.date(from: dateStr)
                ?? iso8601Standard.date(from: dateStr)
            {
                return date
            }
            throw DecodingError.dataCorruptedError(
                in: container, debugDescription: "Invalid date format: \(dateStr)")
        }
        return decoder
    }

    func postAnswer(request: AnswerRequest) -> AnyPublisher<AnswerResponse, APIError> {
        return post(path: "quiz/answer", body: request, responseType: AnswerResponse.self)
    }

    func getDailyQuestions(date: String) -> AnyPublisher<DailyQuestionsResponse, APIError> {
        return get(path: "daily/questions/\(date)", responseType: DailyQuestionsResponse.self)
    }

    func postDailyAnswer(date: String, questionId: Int, userAnswerIndex: Int) -> AnyPublisher<
        DailyAnswerResponse, APIError
    > {
        return postJSON(
            path: "daily/questions/\(date)/answer/\(questionId)",
            body: ["user_answer_index": userAnswerIndex],
            responseType: DailyAnswerResponse.self
        )
    }
}
