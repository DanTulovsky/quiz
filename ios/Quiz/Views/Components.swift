import AVFoundation
import Combine
import MarkdownUI
import MediaPlayer
import SwiftUI
// swiftlint:disable file_length
import UIKit

struct BadgeView: View {
    let text: String
    let color: Color

    var body: some View {
        Text(text)
            .font(AppTheme.Typography.badgeFont)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(color.opacity(0.2))
            .foregroundColor(color)
            .cornerRadius(AppTheme.CornerRadius.badge)
    }
}

// swiftlint:disable:next type_body_length
@MainActor class TTSSynthesizerManager: NSObject, ObservableObject {
    static let shared = TTSSynthesizerManager()
    private var player: AVPlayer?
    private var cancellables = Set<AnyCancellable>()
    private var notificationObservers: [NSObjectProtocol] = []
    private var currentDataTask: URLSessionDataTask?
    private var nowPlayingUpdateTimer: Timer?
    private var backgroundTaskID: UIBackgroundTaskIdentifier = .invalid
    private lazy var backgroundURLSession: URLSession = {
        let config = URLSessionConfiguration.default
        config.allowsCellularAccess = true
        config.waitsForConnectivity = true
        config.timeoutIntervalForRequest = 60
        config.timeoutIntervalForResource = 300
        return URLSession(configuration: config)
    }()

    // Global preferred voice
    var preferredVoice: String?

    // Cache of language name/code -> default voice mappings from server
    private var defaultVoiceCache: [String: String] = [:]

    @Published var currentlySpeakingText: String?
    @Published var errorMessage: String?
    @Published var isPaused: Bool = false
    @Published var isLoading: Bool = false

    override init() {
        super.init()
        setupAudioSession()
        setupRemoteCommandCenter()
        setupAppLifecycleObservers()
    }

    private func setupAppLifecycleObservers() {
        // Observe when app goes to background
        let willResignActiveObserver = NotificationCenter.default.addObserver(
            forName: UIApplication.willResignActiveNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.handleAppGoingToBackground()
            }
        }
        notificationObservers.append(willResignActiveObserver)

        // Observe when app fully enters background state
        let didEnterBackgroundObserver = NotificationCenter.default.addObserver(
            forName: UIApplication.didEnterBackgroundNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.handleAppEnteringBackground()
            }
        }
        notificationObservers.append(didEnterBackgroundObserver)
    }

    private func handleAppGoingToBackground() {
        // Don't cancel downloads - allow them to continue in background
        // The network request will continue, and once the audio data is downloaded,
        // playback will start even if the phone is locked
        // Don't pause playing audio - let it continue in background
        // The background audio mode allows playback to continue
    }

    private func handleAppEnteringBackground() {
        // Audio session should remain active with proper configuration
        // No need to reactivate - it should stay active for background playback
    }

    private func setupAudioSession() {
        do {
            let audioSession = AVAudioSession.sharedInstance()

            // Use .playback category for background audio playback (enables autorun mode)
            // Use .default mode - .spokenAudio can sometimes cause issues with background playback
            // With .playback category and background mode enabled, audio should continue when device locks
            // Options are optional - .playback category alone enables autorun behavior
            // Don't deactivate first - just configure and activate.
            // Deactivation can cause iOS to not maintain the session.
            try audioSession.setCategory(
                .playback,
                mode: .default,
                options: []
            )

            // Activate the session - this puts it in autorun mode
            // With .playback category, it will stay active for background playback
            // The session should remain active and not be deactivated when device locks
            try audioSession.setActive(true)

            // Observe audio session interruptions
            setupAudioSessionInterruptionObserver()
        } catch {
            // Try fallback configuration without mode
            do {
                let audioSession = AVAudioSession.sharedInstance()
                try audioSession.setCategory(.playback, options: [])
                try audioSession.setActive(true)
            } catch {
                // Fallback configuration also failed
            }
        }
    }

    private func setupAudioSessionInterruptionObserver() {
        let interruptionObserver = NotificationCenter.default.addObserver(
            forName: AVAudioSession.interruptionNotification,
            object: nil,
            queue: .main
        ) { [weak self] notification in
            // Extract values from notification before entering Task to avoid Sendable warning
            let userInfo = notification.userInfo
            let typeValue = userInfo?[AVAudioSessionInterruptionTypeKey] as? UInt
            let type = typeValue.map { AVAudioSession.InterruptionType(rawValue: $0) }
            let optionsValue = userInfo?[AVAudioSessionInterruptionOptionKey] as? UInt
            let options = optionsValue.map { AVAudioSession.InterruptionOptions(rawValue: $0) }

            Task { @MainActor [weak self] in
                guard let self = self, let type = type else {
                    return
                }

                switch type {
                case .began:
                    // Interruption began - don't pause, let it handle naturally
                    break
                case .ended:
                    // Interruption ended - resume if we should be playing
                    if let options = options, options.contains(.shouldResume) {
                        // Resume playback if we were playing
                        if let player = self.player,
                           !self.isPaused,
                           self.currentlySpeakingText != nil {
                            do {
                                try AVAudioSession.sharedInstance().setActive(true)
                                player.play()
                            } catch {
                                // Failed to resume after interruption
                            }
                        }
                    }
                case .none:
                    // No interruption
                    break
                @unknown default:
                    break
                }
            }
        }
        notificationObservers.append(interruptionObserver)
    }

    func updateDefaultVoiceCache(languages: [LanguageInfo]) {
        for lang in languages {
            if let voice = lang.ttsVoice {
                defaultVoiceCache[lang.name.lowercased()] = voice
                defaultVoiceCache[lang.code.lowercased()] = voice
            }
        }
    }

    private func setupRemoteCommandCenter() {
        let commandCenter = MPRemoteCommandCenter.shared()

        // Enable commands
        commandCenter.playCommand.isEnabled = true
        commandCenter.pauseCommand.isEnabled = true
        commandCenter.stopCommand.isEnabled = true

        commandCenter.playCommand.addTarget { [weak self] _ in
            Task { @MainActor in
                self?.resume()
            }
            return .success
        }

        commandCenter.pauseCommand.addTarget { [weak self] _ in
            Task { @MainActor in
                self?.pause()
            }
            return .success
        }

        commandCenter.stopCommand.addTarget { [weak self] _ in
            Task { @MainActor in
                self?.stop()
            }
            return .success
        }
    }

    func speak(_ text: String, language: String, voiceIdentifier: String? = nil) {
        // If same text is already loaded, toggle pause/resume or cancel if loading
        if currentlySpeakingText == text {
            if isLoading {
                // Cancel loading
                cancel()
                return
            }
            if isPaused {
                resume()
            } else {
                pause()
            }
            return
        }

        stop()
        currentlySpeakingText = text
        errorMessage = nil
        isPaused = false
        isLoading = true

        // Use provided voice, then preferred voice, then default for language
        let effectiveVoice: String
        if let provided = voiceIdentifier, !provided.isEmpty {
            effectiveVoice = provided
        } else if let preferred = preferredVoice, !preferred.isEmpty {
            effectiveVoice = preferred
        } else {
            effectiveVoice = defaultVoiceForLanguage(language)
        }

        // Audio session should already be configured and active from setupAudioSession() in init()
        // Per Apple docs: configure once before starting playback, not repeatedly

        // Try backend TTS
        let request = TTSRequest(
            input: text, voice: effectiveVoice, responseFormat: "mp3", speed: 1.0)
        APIService.shared.initializeTTSStream(request: request)
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { [weak self] completion in
                    if case .failure(let error) = completion {
                        // Ignore cancellation errors (user intentionally cancelled)
                        if case .requestFailed(let underlyingError) = error {
                            let nsError = underlyingError as NSError
                            if nsError.domain == NSURLErrorDomain
                                && nsError.code == NSURLErrorCancelled {
                                // Request was cancelled, don't show error
                                return
                            }
                        }
                        self?.isLoading = false
                        self?.handleError(
                            "Failed to initialize audio: \(error.localizedDescription)")
                    }
                },
                receiveValue: { [weak self] response in
                    self?.playStream(streamId: response.streamId, token: response.token)
                }
            )
            .store(in: &cancellables)
    }

    private func playStream(streamId: String, token: String?) {
        let url = APIService.shared.streamURL(for: streamId, token: token)
        let request = createStreamRequest(url: url)
        startBackgroundTask()

        let dataTask = backgroundURLSession.dataTask(with: request) { [weak self] data, response, error in
            guard let self = self else { return }
            Task { @MainActor in
                self.handleStreamResponse(data: data, response: response, error: error)
            }
        }
        currentDataTask = dataTask
        dataTask.resume()
    }

    private func createStreamRequest(url: URL) -> URLRequest {
        var request = URLRequest(url: url)
        request.httpShouldHandleCookies = true
        request.cachePolicy = .reloadIgnoringLocalCacheData
        return request
    }

    private func startBackgroundTask() {
        backgroundTaskID = UIApplication.shared.beginBackgroundTask { [weak self] in
            self?.endBackgroundTask()
        }
    }

    private func handleStreamResponse(data: Data?, response: URLResponse?, error: Error?) {
        if let error = error {
            handleStreamError(error: error)
            return
        }

        guard let httpResponse = response as? HTTPURLResponse else {
            DispatchQueue.main.async {
                self.isLoading = false
                self.handleError("Invalid server response.")
                self.endBackgroundTask()
            }
            return
        }

        guard (200...299).contains(httpResponse.statusCode) else {
            DispatchQueue.main.async {
                self.isLoading = false
                self.handleError("Server error \(httpResponse.statusCode)")
                self.endBackgroundTask()
            }
            return
        }

        guard let audioData = data, !audioData.isEmpty else {
            DispatchQueue.main.async {
                self.isLoading = false
                self.handleError("No audio data received.")
                self.endBackgroundTask()
            }
            return
        }

        DispatchQueue.main.async {
            self.playAudioData(audioData)
            self.endBackgroundTask()
        }
    }

    private func handleStreamError(error: Error) {
        DispatchQueue.main.async {
            let nsError = error as NSError
            if nsError.domain == NSURLErrorDomain && nsError.code == NSURLErrorCancelled {
                self.endBackgroundTask()
                return
            }
            self.isLoading = false
            self.handleError("Network error: \(error.localizedDescription)")
            self.endBackgroundTask()
        }
    }

    private func endBackgroundTask() {
        if backgroundTaskID != .invalid {
            UIApplication.shared.endBackgroundTask(backgroundTaskID)
            backgroundTaskID = .invalid
        }
    }

    private func playAudioData(_ data: Data) {
        do {
            let tempFile = try createTempAudioFile(data: data)
            let playerItem = createPlayerItem(url: tempFile)
            setupPlayerObservers(playerItem: playerItem, tempFile: tempFile)
            createPlayer(playerItem: playerItem)
        } catch {
            handleError("Failed to prepare audio: \(error.localizedDescription)")
        }
    }

    private func createTempAudioFile(data: Data) throws -> URL {
        let tempDir = FileManager.default.temporaryDirectory
        let tempFile = tempDir.appendingPathComponent(UUID().uuidString + ".mp3")
        try data.write(to: tempFile)
        return tempFile
    }

    private func createPlayerItem(url: URL) -> AVPlayerItem {
        let asset = AVURLAsset(url: url)
        return AVPlayerItem(asset: asset)
    }

    private func setupPlayerObservers(playerItem: AVPlayerItem, tempFile: URL) {
        let completionObserver = NotificationCenter.default.addObserver(
            forName: .AVPlayerItemDidPlayToEndTime, object: playerItem, queue: .main
        ) { [weak self] _ in
            Task { @MainActor [weak self] in
                guard let self = self else { return }
                self.currentlySpeakingText = nil
                self.isPaused = false
                self.clearNowPlayingInfo()
                try? FileManager.default.removeItem(at: tempFile)
            }
        }
        notificationObservers.append(completionObserver)

        let errorObserver = NotificationCenter.default.addObserver(
            forName: .AVPlayerItemFailedToPlayToEndTime, object: playerItem, queue: .main
        ) { [weak self] _ in
            Task { @MainActor [weak self] in
                guard let self = self else { return }
                self.handleError("Audio playback failed.")
                self.clearNowPlayingInfo()
                try? FileManager.default.removeItem(at: tempFile)
            }
        }
        notificationObservers.append(errorObserver)
    }

    private func createPlayer(playerItem: AVPlayerItem) {
        let player = AVPlayer(playerItem: playerItem)
        player.automaticallyWaitsToMinimizeStalling = false
        self.player = player
        playerItem.addObserver(
            self, forKeyPath: "status", options: [.new, .initial], context: nil)
        player.addObserver(
            self, forKeyPath: "timeControlStatus", options: [.new, .initial], context: nil)
        startPlayback(player: player, playerItem: playerItem)
    }

    private func startPlayback(player: AVPlayer, playerItem: AVPlayerItem) {
        player.play()
        isPaused = false

        Task { @MainActor in
            try? await Task.sleep(nanoseconds: 100_000_000)
            if player.timeControlStatus == .playing {
                isLoading = false
            }
        }

        Task {
            await updateNowPlayingInfo(for: playerItem)
        }

        startNowPlayingUpdates()
    }

    private func updateNowPlayingInfo(for playerItem: AVPlayerItem) async {
        var nowPlayingInfo = [String: Any]()

        // Set title to a preview of the text being spoken
        if let text = currentlySpeakingText {
            let preview = text.prefix(50)
            nowPlayingInfo[MPMediaItemPropertyTitle] =
                String(preview) + (text.count > 50 ? "..." : "")
        } else {
            nowPlayingInfo[MPMediaItemPropertyTitle] = "Text to Speech"
        }

        nowPlayingInfo[MPMediaItemPropertyArtist] = "Quiz"

        // Set duration if available using modern async API
        do {
            let duration = try await playerItem.asset.load(.duration)
            if duration.seconds.isFinite {
                nowPlayingInfo[MPMediaItemPropertyPlaybackDuration] = duration.seconds
            }
        } catch {
            // Duration not available, continue without it
        }

        nowPlayingInfo[MPNowPlayingInfoPropertyElapsedPlaybackTime] = 0
        nowPlayingInfo[MPNowPlayingInfoPropertyPlaybackRate] = 1.0

        await MainActor.run {
            MPNowPlayingInfoCenter.default().nowPlayingInfo = nowPlayingInfo
        }
    }

    private func clearNowPlayingInfo() {
        MPNowPlayingInfoCenter.default().nowPlayingInfo = nil
        stopNowPlayingUpdates()
    }

    private func startNowPlayingUpdates() {
        stopNowPlayingUpdates()
        nowPlayingUpdateTimer = Timer.scheduledTimer(withTimeInterval: 0.5, repeats: true) { [weak self] _ in
            guard let self = self else { return }
            Task { @MainActor in
                self.updateNowPlayingPlaybackState()
            }
        }
        // Ensure timer runs on main run loop
        if let timer = nowPlayingUpdateTimer {
            RunLoop.main.add(timer, forMode: .common)
        }
    }

    private func stopNowPlayingUpdates() {
        nowPlayingUpdateTimer?.invalidate()
        nowPlayingUpdateTimer = nil
    }

    // swiftlint:disable:next block_based_kvo
    override func observeValue(
        forKeyPath keyPath: String?, of object: Any?, change: [NSKeyValueChangeKey: Any]?,
        context: UnsafeMutableRawPointer?
    ) {
        if keyPath == "status", let playerItem = object as? AVPlayerItem {
            if playerItem.status == .failed {
                isLoading = false
                if let error = playerItem.error {
                    handleError("Failed to load audio: \(error.localizedDescription)")
                } else {
                    handleError("Failed to load audio stream.")
                }
            } else if playerItem.status == .readyToPlay, let player = player {
                // When ready, check if it's actually playing
                if player.timeControlStatus == .playing {
                    isLoading = false
                }
            }
        } else if keyPath == "timeControlStatus", let player = object as? AVPlayer {
            // Monitor player state - if it pauses unexpectedly, it may indicate a configuration issue
            // With proper configuration, the player should continue playing when device locks
            if player.timeControlStatus == .playing {
                // Update our state when player starts playing
                if isPaused && player == self.player {
                    isPaused = false
                    updateNowPlayingPlaybackState()
                }
                isLoading = false
            }
        }
    }

    private func handleError(_ message: String) {
        isLoading = false
        DispatchQueue.main.async {
            let userFriendlyMessage: String
            if message.contains("Network error") || message.contains("requestFailed") {
                userFriendlyMessage =
                    "Unable to connect to the server. Please check your internet connection and try again."
            } else if message.contains("No audio data") {
                userFriendlyMessage = "Audio data was not received. Please try again."
            } else if message.contains("Failed to load audio") {
                userFriendlyMessage = "Failed to load audio. Please try again."
            } else {
                userFriendlyMessage = "Text-to-speech error: \(message)"
            }
            self.errorMessage = userFriendlyMessage
            self.currentlySpeakingText = nil
            self.stop()
        }
    }

    func defaultVoiceForLanguage(_ lang: String) -> String {
        // Check cache from server-provided languages (must be loaded)
        let langKey = lang.lowercased()
        if let cachedVoice = defaultVoiceCache[langKey] {
            return cachedVoice
        }

        // If not in cache, return a default (this should only happen if languages haven't loaded yet)
        // In practice, this should be avoided by ensuring languages are loaded before using TTS
        return "en-US-JennyNeural"
    }

    func pause() {
        guard let player = player, currentlySpeakingText != nil else { return }
        player.pause()
        isPaused = true
        updateNowPlayingPlaybackState()
    }

    func resume() {
        guard let player = player, currentlySpeakingText != nil else { return }
        // Audio session should already be active from setupAudioSession()
        // No need to reactivate - it should stay active for background playback
        player.play()
        isPaused = false
        updateNowPlayingPlaybackState()
    }

    private func updateNowPlayingPlaybackState() {
        guard let player = player, let playerItem = player.currentItem else { return }
        var nowPlayingInfo = MPNowPlayingInfoCenter.default().nowPlayingInfo ?? [String: Any]()

        // Update playback rate (0 = paused, 1 = playing)
        nowPlayingInfo[MPNowPlayingInfoPropertyPlaybackRate] = isPaused ? 0.0 : 1.0

        // Update elapsed time
        let currentTime = CMTimeGetSeconds(playerItem.currentTime())
        if currentTime.isFinite {
            nowPlayingInfo[MPNowPlayingInfoPropertyElapsedPlaybackTime] = currentTime
        }

        MPNowPlayingInfoCenter.default().nowPlayingInfo = nowPlayingInfo
    }

    func cancel() {
        // Cancel any ongoing network requests
        currentDataTask?.cancel()
        currentDataTask = nil

        // Cancel any Combine publishers
        cancellables.removeAll()

        // Reset state
        currentlySpeakingText = nil
        isPaused = false
        isLoading = false
        errorMessage = nil
        clearNowPlayingInfo()

        // End background task if active
        endBackgroundTask()
    }

    func stop() {
        // Cancel any ongoing network requests
        currentDataTask?.cancel()
        currentDataTask = nil

        // Remove KVO observers from player and player item
        if let playerItem = player?.currentItem {
            playerItem.removeObserver(self, forKeyPath: "status")
        }
        if let player = player {
            player.removeObserver(self, forKeyPath: "timeControlStatus")
        }

        player?.pause()
        player = nil
        currentlySpeakingText = nil
        isPaused = false
        isLoading = false
        cancellables.removeAll()
        stopNowPlayingUpdates()
        clearNowPlayingInfo()

        // Remove all notification observers
        for observer in notificationObservers {
            NotificationCenter.default.removeObserver(observer)
        }
        notificationObservers.removeAll()

        // End background task if active
        endBackgroundTask()

        // Keep audio session active for background playback support
        // Only deactivate if truly necessary (e.g., app termination)
    }

    deinit {
        // Remove KVO observers if still present
        if let playerItem = player?.currentItem {
            playerItem.removeObserver(self, forKeyPath: "status")
        }
        if let player = player {
            player.removeObserver(self, forKeyPath: "timeControlStatus")
        }

        // Remove all notification observers
        for observer in notificationObservers {
            NotificationCenter.default.removeObserver(observer)
        }

        // Stop timer (can't call MainActor method in deinit, so invalidate directly)
        nowPlayingUpdateTimer?.invalidate()
        nowPlayingUpdateTimer = nil
    }
}

struct TTSButton: View {
    let text: String
    let language: String
    var voiceIdentifier: String?
    @StateObject private var ttsManager = TTSSynthesizerManager.shared
    @State private var rotation: Double = 0

    var isSpeaking: Bool {
        ttsManager.currentlySpeakingText == text
    }

    var isLoading: Bool {
        isSpeaking && ttsManager.isLoading
    }

    var iconName: String {
        if isLoading {
            return "speaker.wave.2.circle.fill"
        } else if isSpeaking {
            return ttsManager.isPaused ? "play.circle.fill" : "pause.circle.fill"
        } else {
            return "speaker.wave.2.circle.fill"
        }
    }

    var body: some View {
        Button(
            action: {
                ttsManager.speak(text, language: language, voiceIdentifier: voiceIdentifier)
            },
            label: {
                Image(systemName: iconName)
                    .font(.title2)
                    .foregroundColor(.blue)
                    .rotationEffect(.degrees(rotation))
            }
        )
        .buttonStyle(.plain)  // Prevent multi-action triggers in Lists
        .onChange(of: isLoading) { _, newValue in
            if newValue {
                withAnimation(.linear(duration: 1.5).repeatForever(autoreverses: false)) {
                    rotation = 360
                }
            } else {
                withAnimation {
                    rotation = 0
                }
            }
        }
    }
}

struct SnippetDetailView: View {
    let snippet: Snippet
    let onClose: () -> Void
    let onNavigateToSnippets: ((String) -> Void)?
    let onDelete: (() -> Void)?

    init(
        snippet: Snippet, onClose: @escaping () -> Void,
        onNavigateToSnippets: ((String) -> Void)? = nil, onDelete: (() -> Void)? = nil
    ) {
        self.snippet = snippet
        self.onClose = onClose
        self.onNavigateToSnippets = onNavigateToSnippets
        self.onDelete = onDelete
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text(snippet.translatedText)
                    .font(.headline)
                Spacer()
                Button(action: onClose) {
                    Image(systemName: "xmark")
                        .foregroundColor(.secondary)
                }
            }

            HStack {
                BadgeView(
                    text:
                        "\(snippet.sourceLanguage?.uppercased() ?? "??") → "
                        + "\(snippet.targetLanguage?.uppercased() ?? "??")",
                    color: .blue)
                if let level = snippet.difficultyLevel {
                    BadgeView(text: level, color: .green)
                }
                Spacer()
                HStack(spacing: 15) {
                    Button(
                        action: {
                            onNavigateToSnippets?(snippet.originalText)
                        },
                        label: {
                            Image(systemName: "arrow.up.right.square")
                                .foregroundColor(.blue)
                        })
                    Button(
                        action: {
                            onDelete?()
                        },
                        label: {
                            Image(systemName: "trash")
                                .foregroundColor(.red)
                        })
                }
            }

            if let context = snippet.context {
                Text("\"\(context)\"")
                    .font(.subheadline)
                    .italic()
                    .foregroundColor(.secondary)
                    .padding(.top, 4)
            }
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(12)
        .shadow(radius: 10)
        .padding()
    }
}

private class SizingTextView: UITextView {
    override var intrinsicContentSize: CGSize {
        guard !text.isEmpty, attributedText != nil else {
            return CGSize(width: UIView.noIntrinsicMetric, height: 0)
        }

        let width = bounds.width > 0 ? bounds.width : UIScreen.main.bounds.width - 64

        if let textLayoutManager = textLayoutManager,
           let textContentManager = textLayoutManager.textContentManager,
           let textContainer = textLayoutManager.textContainer {
            let containerSize = CGSize(width: width, height: .greatestFiniteMagnitude)
            textContainer.size = containerSize

            textLayoutManager.textViewportLayoutController.layoutViewport()

            var totalHeight: CGFloat = 0
            let documentRange = textContentManager.documentRange
            textLayoutManager.enumerateTextLayoutFragments(from: documentRange.location) { fragment in
                totalHeight = max(totalHeight, fragment.layoutFragmentFrame.maxY)
                return true
            }

            let height = ceil(totalHeight)
            return CGSize(width: UIView.noIntrinsicMetric, height: height)
        } else {
            let size = CGSize(width: width, height: .greatestFiniteMagnitude)
            let calculatedSize = sizeThatFits(size)
            return CGSize(width: UIView.noIntrinsicMetric, height: calculatedSize.height)
        }
    }

    override func layoutSubviews() {
        super.layoutSubviews()
        invalidateIntrinsicContentSize()
    }
}

struct MarkdownTextView: View {
    let markdown: String
    let font: UIFont
    let textColor: UIColor
    @Environment(\.fontSizeMultiplier) var fontSizeMultiplier

    init(
        markdown: String, font: UIFont = UIFont.preferredFont(forTextStyle: .body),
        textColor: UIColor = .label
    ) {
        self.markdown = markdown
        self.font = font
        self.textColor = textColor
    }

    var body: some View {
        let scaledFontSize = font.pointSize * fontSizeMultiplier
        return Markdown(markdown)
            .markdownTextStyle {
                FontSize(scaledFontSize)
                ForegroundColor(Color(textColor))
            }
            .markdownBlockStyle(\.paragraph) { configuration in
                configuration.label
                    .padding(.bottom, 8)
            }
    }
}

// MARK: - Shared Question View Components

struct QuestionCardView: View {
    let question: Question
    let snippets: [Snippet]
    let onTextSelected: (String, String) -> Void
    let onSnippetTapped: (Snippet) -> Void
    let showLanguageBadge: Bool

    // Create a stable ID based on snippet IDs to force view updates when snippets change
    // Use a more unique ID that includes snippet count and first snippet ID to ensure changes are detected
    private var snippetsId: String {
        if snippets.isEmpty {
            return "empty"
        }
        return "\(snippets.count)-\(snippets.map { "\($0.id)" }.joined(separator: ","))"
    }

    init(
        question: Question,
        snippets: [Snippet],
        onTextSelected: @escaping (String, String) -> Void,
        onSnippetTapped: @escaping (Snippet) -> Void,
        showLanguageBadge: Bool = true
    ) {
        self.question = question
        self.snippets = snippets
        self.onTextSelected = onTextSelected
        self.onSnippetTapped = onSnippetTapped
        self.showLanguageBadge = showLanguageBadge
    }

    private func stringValue(_ value: JSONValue?) -> String? {
        guard let value else { return nil }
        if case .string(let stringValue) = value { return stringValue }
        return nil
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 15) {
            HStack {
                BadgeView(
                    text: question.type.replacingOccurrences(of: "_", with: " ").uppercased(),
                    color: AppTheme.Colors.accentIndigo)
                if showLanguageBadge {
                    Spacer()
                    BadgeView(
                        text: "\(question.language.uppercased()) - \(question.level)",
                        color: AppTheme.Colors.primaryBlue)
                }
            }

            if let passage = stringValue(question.content["passage"]) {
                VStack(alignment: .trailing) {
                    TTSButton(text: passage, language: question.language)
                    SelectableTextView(
                        text: passage,
                        language: question.language,
                        onTextSelected: { text in
                            onTextSelected(text, passage)
                        },
                        highlightedSnippets: snippets,
                        onSnippetTapped: onSnippetTapped
                    )
                    .id("\(passage)-\(snippetsId)")
                    .frame(minHeight: 100)
                }
                .appInnerCard()
            }

            if let sentence = stringValue(question.content["sentence"]) {
                SelectableTextView(
                    text: sentence,
                    language: question.language,
                    onTextSelected: { text in
                        onTextSelected(text, sentence)
                    },
                    highlightedSnippets: snippets,
                    onSnippetTapped: onSnippetTapped
                )
                .id("\(sentence)-\(snippetsId)")
                .frame(minHeight: 44)
            } else if let questionText = stringValue(question.content["question"])
                        ?? stringValue(question.content["prompt"]) {
                SelectableTextView(
                    text: questionText,
                    language: question.language,
                    onTextSelected: { text in
                        onTextSelected(text, questionText)
                    },
                    highlightedSnippets: snippets,
                    onSnippetTapped: onSnippetTapped
                )
                .id("\(questionText)-\(snippetsId)")
                .frame(minHeight: 44)
            }

            if question.type == "vocabulary",
               let targetWord = stringValue(question.content["question"]) {
                let vocabText = "What does \(targetWord) mean in this context?"
                SelectableTextView(
                    text: vocabText,
                    language: question.language,
                    onTextSelected: { text in
                        onTextSelected(text, vocabText)
                    },
                    highlightedSnippets: snippets,
                    onSnippetTapped: onSnippetTapped
                )
                .id("vocab-\(targetWord)-\(snippetsId)")
                .frame(minHeight: 44)
            }
        }
        .appCard()
    }
}

struct QuestionOptionsView: View {
    let question: Question
    let selectedAnswerIndex: Int?
    let answerResponse: AnswerResponse?
    let correctAnswerIndex: Int?
    let userAnswerIndex: Int?
    let showResults: Bool
    let onOptionSelected: (Int) -> Void

    init(
        question: Question,
        selectedAnswerIndex: Int?,
        answerResponse: AnswerResponse? = nil,
        correctAnswerIndex: Int? = nil,
        userAnswerIndex: Int? = nil,
        showResults: Bool,
        onOptionSelected: @escaping (Int) -> Void
    ) {
        self.question = question
        self.selectedAnswerIndex = selectedAnswerIndex
        self.answerResponse = answerResponse
        self.correctAnswerIndex = correctAnswerIndex
        self.userAnswerIndex = userAnswerIndex
        self.showResults = showResults
        self.onOptionSelected = onOptionSelected
    }

    private func stringArrayValue(_ value: JSONValue?) -> [String]? {
        guard let value else { return nil }
        guard case .array(let arr) = value else { return nil }
        let strings = arr.compactMap { item -> String? in
            guard case .string(let stringValue) = item else { return nil }
            return stringValue
        }
        return strings.isEmpty ? nil : strings
    }

    var body: some View {
        if let options = stringArrayValue(question.content["options"]) {
            VStack(spacing: 12) {
                ForEach(Array(options.enumerated()), id: \.offset) { idx, option in
                    QuestionOptionButton(
                        option: option,
                        index: idx,
                        isSelected: selectedAnswerIndex == idx,
                        isCorrect: showResults
                            && (correctAnswerIndex ?? answerResponse?.correctAnswerIndex) == idx,
                        isUserIncorrect: showResults
                            && (userAnswerIndex ?? answerResponse?.userAnswerIndex) == idx
                            && (correctAnswerIndex ?? answerResponse?.correctAnswerIndex) != idx,
                        showResults: showResults,
                        onTap: { onOptionSelected(idx) }
                    )
                }
            }
        }
    }
}

struct QuestionOptionButton: View {
    let option: String
    let index: Int
    let isSelected: Bool
    let isCorrect: Bool
    let isUserIncorrect: Bool
    let showResults: Bool
    let onTap: () -> Void

    var body: some View {
        HStack {
            if showResults {
                if isCorrect {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundColor(AppTheme.Colors.successGreen)
                } else if isUserIncorrect {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundColor(AppTheme.Colors.errorRed)
                }
            }

            Text(option)
                .font(AppTheme.Typography.bodyFont)
                .foregroundColor(
                    isUserIncorrect
                        ? AppTheme.Colors.errorRed
                        : (isCorrect
                            ? AppTheme.Colors.successGreen
                            : (isSelected ? .white : AppTheme.Colors.primaryText))
                )
                .frame(maxWidth: .infinity, alignment: .leading)

            Spacer()
        }
        .padding(AppTheme.Spacing.innerPadding)
        .frame(maxWidth: .infinity)
        .background(
            isUserIncorrect
                ? AppTheme.Colors.errorRed.opacity(0.1)
                : (isCorrect
                    ? AppTheme.Colors.successGreen.opacity(0.1)
                    : (isSelected
                        ? AppTheme.Colors.primaryBlue : AppTheme.Colors.primaryBlue.opacity(0.05)))
        )
        .cornerRadius(AppTheme.CornerRadius.button)
        .overlay(
            RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button)
                .stroke(
                    isUserIncorrect
                        ? AppTheme.Colors.errorRed
                        : (isCorrect ? AppTheme.Colors.successGreen : AppTheme.Colors.borderBlue),
                    lineWidth: 1)
        )
        .contentShape(Rectangle())
        .onTapGesture {
            if !showResults {
                onTap()
            }
        }
        .disabled(showResults)
    }
}

struct AnswerFeedbackView: View {
    let isCorrect: Bool
    let explanation: String
    let language: String
    let snippets: [Snippet]
    let onTextSelected: (String, String) -> Void
    let onSnippetTapped: (Snippet) -> Void
    let showOverlay: Bool

    // Create a stable ID based on snippet IDs to force view updates when snippets change
    private var snippetsId: String {
        snippets.map { "\($0.id)" }.joined(separator: ",")
    }

    init(
        isCorrect: Bool,
        explanation: String,
        language: String,
        snippets: [Snippet],
        onTextSelected: @escaping (String, String) -> Void,
        onSnippetTapped: @escaping (Snippet) -> Void,
        showOverlay: Bool = false
    ) {
        self.isCorrect = isCorrect
        self.explanation = explanation
        self.language = language
        self.snippets = snippets
        self.onTextSelected = onTextSelected
        self.onSnippetTapped = onSnippetTapped
        self.showOverlay = showOverlay
    }

    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.Spacing.itemSpacing) {
            HStack {
                Image(systemName: isCorrect ? "checkmark.circle.fill" : "xmark.circle.fill")
                    .foregroundColor(
                        isCorrect ? AppTheme.Colors.successGreen : AppTheme.Colors.errorRed)
                Text(isCorrect ? "Correct!" : "Incorrect")
                    .font(AppTheme.Typography.headingFont)
                    .foregroundColor(
                        isCorrect ? AppTheme.Colors.successGreen : AppTheme.Colors.errorRed
                    )
                    .textSelection(.enabled)
            }

            SelectableTextView(
                text: explanation,
                language: language,
                onTextSelected: { text in
                    onTextSelected(text, explanation)
                },
                highlightedSnippets: snippets,
                onSnippetTapped: onSnippetTapped
            )
            .id("explanation-\(snippetsId)")
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(AppTheme.Spacing.innerPadding)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(
            isCorrect
                ? AppTheme.Colors.successGreen.opacity(0.05)
                : AppTheme.Colors.errorRed.opacity(0.05)
        )
        .cornerRadius(AppTheme.CornerRadius.button)
        .overlay(
            Group {
                if showOverlay {
                    RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button)
                        .stroke(
                            isCorrect
                                ? AppTheme.Colors.successGreen.opacity(0.2)
                                : AppTheme.Colors.errorRed.opacity(0.2),
                            lineWidth: 1
                        )
                }
            }
        )
    }
}

struct QuestionActionButtons: View {
    let isReported: Bool
    let onReport: () -> Void
    let onMarkKnown: () -> Void

    var body: some View {
        HStack(spacing: 20) {
            Button(action: onReport) {
                Label(isReported ? "Reported" : "Report issue", systemImage: "flag")
                    .font(.caption)
            }
            .disabled(isReported)
            .foregroundColor(.secondary)

            Spacer()

            Button(action: onMarkKnown) {
                Label("Adjust frequency", systemImage: "slider.horizontal.3")
                    .font(.caption)
            }
            .foregroundColor(.secondary)
        }
        .padding(.top, 10)
    }
}

struct FormTextField: View {
    let placeholder: String
    @Binding var text: String
    var autocapitalization: TextInputAutocapitalization = .never
    var autocorrection: Bool = false
    var showBorder: Bool = false
    var padding: CGFloat?

    var body: some View {
        TextField(placeholder, text: $text)
            .textInputAutocapitalization(autocapitalization)
            .autocorrectionDisabled(autocorrection)
            .padding(padding ?? AppTheme.Spacing.innerPadding)
            .background(AppTheme.Colors.secondaryBackground)
            .cornerRadius(AppTheme.CornerRadius.button)
            .overlay(
                Group {
                    if showBorder {
                        RoundedRectangle(cornerRadius: AppTheme.CornerRadius.button)
                            .stroke(AppTheme.Colors.borderGray, lineWidth: 1)
                    }
                }
            )
    }
}

struct FormSecureField: View {
    let placeholder: String
    @Binding var text: String
    var showPasswordToggle: Bool = true

    @State private var showPassword = false

    var body: some View {
        HStack {
            if showPassword {
                TextField("", text: $text)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled(true)
            } else {
                SecureField(placeholder, text: $text)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled(true)
            }

            if showPasswordToggle {
                Button(
                    action: { showPassword.toggle() },
                    label: {
                        Image(systemName: showPassword ? "eye.slash" : "eye")
                            .foregroundColor(.secondary)
                    })
            }
        }
        .padding()
        .background(AppTheme.Colors.secondaryBackground)
        .cornerRadius(AppTheme.CornerRadius.button)
    }
}

struct ModalHeader: View {
    let title: String
    let onClose: (() -> Void)?

    init(title: String, onClose: (() -> Void)? = nil) {
        self.title = title
        self.onClose = onClose
    }

    var body: some View {
        HStack {
            Text(title)
                .font(AppTheme.Typography.headingFont)
            Spacer()
            if let onClose = onClose {
                Button(action: onClose) {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundColor(.secondary)
                        .font(.title2)
                }
            }
        }
        .padding()
    }
}

struct ReportQuestionSheet: View {
    @Binding var reportReason: String
    @Binding var isPresented: Bool
    let isSubmitting: Bool
    let onSubmit: (String) -> Void

    var body: some View {
        NavigationView {
            Form {
                Section(header: Text("Why are you reporting this question?")) {
                    TextEditor(text: $reportReason)
                        .frame(minHeight: 100)
                }

                Button("Submit Report") {
                    onSubmit(reportReason)
                }
                .disabled(isSubmitting)
            }
            .navigationTitle("Report Issue")
            .navigationBarItems(trailing: Button("Cancel") { isPresented = false })
        }
    }
}

struct MarkKnownSheet: View {
    @Binding var selectedConfidence: Int?
    @Binding var isPresented: Bool
    let isSubmitting: Bool
    let onSubmit: (Int) -> Void

    var body: some View {
        NavigationView {
            VStack(spacing: 20) {
                Text(
                    "Choose how often you want to see this question in future quizzes: "
                        + "1–2 show it more, 3 no change, 4–5 show it less."
                )
                .font(.subheadline)
                .foregroundColor(.secondary)
                .padding()

                Text("How confident are you about this question?")
                    .font(.headline)

                HStack(spacing: 10) {
                    ForEach(1...5, id: \.self) { level in
                        Button("\(level)") {
                            selectedConfidence = level
                        }
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(
                            selectedConfidence == level
                                ? AppTheme.Colors.primaryBlue
                                : AppTheme.Colors.primaryBlue.opacity(0.1)
                        )
                        .foregroundColor(
                            selectedConfidence == level ? .white : AppTheme.Colors.primaryBlue
                        )
                        .cornerRadius(AppTheme.CornerRadius.button)
                    }
                }
                .padding(.horizontal)

                Spacer()

                Button("Save Preference") {
                    if let confidence = selectedConfidence {
                        onSubmit(confidence)
                    }
                }
                .buttonStyle(
                    PrimaryButtonStyle(
                        isDisabled: selectedConfidence == nil || isSubmitting)
                )
                .padding()
            }
            .navigationTitle("Adjust Frequency")
            .navigationBarItems(trailing: Button("Cancel") { isPresented = false })
        }
    }
}
