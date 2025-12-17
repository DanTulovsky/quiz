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

@MainActor class TTSSynthesizerManager: NSObject, ObservableObject, AVSpeechSynthesizerDelegate {
    static let shared = TTSSynthesizerManager()
    private let synthesizer = AVSpeechSynthesizer()
    @Published var currentlySpeakingText: String?

    override init() {
        super.init()
        synthesizer.delegate = self
    }

    func speak(_ text: String, language: String, voiceIdentifier: String? = nil) {
        if currentlySpeakingText == text {
            stop()
            return
        }
        
        stop()
        
        // Configure AVAudioSession for background playback
        do {
            try AVAudioSession.sharedInstance().setCategory(.playback, mode: .default, options: [])
            try AVAudioSession.sharedInstance().setActive(true)
        } catch {
            print("Failed to configure AVAudioSession: \(error)")
        }
        
        let utterance = AVSpeechUtterance(string: text)
        
        if let voiceId = voiceIdentifier, !voiceIdentifier!.isEmpty, let voice = AVSpeechSynthesisVoice(identifier: voiceId) {
            utterance.voice = voice
        } else {
            utterance.voice = AVSpeechSynthesisVoice(language: mapLanguageCode(language))
        }
        
        currentlySpeakingText = text
        synthesizer.speak(utterance)
    }

    func stop() {
        synthesizer.stopSpeaking(at: .immediate)
        currentlySpeakingText = nil
        
        do {
            try AVAudioSession.sharedInstance().setActive(false, options: .notifyOthersOnDeactivation)
        } catch {
            print("Failed to deactivate AVAudioSession: \(error)")
        }
    }

    func speechSynthesizer(_ synthesizer: AVSpeechSynthesizer, didFinish utterance: AVSpeechUtterance) {
        DispatchQueue.main.async {
            self.currentlySpeakingText = nil
        }
    }

    func speechSynthesizer(_ synthesizer: AVSpeechSynthesizer, didCancel utterance: AVSpeechUtterance) {
        DispatchQueue.main.async {
            self.currentlySpeakingText = nil
        }
    }

    private func mapLanguageCode(_ lang: String) -> String {
        switch lang.lowercased() {
        case "italian", "it": return "it-IT"
        case "spanish", "es": return "es-ES"
        case "french", "fr": return "fr-FR"
        case "german", "de": return "de-DE"
        case "english", "en": return "en-US"
        default: return "en-US"
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
    let onOpenQuiz: (() -> Void)? = nil // Placeholder for "Open Quiz" icon

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
