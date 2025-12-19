import AVFoundation
import Combine
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
            handleError("Failed to configure audio session.")
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
        URLSession.shared.dataTask(with: request) { [weak self] data, response, error in
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
        }.resume()
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
            NotificationCenter.default.addObserver(
                forName: .AVPlayerItemDidPlayToEndTime, object: playerItem, queue: .main
            ) { _ in
                Task { @MainActor [weak self] in
                    guard let self = self else { return }
                    self.currentlySpeakingText = nil
                    self.clearNowPlayingInfo()
                    // Clean up temp file
                    try? FileManager.default.removeItem(at: tempFile)
                }
            }

            // Listen for errors
            NotificationCenter.default.addObserver(
                forName: .AVPlayerItemFailedToPlayToEndTime, object: playerItem, queue: .main
            ) { _ in
                Task { @MainActor [weak self] in
                    guard let self = self else { return }
                    self.handleError("Audio playback failed.")
                    self.clearNowPlayingInfo()
                    try? FileManager.default.removeItem(at: tempFile)
                }
            }

            let player = AVPlayer(playerItem: playerItem)
            player.automaticallyWaitsToMinimizeStalling = false
            self.player = player

            // Add observer for status
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
            self.errorMessage = message
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
        player?.pause()
        player = nil
        currentlySpeakingText = nil
        cancellables.removeAll()
        clearNowPlayingInfo()

        do {
            try AVAudioSession.sharedInstance().setActive(
                false, options: .notifyOthersOnDeactivation)
        } catch {}
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
                    Image(systemName: "arrow.up.right.square")
                        .foregroundColor(.blue)
                    Image(systemName: "trash")
                        .foregroundColor(.red)
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

        let layoutManager = self.layoutManager
        let textContainer = self.textContainer

        let width = bounds.width > 0 ? bounds.width : UIScreen.main.bounds.width - 64
        textContainer.size = CGSize(width: width, height: .greatestFiniteMagnitude)

        layoutManager.ensureLayout(for: textContainer)
        let usedRect = layoutManager.usedRect(for: textContainer)
        let height = ceil(usedRect.height)

        return CGSize(width: UIView.noIntrinsicMetric, height: height)
    }

    override func layoutSubviews() {
        super.layoutSubviews()
        invalidateIntrinsicContentSize()
    }
}

struct MarkdownTextView: UIViewRepresentable {
    let markdown: String
    let font: UIFont
    let textColor: UIColor

    init(
        markdown: String, font: UIFont = UIFont.preferredFont(forTextStyle: .body),
        textColor: UIColor = .label
    ) {
        self.markdown = markdown
        self.font = font
        self.textColor = textColor
    }

    func makeUIView(context: Context) -> UITextView {
        let textView = SizingTextView()
        textView.isEditable = false
        textView.isSelectable = true
        textView.backgroundColor = .clear
        textView.textContainerInset = .zero
        textView.textContainer.lineFragmentPadding = 0
        textView.textContainer.widthTracksTextView = true
        textView.textContainer.heightTracksTextView = false
        textView.isScrollEnabled = false
        textView.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        textView.setContentCompressionResistancePriority(.required, for: .vertical)
        updateTextView(textView)
        return textView
    }

    func updateUIView(_ uiView: UITextView, context: Context) {
        updateTextView(uiView)
        DispatchQueue.main.async {
            uiView.layoutIfNeeded()
            uiView.invalidateIntrinsicContentSize()
        }
    }

    private func updateTextView(_ textView: UITextView) {
        guard !markdown.isEmpty else {
            textView.attributedText = nil
            textView.text = ""
            textView.invalidateIntrinsicContentSize()
            return
        }

        let mutableAttributedString: NSMutableAttributedString
        if let attributedString = try? AttributedString(markdown: markdown) {
            let nsAttributedString = NSAttributedString(attributedString)
            mutableAttributedString = NSMutableAttributedString(attributedString: nsAttributedString)
        } else {
            mutableAttributedString = NSMutableAttributedString(string: markdown)
        }

        let fullRange = NSRange(location: 0, length: mutableAttributedString.length)
        mutableAttributedString.addAttribute(.font, value: font, range: fullRange)
        mutableAttributedString.addAttribute(.foregroundColor, value: textColor, range: fullRange)

        let string = mutableAttributedString.string as NSString
        let hasDoubleNewlines = markdown.contains("\n\n")

        string.enumerateSubstrings(in: NSRange(location: 0, length: string.length), options: [.byParagraphs, .localized]) { _, paragraphRange, enclosingRange, stop in
            var effectiveRange = NSRange()
            let existingStyle = mutableAttributedString.attribute(.paragraphStyle, at: paragraphRange.location, longestEffectiveRange: &effectiveRange, in: paragraphRange) as? NSParagraphStyle

            let paragraphStyle: NSMutableParagraphStyle
            if let existingStyle = existingStyle {
                paragraphStyle = existingStyle.mutableCopy() as! NSMutableParagraphStyle
            } else {
                paragraphStyle = NSMutableParagraphStyle()
            }
            paragraphStyle.lineSpacing = 4

            let isLastParagraph = (paragraphRange.location + paragraphRange.length >= string.length)
            let hasTrailingSeparator = paragraphRange.location + paragraphRange.length < enclosingRange.location + enclosingRange.length

            if hasDoubleNewlines && hasTrailingSeparator && !isLastParagraph {
                paragraphStyle.paragraphSpacing = 16
            } else if hasTrailingSeparator && !isLastParagraph {
                paragraphStyle.paragraphSpacing = 8
            } else {
                paragraphStyle.paragraphSpacing = 0
            }

            mutableAttributedString.addAttribute(.paragraphStyle, value: paragraphStyle, range: paragraphRange)
        }

        textView.attributedText = mutableAttributedString
    }
}
