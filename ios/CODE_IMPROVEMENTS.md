# iOS Code Improvements - Code Sharing & Refactoring Suggestions

## Overview
This document outlines opportunities to improve code sharing, reduce duplication, and enhance maintainability in the iOS Quiz app.

## 1. Base ViewModel Protocol/Class

### Problem
All ViewModels have identical boilerplate:
- `@Published var error: APIService.APIError?`
- `@Published var isLoading = false`
- `private var cancellables = Set<AnyCancellable>()`
- Similar error handling patterns
- `cancelAllRequests()` and `deinit` implementations

### Solution
Create a base `BaseViewModel` class that all ViewModels can inherit from:

```swift
// Utils/BaseViewModel.swift
import Foundation
import Combine

class BaseViewModel: ObservableObject {
    @Published var error: APIService.APIError?
    @Published var isLoading = false

    var apiService: APIService
    var cancellables = Set<AnyCancellable>()

    init(apiService: APIService = APIService.shared) {
        self.apiService = apiService
    }

    func cancelAllRequests() {
        cancellables.removeAll()
    }

    deinit {
        cancelAllRequests()
    }

    // Helper method for common error handling pattern
    func handleError(_ error: APIService.APIError) {
        self.error = error
        self.isLoading = false
    }
}
```

**Benefits:**
- Eliminates ~15-20 lines of boilerplate per ViewModel
- Ensures consistent error handling
- Centralizes cancellable management

**Migration:** Change all ViewModels from `class XViewModel: ObservableObject` to `class XViewModel: BaseViewModel` and remove duplicate properties.

---

## 2. Publisher Extension for Error Handling

### Problem
Every API call has repetitive error handling:
```swift
.receive(on: DispatchQueue.main)
.sink(receiveCompletion: { [weak self] completion in
    self?.isLoading = false
    if case .failure(let error) = completion {
        self?.error = error
    }
}, receiveValue: { [weak self] value in
    // handle value
})
```

### Solution
Extend `Extensions.swift` with a reusable publisher extension:

```swift
// Utils/Extensions.swift (add to existing file)
extension Publisher where Failure == APIService.APIError {
    func handleLoadingAndError<T: BaseViewModel>(
        on viewModel: T,
        isLoading: ReferenceWritableKeyPath<T, Bool> = \.isLoading,
        error: ReferenceWritableKeyPath<T, APIService.APIError?> = \.error
    ) -> AnyPublisher<Output, Failure> {
        return self
            .receive(on: DispatchQueue.main)
            .handleEvents(
                receiveSubscription: { _ in
                    viewModel[keyPath: isLoading] = true
                },
                receiveCompletion: { completion in
                    viewModel[keyPath: isLoading] = false
                    if case .failure(let err) = completion {
                        viewModel[keyPath: error] = err
                    }
                }
            )
            .eraseToAnyPublisher()
    }
}
```

**Usage:**
```swift
// Before:
apiService.getStories()
    .receive(on: DispatchQueue.main)
    .sink(receiveCompletion: { [weak self] completion in
        self?.isLoading = false
        if case .failure(let error) = completion {
            self?.error = error
        }
    }, receiveValue: { [weak self] stories in
        self?.stories = stories
    })
    .store(in: &cancellables)

// After:
apiService.getStories()
    .handleLoadingAndError(on: self)
    .sink(receiveValue: { [weak self] stories in
        self?.stories = stories
    })
    .store(in: &cancellables)
```

**Benefits:**
- Reduces each API call by ~5-7 lines
- Consistent error handling across all ViewModels
- Automatic loading state management

---

## 3. APIService Refactoring

### Problem
`APIService.swift` is 900+ lines with repetitive patterns:
- Every method builds URLs similarly
- Every method creates requests similarly
- Every method handles responses similarly
- Encoding/decoding logic is duplicated

### Solution
Extract common request building into helper methods:

```swift
// Services/APIService.swift (refactor existing)
extension APIService {
    // Generic GET request
    func get<T: Decodable>(
        path: String,
        queryItems: [URLQueryItem]? = nil,
        responseType: T.Type
    ) -> AnyPublisher<T, APIError> {
        guard case .success(let url) = buildURL(path: path, queryItems: queryItems) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let request = authenticatedRequest(for: url)
        return URLSession.shared.dataTaskPublisher(for: request)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    // Generic POST request
    func post<T: Decodable, U: Encodable>(
        path: String,
        body: U,
        responseType: T.Type
    ) -> AnyPublisher<T, APIError> {
        guard case .success(let url) = buildURL(path: path) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        guard case .success(let bodyData) = encodeBody(body) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: nil)))
                .eraseToAnyPublisher()
        }
        let request = authenticatedRequest(for: url, method: "POST", body: bodyData)
        return URLSession.shared.dataTaskPublisher(for: request)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    // Generic PUT request
    func put<T: Decodable, U: Encodable>(
        path: String,
        body: U,
        responseType: T.Type
    ) -> AnyPublisher<T, APIError> {
        guard case .success(let url) = buildURL(path: path) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        guard case .success(let bodyData) = encodeBody(body) else {
            return Fail(error: .encodingFailed(NSError(domain: "APIService", code: -1, userInfo: nil)))
                .eraseToAnyPublisher()
        }
        let request = authenticatedRequest(for: url, method: "PUT", body: bodyData)
        return URLSession.shared.dataTaskPublisher(for: request)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }

    // Generic DELETE request
    func delete<T: Decodable>(
        path: String,
        responseType: T.Type
    ) -> AnyPublisher<T, APIError> {
        guard case .success(let url) = buildURL(path: path) else {
            return Fail(error: .invalidURL).eraseToAnyPublisher()
        }
        let request = authenticatedRequest(for: url, method: "DELETE")
        return URLSession.shared.dataTaskPublisher(for: request)
            .mapError { .requestFailed($0) }
            .flatMap { self.handleResponse($0.data, $0.response) }
            .eraseToAnyPublisher()
    }
}
```

**Then simplify methods:**
```swift
// Before (30+ lines):
func getStories() -> AnyPublisher<[StorySummary], APIError> {
    let url = baseURL.appendingPathComponent("story")
    let urlRequest = authenticatedRequest(for: url)
    return URLSession.shared.dataTaskPublisher(for: urlRequest)
        .mapError { .requestFailed($0) }
        .flatMap { self.handleResponse($0.data, $0.response) }
        .eraseToAnyPublisher()
}

// After (1 line):
func getStories() -> AnyPublisher<[StorySummary], APIError> {
    return get(path: "story", responseType: [StorySummary].self)
}
```

**Benefits:**
- Reduces APIService from ~900 lines to ~400-500 lines
- Eliminates ~20-30 lines per API method
- Easier to add new endpoints
- Consistent error handling

---

## 4. Question Action Protocol

### Problem
Both `QuizViewModel` and `DailyViewModel` have identical `reportQuestion` and `markQuestionKnown` methods with slight variations.

### Solution
Create a protocol for question actions:

```swift
// Utils/QuestionActions.swift
protocol QuestionActions {
    var apiService: APIService { get }
    var isSubmittingAction: Bool { get set }
    var error: APIService.APIError? { get set }
    var showReportModal: Bool { get set }
    var showMarkKnownModal: Bool { get set }
    var isReported: Bool { get set }
}

extension QuestionActions {
    func reportQuestion(id: Int, reason: String?) {
        isSubmittingAction = true
        let request = ReportQuestionRequest(reportReason: reason)
        apiService.reportQuestion(id: id, request: request)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                guard let self = self else { return }
                self.isSubmittingAction = false
                if case .failure(let error) = completion {
                    self.error = error
                } else {
                    self.isReported = true
                    self.showReportModal = false
                }
            }, receiveValue: { _ in })
            .store(in: &cancellables) // Note: requires cancellables access
    }

    func markQuestionKnown(id: Int, confidence: Int) {
        isSubmittingAction = true
        let request = MarkQuestionKnownRequest(confidenceLevel: confidence)
        apiService.markQuestionKnown(id: id, request: request)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                guard let self = self else { return }
                self.isSubmittingAction = false
                if case .failure(let error) = completion {
                    self.error = error
                } else {
                    self.showMarkKnownModal = false
                }
            }, receiveValue: { _ in })
            .store(in: &cancellables)
    }
}
```

**Note:** This requires access to `cancellables`, which could be handled via a protocol requirement or by making it part of `BaseViewModel`.

**Benefits:**
- Eliminates ~30 lines of duplicate code per ViewModel
- Consistent question action behavior

---

## 5. Date Utilities Enhancement

### Problem
Date formatting is scattered and `DateFormatters` could be more comprehensive.

### Solution
Enhance `DateFormatters.swift`:

```swift
// Utils/DateFormatters.swift (enhance existing)
struct DateFormatters {
    static let iso8601: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        return formatter
    }()

    static let iso8601Full: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()

    static let displayFull: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateStyle = .full
        return formatter
    }()

    static let displayMedium: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateStyle = .medium
        return formatter
    }()

    static let displayShort: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateStyle = .short
        return formatter
    }()
}

// Add Date extension for convenience
extension Date {
    var iso8601String: String {
        return DateFormatters.iso8601.string(from: self)
    }

    var displayString: String {
        return DateFormatters.displayMedium.string(from: self)
    }

    static func from(iso8601String: String) -> Date? {
        return DateFormatters.iso8601.date(from: iso8601String)
    }
}
```

**Usage:**
```swift
// Before:
let today = DateFormatters.iso8601.string(from: Date())

// After:
let today = Date().iso8601String
```

**Benefits:**
- More convenient date handling
- Consistent date formatting across app

---

## 6. Network Request Builder Pattern

### Problem
URL building and query parameter handling is repetitive.

### Solution
Create a request builder:

```swift
// Utils/RequestBuilder.swift
struct RequestBuilder {
    private let baseURL: URL

    init(baseURL: URL) {
        self.baseURL = baseURL
    }

    func path(_ path: String) -> RequestBuilder {
        // Returns new builder with appended path
        return RequestBuilder(baseURL: baseURL.appendingPathComponent(path))
    }

    func query(_ items: [URLQueryItem]) -> RequestBuilder {
        // Returns new builder with query items
        // Implementation details...
    }

    func build() -> Result<URL, APIService.APIError> {
        // Builds final URL
    }
}
```

**Note:** This might be overkill given the existing `buildURL` method, but could be useful if you want a fluent API.

---

## 7. Snippet Loading Protocol

### Problem
Multiple ViewModels load snippets with identical code:
- `QuizViewModel.getSnippets(questionId:)`
- `DailyViewModel.getSnippetsForQuestion(questionId:)`
- `StoryViewModel.getSnippets(storyId:)`

### Solution
Create a protocol extension:

```swift
// Utils/SnippetLoading.swift
protocol SnippetLoading {
    var apiService: APIService { get }
    var snippets: [Snippet] { get set }
    var cancellables: Set<AnyCancellable> { get set }
}

extension SnippetLoading {
    func loadSnippets(questionId: Int? = nil, storyId: Int? = nil) {
        apiService.getSnippets(
            sourceLang: nil,
            targetLang: nil,
            storyId: storyId,
            query: nil,
            level: nil
        )
        .receive(on: DispatchQueue.main)
        .sink(
            receiveCompletion: { _ in },
            receiveValue: { [weak self] snippetList in
                self?.snippets = snippetList.snippets
            }
        )
        .store(in: &cancellables)
    }
}
```

**Benefits:**
- Eliminates ~8 lines per ViewModel
- Consistent snippet loading behavior

---

## 8. Error Display Helper

### Problem
Error messages are displayed inconsistently across views.

### Solution
Create a reusable error display component:

```swift
// Views/ErrorDisplay.swift
struct ErrorDisplay: View {
    let error: APIService.APIError?
    let onDismiss: (() -> Void)?

    init(error: APIService.APIError?, onDismiss: (() -> Void)? = nil) {
        self.error = error
        self.onDismiss = onDismiss
    }

    var body: some View {
        if let error = error {
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundColor(.red)
                    Text("Error")
                        .font(.headline)
                    Spacer()
                    if let onDismiss = onDismiss {
                        Button(action: onDismiss) {
                            Image(systemName: "xmark.circle.fill")
                        }
                    }
                }
                Text(error.localizedDescription)
                    .font(.subheadline)
            }
            .padding()
            .background(Color.red.opacity(0.1))
            .cornerRadius(8)
            .padding(.horizontal)
        }
    }
}
```

**Usage:**
```swift
// In any view:
ErrorDisplay(error: viewModel.error) {
    viewModel.error = nil
}
```

**Benefits:**
- Consistent error UI across app
- Reusable component

---

## 9. Loading State Helper

### Problem
Loading indicators are implemented inconsistently.

### Solution
Create a reusable loading view:

```swift
// Views/LoadingView.swift
struct LoadingView: View {
    let message: String?

    init(message: String? = nil) {
        self.message = message
    }

    var body: some View {
        VStack(spacing: 16) {
            ProgressView()
                .scaleEffect(1.5)
            if let message = message {
                Text(message)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color(.systemBackground).opacity(0.8))
    }
}
```

**Usage:**
```swift
if viewModel.isLoading {
    LoadingView(message: "Loading questions...")
}
```

**Benefits:**
- Consistent loading UI
- Easy to customize

---

## 10. Combine Helpers for Common Patterns

### Problem
Common Combine patterns are repeated.

### Solution
Add more Combine extensions:

```swift
// Utils/CombineHelpers.swift
extension Publisher {
    // Assign to published property with automatic weak self
    func assignWeak<T: AnyObject>(
        to keyPath: ReferenceWritableKeyPath<T, Output>,
        on object: T
    ) -> AnyCancellable {
        return sink { [weak object] value in
            object?[keyPath: keyPath] = value
        }
    }

    // Assign to optional published property
    func assignOptional<T: AnyObject>(
        to keyPath: ReferenceWritableKeyPath<T, Output?>,
        on object: T
    ) -> AnyCancellable {
        return sink { [weak object] value in
            object?[keyPath: keyPath] = value
        }
    }
}
```

---

## Implementation Priority

### High Priority (Immediate Impact)
1. **BaseViewModel** - Eliminates most boilerplate
2. **Publisher Extension for Error Handling** - Reduces repetitive code
3. **APIService Refactoring** - Major code reduction

### Medium Priority (Good ROI)
4. **Question Actions Protocol** - Eliminates duplication
5. **Snippet Loading Protocol** - Consistent behavior
6. **Date Utilities Enhancement** - Better DX

### Low Priority (Nice to Have)
7. **Error Display Helper** - UI consistency
8. **Loading State Helper** - UI consistency
9. **Combine Helpers** - Advanced patterns

---

## Migration Strategy

1. **Phase 1:** Create `BaseViewModel` and migrate one ViewModel as a test
2. **Phase 2:** Add publisher extensions and update all ViewModels
3. **Phase 3:** Refactor `APIService` using generic methods
4. **Phase 4:** Add protocol extensions for shared behaviors
5. **Phase 5:** Add UI helpers and polish

---

## Estimated Code Reduction

- **BaseViewModel:** ~15-20 lines per ViewModel × 11 ViewModels = **165-220 lines**
- **Publisher Extensions:** ~5-7 lines per API call × ~50 calls = **250-350 lines**
- **APIService Refactoring:** ~20-30 lines per method × ~30 methods = **600-900 lines**
- **Question Actions:** ~30 lines × 2 ViewModels = **60 lines**
- **Snippet Loading:** ~8 lines × 3 ViewModels = **24 lines**

**Total Estimated Reduction: ~1,000-1,500 lines of code**

---

## Testing Considerations

- Ensure all existing tests still pass after refactoring
- Add tests for new base classes and extensions
- Verify error handling behavior is unchanged
- Test loading state management

---

## Notes

- Some suggestions require careful consideration of `cancellables` access
- Protocol extensions may need associated type constraints
- Consider Swift 5.9+ features like macros for further code generation
- Keep backward compatibility during migration

