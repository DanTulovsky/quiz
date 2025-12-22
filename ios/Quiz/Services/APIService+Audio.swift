import Combine
import Foundation

extension APIService {
    func initializeTTSStream(request: TTSRequest) -> AnyPublisher<TTSStreamInitResponse, APIError> {
        return post(
            path: "audio/speech/init",
            body: request,
            responseType: TTSStreamInitResponse.self
        )
    }

    func streamURL(for streamId: String, token: String?) -> URL {
        let streamPath = baseURL.appendingPathComponent("audio")
            .appendingPathComponent("speech")
            .appendingPathComponent("stream")
            .appendingPathComponent(streamId)

        guard var components = URLComponents(url: streamPath, resolvingAgainstBaseURL: false) else {
            return streamPath
        }
        if let token = token {
            components.queryItems = [URLQueryItem(name: "token", value: token)]
        }
        return components.url ?? streamPath
    }

    func getVoices(language: String) -> AnyPublisher<[EdgeTTSVoiceInfo], APIError> {
        var params = QueryParameters()
        params.add("language", value: language)
        guard case .success(let url) = buildURL(path: "voices", queryItems: params.build()) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let urlRequest = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: urlRequest)
            .mapError { .requestFailed($0) }
            .flatMap { data, response -> AnyPublisher<[EdgeTTSVoiceInfo], APIError> in
                guard let httpResponse = response as? HTTPURLResponse else {
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }
                guard (200...299).contains(httpResponse.statusCode) else {
                    let decoder = JSONDecoder()
                    if let errorResp = try? decoder.decode(ErrorResponse.self, from: data),
                       let msg = errorResp.message ?? errorResp.error {
                        return Fail(
                            error: .backendError(code: errorResp.code, message: msg, details: errorResp.details)
                        ).eraseToAnyPublisher()
                    }
                    return Fail(error: .invalidResponse).eraseToAnyPublisher()
                }
                return self.decodeVoicesWithFallbacks(data: data)
            }
            .eraseToAnyPublisher()
    }

    private func decodeVoicesWithFallbacks(data: Data) -> AnyPublisher<[EdgeTTSVoiceInfo], APIError> {
        let decoder = JSONDecoder()

        if let voices = try? decoder.decode([EdgeTTSVoiceInfo].self, from: data) {
            return Just(voices).setFailureType(to: APIError.self).eraseToAnyPublisher()
        }

        if let wrapper = try? decoder.decode([String: [EdgeTTSVoiceInfo]].self, from: data),
           let voices = wrapper["voices"] {
            return Just(voices).setFailureType(to: APIError.self).eraseToAnyPublisher()
        }

        if let strings = try? decoder.decode([String].self, from: data) {
            let voices = strings.map { EdgeTTSVoiceInfo(shortName: $0) }
            return Just(voices).setFailureType(to: APIError.self).eraseToAnyPublisher()
        }

        if let json = try? JSONSerialization.jsonObject(with: data, options: []),
           let voicesArray = json as? [String] {
            let voices = voicesArray.map { EdgeTTSVoiceInfo(shortName: $0) }
            return Just(voices).setFailureType(to: APIError.self).eraseToAnyPublisher()
        }

        return Fail(
            error: .decodingFailed(
                NSError(
                    domain: "", code: 0,
                    userInfo: [NSLocalizedDescriptionKey: "Failed to decode voices response"]
                )
            )
        ).eraseToAnyPublisher()
    }
}
