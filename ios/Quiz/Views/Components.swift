import AVFoundation
import Combine
import MarkdownUI
import MediaPlayer
import SwiftUI
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

@MainActor class TTSSynthesizerManager: NSObject, ObservableObject {
    static let shared = TTSSynthesizerManager()
    private var player: AVPlayer?
    private var cancellables = Set<AnyCancellable>()
    private var notificationObservers: [NSObjectProtocol] = []
    private var currentDataTask: URLSessionDataTask?

    // Global preferred voice
    var preferredVoice: String?

    // Cache of language name/code -> default voice mappings from server
    private var defaultVoiceCache: [String: String] = [:]

    @Published var currentlySpeakingText: String?
    @Published var errorMessage: String?

    override init() {
        super.init()
        setupRemoteCommandCenter()
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

        commandCenter.playCommand.addTarget { [weak self] _ in
            Task { @MainActor in
                self?.player?.play()
            }
            return .success
        }

        commandCenter.pauseCommand.addTarget { [weak self] _ in
            Task { @MainActor in
                self?.player?.pause()
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
        if currentlySpeakingText == text {
            stop()
            return
        }

        stop()
        currentlySpeakingText = text
        errorMessage = nil

        // Use provided voice, then preferred voice, then default for language
        let effectiveVoice: String
        if let provided = voiceIdentifier, !provided.isEmpty {
            effectiveVoice = provided
        } else if let preferred = preferredVoice, !preferred.isEmpty {
            effectiveVoice = preferred
        } else {
            effectiveVoice = defaultVoiceForLanguage(language)
        }

        // Configure AVAudioSession for background playback
        do {
            try AVAudioSession.sharedInstance().setCategory(.playback, mode: .default, options: [])
            try AVAudioSession.sharedInstance().setActive(true)
        } catch {
            handleError("Failed to configure audio session: \(error.localizedDescription)")
        }

        // Try backend TTS
        let request = TTSRequest(
            input: text, voice: effectiveVoice, responseFormat: "mp3", speed: 1.0)
        APIService.shared.initializeTTSStream(request: request)
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { [weak self] completion in
                    if case .failure(let error) = completion {
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

        // Create request with authentication cookies
        var request = URLRequest(url: url)
        request.httpShouldHandleCookies = true
        request.cachePolicy = .reloadIgnoringLocalCacheData

        // Download the complete audio data
        let dataTask = URLSession.shared.dataTask(with: request) {
            [weak self] data, response, error in
            guard let self = self else { return }

            if let error = error {
                DispatchQueue.main.async {
                    self.handleError("Network error: \(error.localizedDescription)")
                }
                return
            }

            guard let httpResponse = response as? HTTPURLResponse else {
                DispatchQueue.main.async {
                    self.handleError("Invalid server response.")
                }
                return
            }

            guard (200...299).contains(httpResponse.statusCode) else {
                DispatchQueue.main.async {
                    self.handleError("Server error \(httpResponse.statusCode)")
                }
                return
            }

            guard let audioData = data, !audioData.isEmpty else {
                DispatchQueue.main.async {
                    self.handleError("No audio data received.")
                }
                return
            }

            // Play the audio data on the main thread
            DispatchQueue.main.async {
                self.playAudioData(audioData)
            }
        }
        currentDataTask = dataTask
        dataTask.resume()
    }

    private func playAudioData(_ data: Data) {
        do {
            // Write to temporary file
            let tempDir = FileManager.default.temporaryDirectory
            let tempFile = tempDir.appendingPathComponent(UUID().uuidString + ".mp3")
            try data.write(to: tempFile)

            // Create player with local file
            let asset = AVURLAsset(url: tempFile)
            let playerItem = AVPlayerItem(asset: asset)

            // Listen for completion
            let completionObserver = NotificationCenter.default.addObserver(
                forName: .AVPlayerItemDidPlayToEndTime, object: playerItem, queue: .main
            ) { [weak self] _ in
                Task { @MainActor [weak self] in
                    guard let self = self else { return }
                    self.currentlySpeakingText = nil
                    self.clearNowPlayingInfo()
                    // Clean up temp file
                    do {
                        try FileManager.default.removeItem(at: tempFile)
                    } catch {
                        print("Warning: Failed to remove temp file: \(error.localizedDescription)")
                    }
                }
            }
            notificationObservers.append(completionObserver)

            // Listen for errors
            let errorObserver = NotificationCenter.default.addObserver(
                forName: .AVPlayerItemFailedToPlayToEndTime, object: playerItem, queue: .main
            ) { [weak self] _ in
                Task { @MainActor [weak self] in
                    guard let self = self else { return }
                    self.handleError("Audio playback failed.")
                    self.clearNowPlayingInfo()
                    do {
                        try FileManager.default.removeItem(at: tempFile)
                    } catch {
                        print(
                            "Warning: Failed to remove temp file after error: \(error.localizedDescription)"
                        )
                    }
                }
            }
            notificationObservers.append(errorObserver)

            let player = AVPlayer(playerItem: playerItem)
            player.automaticallyWaitsToMinimizeStalling = false
            self.player = player

            // Add observer for status (will be removed in stop() or deinit)
            playerItem.addObserver(
                self, forKeyPath: "status", options: [.new, .initial], context: nil)

            player.play()

            // Update Now Playing info for lock screen
            Task {
                await updateNowPlayingInfo(for: playerItem)
            }
        } catch {
            handleError("Playback error: \(error.localizedDescription)")
        }
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
    }

    override func observeValue(
        forKeyPath keyPath: String?, of object: Any?, change: [NSKeyValueChangeKey: Any]?,
        context: UnsafeMutableRawPointer?
    ) {
        if keyPath == "status", let playerItem = object as? AVPlayerItem {
            if playerItem.status == .failed {
                if let error = playerItem.error {
                    handleError("Failed to load audio: \(error.localizedDescription)")
                } else {
                    handleError("Failed to load audio stream.")
                }
            }
        }
    }

    private func handleError(_ message: String) {
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
        print(
            "⚠️ TTS Warning: Default voice cache not populated for language '\(lang)'. Using English fallback. This may indicate TTS initialization failed."
        )
        return "en-US-JennyNeural"
    }

    func stop() {
        // Cancel any ongoing network requests
        currentDataTask?.cancel()
        currentDataTask = nil

        // Remove KVO observer from player item
        if let playerItem = player?.currentItem {
            playerItem.removeObserver(self, forKeyPath: "status")
        }

        player?.pause()
        player = nil
        currentlySpeakingText = nil
        cancellables.removeAll()
        clearNowPlayingInfo()

        // Remove all notification observers
        for observer in notificationObservers {
            NotificationCenter.default.removeObserver(observer)
        }
        notificationObservers.removeAll()

        do {
            try AVAudioSession.sharedInstance().setActive(
                false, options: .notifyOthersOnDeactivation)
        } catch {}
    }

    deinit {
        // Remove KVO observer if still present
        if let playerItem = player?.currentItem {
            playerItem.removeObserver(self, forKeyPath: "status")
        }

        // Remove all notification observers
        for observer in notificationObservers {
            NotificationCenter.default.removeObserver(observer)
        }
    }
}

struct TTSButton: View {
    let text: String
    let language: String
    var voiceIdentifier: String? = nil
    @StateObject private var ttsManager = TTSSynthesizerManager.shared

    var isSpeaking: Bool {
        ttsManager.currentlySpeakingText == text
    }

    var body: some View {
        Button(action: {
            ttsManager.speak(text, language: language, voiceIdentifier: voiceIdentifier)
        }) {
            Image(systemName: isSpeaking ? "stop.circle.fill" : "speaker.wave.2.circle.fill")
                .font(.title2)
                .foregroundColor(.blue)
        }
        .buttonStyle(.plain)  // Prevent multi-action triggers in Lists
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
                        "\(snippet.sourceLanguage?.uppercased() ?? "??") → \(snippet.targetLanguage?.uppercased() ?? "??")",
                    color: .blue)
                if let level = snippet.difficultyLevel {
                    BadgeView(text: level, color: .green)
                }
                Spacer()
                HStack(spacing: 15) {
                    Button(action: {
                        onNavigateToSnippets?(snippet.originalText)
                    }) {
                        Image(systemName: "arrow.up.right.square")
                            .foregroundColor(.blue)
                    }
                    Button(action: {
                        onDelete?()
                    }) {
                        Image(systemName: "trash")
                            .foregroundColor(.red)
                    }
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
            let textContainer = textLayoutManager.textContainer
        {
            let containerSize = CGSize(width: width, height: .greatestFiniteMagnitude)
            textContainer.size = containerSize

            textLayoutManager.textViewportLayoutController.layoutViewport()

            var totalHeight: CGFloat = 0
            let documentRange = textContentManager.documentRange
            textLayoutManager.enumerateTextLayoutFragments(from: documentRange.location) {
                fragment in
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

    private func stringValue(_ v: JSONValue?) -> String? {
        guard let v else { return nil }
        if case .string(let s) = v { return s }
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
                ?? stringValue(question.content["prompt"])
            {
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
                let targetWord = stringValue(question.content["question"])
            {
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

    private func stringArrayValue(_ v: JSONValue?) -> [String]? {
        guard let v else { return nil }
        guard case .array(let arr) = v else { return nil }
        let strings = arr.compactMap { item -> String? in
            guard case .string(let s) = item else { return nil }
            return s
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
    var padding: CGFloat? = nil

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
                Button(action: { showPassword.toggle() }) {
                    Image(systemName: showPassword ? "eye.slash" : "eye")
                        .foregroundColor(.secondary)
                }
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
                    "Choose how often you want to see this question in future quizzes: 1–2 show it more, 3 no change, 4–5 show it less."
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
