import Combine
import SwiftUI

struct BadgeView: View {
    let text: String
    let color: Color

    var body: some View {
        Text(text)
            .font(.caption2)
            .fontWeight(.bold)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(color.opacity(0.2))
            .foregroundColor(color)
            .cornerRadius(8)
    }
}

import AVFoundation

@MainActor class TTSSynthesizerManager: NSObject, ObservableObject {
    static let shared = TTSSynthesizerManager()
    private var player: AVPlayer?
    private var cancellables = Set<AnyCancellable>()

    // Global preferred voice
    var preferredVoice: String?

    @Published var currentlySpeakingText: String?
    @Published var errorMessage: String?

    override init() {
        super.init()
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

        print("Requested TTS for text: \"\(text.prefix(20))...\", language: \(language), voice: \(effectiveVoice)")

        // Configure AVAudioSession for background playback
        do {
            try AVAudioSession.sharedInstance().setCategory(.playback, mode: .default, options: [])
            try AVAudioSession.sharedInstance().setActive(true)
        } catch {
            print("Failed to configure AVAudioSession: \(error)")
        }

        // Try backend TTS
        let request = TTSRequest(input: text, voice: effectiveVoice, responseFormat: "mp3", speed: 1.0)
        APIService.shared.initializeTTSStream(request: request)
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { [weak self] completion in
                if case .failure(let error) = completion {
                    print("Backend TTS initialization failed: \(error).")
                    self?.handleError("Failed to initialize audio: \(error.localizedDescription)")
                }
            }, receiveValue: { [weak self] response in
                print("Backend TTS initialized. Stream ID: \(response.streamId), Token: \(response.token ?? "none")")
                self?.playStream(streamId: response.streamId, token: response.token)
            })
            .store(in: &cancellables)
    }

    private func playStream(streamId: String, token: String?) {
        let url = APIService.shared.streamURL(for: streamId, token: token)
        print("Downloading TTS audio from URL: \(url.absoluteString)")

        // Create request with authentication cookies
        var request = URLRequest(url: url)
        request.httpShouldHandleCookies = true
        request.cachePolicy = .reloadIgnoringLocalCacheData

        // Download the complete audio data
        URLSession.shared.dataTask(with: request) { [weak self] data, response, error in
            guard let self = self else { return }

            if let error = error {
                print("Failed to download TTS audio: \(error.localizedDescription)")
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
                print("Server returned status \(httpResponse.statusCode)")
                DispatchQueue.main.async {
                    self.handleError("Server error \(httpResponse.statusCode)")
                }
                return
            }

            guard let audioData = data, !audioData.isEmpty else {
                print("Received empty audio data")
                DispatchQueue.main.async {
                    self.handleError("No audio data received.")
                }
                return
            }

            print("Downloaded \(audioData.count) bytes of audio data. Playing...")

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
            NotificationCenter.default.addObserver(forName: .AVPlayerItemDidPlayToEndTime, object: playerItem, queue: .main) { [weak self] _ in
                Task { @MainActor in
                    self?.currentlySpeakingText = nil
                    // Clean up temp file
                    try? FileManager.default.removeItem(at: tempFile)
                }
            }

            // Listen for errors
            NotificationCenter.default.addObserver(forName: .AVPlayerItemFailedToPlayToEndTime, object: playerItem, queue: .main) { [weak self] _ in
                print("AVPlayerItem failed to play to end.")
                self?.handleError("Audio playback failed.")
                try? FileManager.default.removeItem(at: tempFile)
            }

            let player = AVPlayer(playerItem: playerItem)
            player.automaticallyWaitsToMinimizeStalling = false
            self.player = player

            // Add observer for status
            playerItem.addObserver(self, forKeyPath: "status", options: [.new, .initial], context: nil)

            player.play()
            print("Audio playback started")
        } catch {
            print("Failed to save or play audio: \(error.localizedDescription)")
            handleError("Playback error: \(error.localizedDescription)")
        }
    }

    override func observeValue(forKeyPath keyPath: String?, of object: Any?, change: [NSKeyValueChangeKey : Any]?, context: UnsafeMutableRawPointer?) {
        if keyPath == "status", let playerItem = object as? AVPlayerItem {
            if playerItem.status == .failed {
                if let error = playerItem.error {
                    print("AVPlayerItem status failed: \(error.localizedDescription).")
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

    private func defaultVoiceForLanguage(_ lang: String) -> String {
        switch lang.lowercased() {
        case "italian", "it": return "it-IT-IsabellaNeural"
        case "spanish", "es": return "es-ES-ElviraNeural"
        case "french", "fr": return "fr-FR-DeniseNeural"
        case "german", "de": return "de-DE-KatjaNeural"
        case "english", "en": return "en-US-JennyNeural"
        case "portuguese": return "pt-PT-RaquelNeural"
        case "russian": return "ru-RU-DariyaNeural"
        case "japanese": return "ja-JP-NanamiNeural"
        case "korean": return "ko-KR-SunHiNeural"
        case "chinese": return "zh-CN-XiaoxiaoNeural"
        case "hindi": return "hi-IN-SwaraNeural"
        default: return "en-US-JennyNeural"
        }
    }

    func stop() {
        player?.pause()
        player = nil
        currentlySpeakingText = nil
        cancellables.removeAll()

        do {
            try AVAudioSession.sharedInstance().setActive(false, options: .notifyOthersOnDeactivation)
        } catch {
            print("Failed to deactivate AVAudioSession: \(error)")
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
        .buttonStyle(.plain) // Prevent multi-action triggers in Lists
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
                BadgeView(text: "\(snippet.sourceLanguage?.uppercased() ?? "??") → \(snippet.targetLanguage?.uppercased() ?? "??")", color: .blue)
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
