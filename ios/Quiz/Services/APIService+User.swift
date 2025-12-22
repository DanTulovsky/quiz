import Combine
import Foundation

extension APIService {

    func getLearningPreferences() -> AnyPublisher<UserLearningPreferences, APIError> {
        return get(path: "preferences/learning", responseType: UserLearningPreferences.self)
    }

    func updateLearningPreferences(prefs: UserLearningPreferences) -> AnyPublisher<
        UserLearningPreferences, APIError
    > {
        return put(
            path: "preferences/learning", body: prefs, responseType: UserLearningPreferences.self)
    }
}

