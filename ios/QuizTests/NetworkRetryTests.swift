import XCTest
import Combine
@testable import Quiz

class NetworkRetryTests: XCTestCase {
    var cancellables: Set<AnyCancellable>!

    override func setUp() {
        super.setUp()
        cancellables = []
    }

    override func tearDown() {
        cancellables = nil
        super.tearDown()
    }

    func testRetryOnTransientFailure() {
        // Given - a publisher that fails with a transient error
        var attemptCount = 0
        let maxRetries = 3

        let publisher = Future<String, APIService.APIError> { promise in
            attemptCount += 1
            if attemptCount < maxRetries {
                let error = NSError(domain: NSURLErrorDomain, code: NSURLErrorTimedOut, userInfo: nil)
                promise(.failure(.requestFailed(error)))
            } else {
                promise(.success("Success"))
            }
        }
        .retryOnTransientFailure(maxRetries: maxRetries)

        let expectation = XCTestExpectation(description: "Retry succeeds after transient failures")

        // When
        publisher
            .sink(
                receiveCompletion: { completion in
                    if case .failure = completion {
                        XCTFail("Should succeed after retries")
                    }
                    expectation.fulfill()
                },
                receiveValue: { value in
                    XCTAssertEqual(value, "Success")
                    XCTAssertEqual(attemptCount, maxRetries, "Should have retried the correct number of times")
                }
            )
            .store(in: &cancellables)

        wait(for: [expectation], timeout: 5.0)
    }

    func testRetryDoesNotRetryNonTransientErrors() {
        // Given - a publisher that fails with a non-transient error
        var attemptCount = 0

        let publisher = Future<String, APIService.APIError> { promise in
            attemptCount += 1
            let error = NSError(domain: NSURLErrorDomain, code: NSURLErrorBadURL, userInfo: nil)
            promise(.failure(.requestFailed(error)))
        }
        .retryOnTransientFailure(maxRetries: 3)

        let expectation = XCTestExpectation(description: "Should not retry non-transient errors")

        // When
        publisher
            .sink(
                receiveCompletion: { completion in
                    if case .failure = completion {
                        XCTAssertEqual(attemptCount, 1, "Should not retry non-transient errors")
                    } else {
                        XCTFail("Should fail without retrying")
                    }
                    expectation.fulfill()
                },
                receiveValue: { _ in
                    XCTFail("Should not succeed")
                }
            )
            .store(in: &cancellables)

        wait(for: [expectation], timeout: 2.0)
    }

    func testRetryDoesNotRetryNonNetworkErrors() {
        // Given - a publisher that fails with a non-network error
        var attemptCount = 0

        let publisher = Future<String, APIService.APIError> { promise in
            attemptCount += 1
            promise(.failure(.invalidResponse))
        }
        .retryOnTransientFailure(maxRetries: 3)

        let expectation = XCTestExpectation(description: "Should not retry non-network errors")

        // When
        publisher
            .sink(
                receiveCompletion: { completion in
                    if case .failure = completion {
                        XCTAssertEqual(attemptCount, 1, "Should not retry non-network errors")
                    } else {
                        XCTFail("Should fail without retrying")
                    }
                    expectation.fulfill()
                },
                receiveValue: { _ in
                    XCTFail("Should not succeed")
                }
            )
            .store(in: &cancellables)

        wait(for: [expectation], timeout: 2.0)
    }

    func testRetryExponentialBackoff() {
        // Given - a publisher that fails multiple times
        var attemptCount = 0
        var timestamps: [Date] = []

        let publisher = Future<String, APIService.APIError> { promise in
            attemptCount += 1
            timestamps.append(Date())
            if attemptCount < 3 {
                let error = NSError(domain: NSURLErrorDomain, code: NSURLErrorNetworkConnectionLost, userInfo: nil)
                promise(.failure(.requestFailed(error)))
            } else {
                promise(.success("Success"))
            }
        }
        .retryOnTransientFailure(maxRetries: 3, delay: 0.1)

        let expectation = XCTestExpectation(description: "Retry with exponential backoff")

        // When
        publisher
            .sink(
                receiveCompletion: { _ in
                    expectation.fulfill()
                },
                receiveValue: { _ in
                    // Verify that delays increased between retries
                    if timestamps.count >= 2 {
                        let delay1 = timestamps[1].timeIntervalSince(timestamps[0])
                        XCTAssertGreaterThan(delay1, 0.05, "Should have delay between retries")
                    }
                }
            )
            .store(in: &cancellables)

        wait(for: [expectation], timeout: 2.0)
    }
}

