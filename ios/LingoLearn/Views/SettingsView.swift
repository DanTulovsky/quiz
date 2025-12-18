import SwiftUI

struct SettingsView: View {
    @StateObject private var viewModel = SettingsViewModel()
    @EnvironmentObject var authViewModel: AuthenticationViewModel

    // Account Info State
    @State private var username = ""
    @State private var email = ""
    @State private var timezone = ""

    // Common timezones list
    private let commonTimezones = [
        "UTC",
        "America/New_York",
        "America/Chicago",
        "America/Denver",
        "America/Los_Angeles",
        "America/Mexico_City",
        "America/Sao_Paulo",
        "Europe/London",
        "Europe/Paris",
        "Europe/Berlin",
        "Europe/Madrid",
        "Europe/Moscow",
        "Asia/Dubai",
        "Asia/Kolkata",
        "Asia/Bangkok",
        "Asia/Singapore",
        "Asia/Shanghai",
        "Asia/Tokyo",
        "Australia/Sydney",
        "Pacific/Auckland"
    ]

    // Learning Preferences State
    @State private var learningLanguage: String = "italian"
    @State private var currentLevel: String = "A1"
    @State private var ttsVoice: String = ""
    @State private var focusOnWeakAreas = true
    @State private var freshQuestionRatio: Float = 0.2
    @State private var knownQuestionPenalty: Float = 0.1
    @State private var weakAreaBoost: Float = 2.0
    @State private var reviewIntervalDays: Int = 7
    @State private var dailyGoal: Int = 10

    // Expanded States
    @State private var expandedSections: Set<String> = ["account"]

    // AI Settings State
    @State private var aiEnabled = false
    @State private var selectedProvider: String = ""
    @State private var selectedModel: String = ""
    @State private var apiKey: String = ""

    // Notifications State
    @State private var wordOfDayEmailEnabled = false
    @State private var dailyReminderEnabled = true

    // Theme State
    @AppStorage("app_theme") private var appTheme: String = "system"
    @AppStorage("app_font_size") private var appFontSize: String = "M"

    // Success feedback
    @State private var showSuccessMessage = false

    private func formatTimezone(_ tz: String) -> String {
        let cityName = tz.split(separator: "/").last?.replacingOccurrences(of: "_", with: " ") ?? tz
        return "\(cityName) (\(tz.split(separator: "/").first ?? ""))"
    }

    var body: some View {
        ScrollView {
            VStack(spacing: 16) {
                if viewModel.isLoading && viewModel.learningPrefs == nil {
                    ProgressView()
                        .padding(.top, 50)
                } else {
                    settingsSection(title: "Theme", icon: "paintbrush", id: "theme") {
                        VStack(alignment: .leading, spacing: 20) {
                            VStack(alignment: .leading, spacing: 8) {
                                Text("Choose your preferred color theme and mode").font(.caption).foregroundColor(.secondary)

                                Toggle(isOn: Binding(
                                    get: { appTheme == "light" || (appTheme == "system" && UITraitCollection.current.userInterfaceStyle == .light) },
                                    set: { newValue in
                                        appTheme = newValue ? "light" : "dark"
                                    }
                                )) {
                                    Text("Light mode")
                                        .font(.subheadline)
                                }
                            }

                            VStack(alignment: .leading, spacing: 8) {
                                Text("Font Size").font(.subheadline).fontWeight(.medium)
                                HStack(spacing: 12) {
                                    ForEach(["S", "M", "L", "XL"], id: \.self) { size in
                                        Button(action: {
                                            appFontSize = size
                                        }) {
                                            Text(size)
                                                .font(.subheadline)
                                                .fontWeight(appFontSize == size ? .bold : .regular)
                                                .frame(maxWidth: .infinity)
                                                .padding(.vertical, 12)
                                                .background(appFontSize == size ? Color.blue : Color.blue.opacity(0.1))
                                                .foregroundColor(appFontSize == size ? .white : .blue)
                                                .cornerRadius(8)
                                        }
                                    }
                                }
                            }
                        }
                    }

                    settingsSection(title: "Account Information", icon: "person", id: "account") {
                        VStack(alignment: .leading, spacing: 15) {
                            settingsField(label: "Username", text: $username, required: true)
                            settingsField(label: "Email", text: $email, required: true)

                            VStack(alignment: .leading, spacing: 8) {
                                Text("Timezone").font(.subheadline).fontWeight(.medium)
                                Picker("Timezone", selection: $timezone) {
                                    Text("Select Timezone").tag("")
                                    ForEach(commonTimezones, id: \.self) { tz in
                                        Text(formatTimezone(tz)).tag(tz)
                                    }
                                }
                                .pickerStyle(.menu)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(8)
                                .background(Color(.secondarySystemBackground))
                                .cornerRadius(8)
                            }
                        }
                    }

                    settingsSection(title: "Learning Preferences", icon: "target", id: "learning") {
                        VStack(alignment: .leading, spacing: 20) {
                            VStack(alignment: .leading, spacing: 8) {
                                Text("Learning Language").font(.subheadline).fontWeight(.medium)
                                Picker("Language", selection: $learningLanguage) {
                                    Text("Italian").tag("italian")
                                    Text("Spanish").tag("spanish")
                                    Text("French").tag("french")
                                    Text("German").tag("german")
                                    Text("English").tag("english")
                                }
                                .pickerStyle(.menu)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(8)
                                .background(Color(.secondarySystemBackground))
                                .cornerRadius(8)
                            }

                            VStack(alignment: .leading, spacing: 8) {
                                Text("Current Level").font(.subheadline).fontWeight(.medium)
                                Picker("Level", selection: $currentLevel) {
                                    ForEach(["A1", "A2", "B1", "B2", "C1", "C2"], id: \.self) { level in
                                        Text(level).tag(level)
                                    }
                                }
                                .pickerStyle(.menu)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(8)
                                .background(Color(.secondarySystemBackground))
                                .cornerRadius(8)
                            }

                            VStack(alignment: .leading, spacing: 8) {
                                Text("TTS Voice").font(.subheadline).fontWeight(.medium)
                                HStack {
                                    Picker("Select Voice", selection: $ttsVoice) {
                                        if viewModel.availableVoices.isEmpty {
                                            if !ttsVoice.isEmpty {
                                                Text(ttsVoice).tag(ttsVoice)
                                            } else {
                                                Text("Loading voices...").tag("")
                                            }
                                        } else {
                                            Text("Default").tag("")
                                            ForEach(viewModel.availableVoices) { voice in
                                                let identifier = voice.shortName ?? voice.name ?? ""
                                                let displayName = voice.displayName ?? voice.shortName ?? voice.name ?? identifier
                                                Text(displayName).tag(identifier)
                                            }
                                        }
                                    }
                                    .pickerStyle(.menu)
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                    .padding(4)
                                    .background(Color(.secondarySystemBackground))
                                    .cornerRadius(8)

                                    TTSButton(text: ttsTestText, language: learningLanguage, voiceIdentifier: ttsVoice)
                                        .padding(8)
                                        .background(Color.blue.opacity(0.1))
                                        .cornerRadius(8)
                                }
                            }

                            Toggle(isOn: $focusOnWeakAreas) {
                                Label("Focus on weak areas", systemImage: "lightbulb")
                                    .font(.subheadline)
                            }

                            sliderSetting(label: "Fresh question ratio", value: $freshQuestionRatio, range: 0...1, step: 0.1, unit: "%", multiplier: 100)

                            sliderSetting(label: "Known question penalty", value: $knownQuestionPenalty, range: 0...1, step: 0.1, unit: "x")

                            sliderSetting(label: "Weak area boost", value: $weakAreaBoost, range: 1...5, step: 0.5, unit: "x")

                            VStack(alignment: .leading, spacing: 8) {
                                Text("Review interval (days): \(reviewIntervalDays)").font(.subheadline)
                                Stepper("Days", value: $reviewIntervalDays, in: 1...60)
                            }

                            VStack(alignment: .leading, spacing: 8) {
                                Text("Daily goal: \(dailyGoal) questions").font(.subheadline)
                                Stepper("Questions", value: $dailyGoal, in: 1...100)
                            }
                        }
                    }

                    settingsSection(title: "Notifications", icon: "bell", id: "notifications") {
                        VStack(alignment: .leading, spacing: 15) {
                            Toggle(isOn: $dailyReminderEnabled) {
                                Text("Daily Email Reminders")
                                    .font(.subheadline)
                            }
                            Text("Stay on track with your learning goals.").font(.caption).foregroundColor(.secondary)

                            Button(action: { viewModel.sendTestEmail() }) {
                                Text("Test Email")
                                    .font(.subheadline)
                                    .fontWeight(.medium)
                                    .foregroundColor(.blue)
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 12)
                                    .background(Color.blue.opacity(0.1))
                                    .cornerRadius(10)
                            }
                            .disabled(email.isEmpty)
                        }
                    }

                    settingsSection(title: "Word of the Day Emails", icon: "envelope", id: "wotd") {
                        VStack(alignment: .leading, spacing: 15) {
                            Toggle(isOn: $wordOfDayEmailEnabled) {
                                Label("Daily Email Delivery", systemImage: "envelope.fill")
                            }
                            Button(action: { viewModel.sendTestEmail() }) {
                                Text("Test Email")
                                    .font(.subheadline)
                                    .fontWeight(.medium)
                                    .foregroundColor(.blue)
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 12)
                                    .background(Color.blue.opacity(0.1))
                                    .cornerRadius(10)
                            }
                            .disabled(email.isEmpty)
                        }
                    }

                    settingsSection(title: "AI Settings", icon: "cpu", id: "ai") {
                        VStack(alignment: .leading, spacing: 20) {
                            Toggle(isOn: $aiEnabled) {
                                Label("Enable AI Features", systemImage: "sparkles")
                            }

                            if aiEnabled {
                                VStack(alignment: .leading, spacing: 12) {
                                    Text("AI Provider").font(.subheadline).fontWeight(.medium)
                                    Picker("Provider", selection: $selectedProvider) {
                                        Text("Select Provider").tag("")
                                        ForEach(viewModel.aiProviders) { provider in
                                            Text(provider.name).tag(provider.code)
                                        }
                                    }
                                    .pickerStyle(.menu)
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                    .padding(8)
                                    .background(Color(.secondarySystemBackground))
                                    .cornerRadius(8)

                                    if !selectedProvider.isEmpty {
                                        Text("AI Model").font(.subheadline).fontWeight(.medium)
                                        Picker("Model", selection: $selectedModel) {
                                            Text("Select Model").tag("")
                                            if let models = viewModel.aiProviders.first(where: { $0.code == selectedProvider })?.models {
                                                ForEach(models, id: \.code) { model in
                                                    Text(model.name).tag(model.code)
                                                }
                                            }
                                        }
                                        .pickerStyle(.menu)
                                        .frame(maxWidth: .infinity, alignment: .leading)
                                        .padding(8)
                                        .background(Color(.secondarySystemBackground))
                                        .cornerRadius(8)

                                        Text("Endpoint URL").font(.subheadline).fontWeight(.medium)
                                        Text(viewModel.aiProviders.first(where: { $0.code == selectedProvider })?.url ?? "N/A")
                                            .padding(10)
                                            .frame(maxWidth: .infinity, alignment: .leading)
                                            .background(Color(.secondarySystemBackground))
                                            .cornerRadius(8)
                                            .foregroundColor(.secondary)
                                    }

                                    HStack {
                                        Text("API Key")
                                        if authViewModel.user?.hasApiKey == true {
                                            Text("(Saved)")
                                                .foregroundColor(.green)
                                                .font(.caption)
                                        }
                                    }
                                    .font(.subheadline).fontWeight(.medium)
                                    SecureField("Enter API Key (Optional if saved)", text: $apiKey)
                                        .padding(10)
                                        .background(Color(.secondarySystemBackground))
                                        .cornerRadius(8)

                                    Button(action: { viewModel.testAI(provider: selectedProvider, model: selectedModel, apiKey: apiKey) }) {
                                        Label("Test AI Connection", systemImage: "bolt.fill")
                                            .font(.subheadline)
                                            .frame(maxWidth: .infinity)
                                            .padding(.vertical, 10)
                                            .background(Color.blue.opacity(0.1))
                                            .cornerRadius(8)
                                    }
                                    .disabled(selectedProvider.isEmpty || selectedModel.isEmpty)

                                    if let testResult = viewModel.testResult {
                                        Text(testResult)
                                            .font(.caption)
                                            .foregroundColor(testResult.contains("Success") ? .green : .red)
                                    }
                                }
                            }
                        }
                    }

                    settingsSection(title: "Data Management", icon: "tray.full", id: "data", color: .red) {
                        VStack(alignment: .leading, spacing: 15) {
                            Text("Destructive actions cannot be undone.").font(.caption).foregroundColor(.secondary)

                            dataButton(title: "Clear All Stories", icon: "book.closed", action: { viewModel.clearStories() })
                            dataButton(title: "Clear AI History", icon: "bubble.left.and.bubble.right", action: { viewModel.clearAIChats() })
                            dataButton(title: "Clear Translation History", icon: "arrow.left.and.right", action: { viewModel.clearTranslationHistory() })

                            Divider().padding(.vertical, 5)

                            Button(action: { viewModel.resetAccount() }) {
                                Label("Reset All Progress", systemImage: "exclamationmark.triangle")
                                    .foregroundColor(.white)
                                    .frame(maxWidth: .infinity)
                                    .padding()
                                    .background(Color.red)
                                    .cornerRadius(10)
                            }
                        }
                    }

                    Button(action: saveChanges) {
                        HStack {
                            Image(systemName: "checkmark.circle")
                            Text("Save Changes")
                        }
                        .font(.headline)
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(Color.blue)
                        .foregroundColor(.white)
                        .cornerRadius(12)
                    }
                    .padding(.top, 10)
                    .disabled(viewModel.isLoading)

                    if let error = viewModel.error {
                        Text("Error: \(error.localizedDescription)")
                            .foregroundColor(.red)
                            .font(.caption)
                            .padding()
                            .background(Color.red.opacity(0.1))
                            .cornerRadius(8)
                    }

                    // Logout Button
                    Button(action: {
                        authViewModel.logout()
                    }) {
                        HStack {
                            Image(systemName: "arrow.right.square")
                            Text("Logout")
                        }
                        .font(.headline)
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(Color.red)
                        .foregroundColor(.white)
                        .cornerRadius(12)
                    }
                    .padding(.top, 20)
                }
            }
            .padding()
        }
        .overlay(
            Group {
                if showSuccessMessage {
                    VStack {
                        Spacer()
                        HStack {
                            Image(systemName: "checkmark.circle.fill")
                                .foregroundColor(.white)
                            Text("Changes saved successfully!")
                                .foregroundColor(.white)
                                .fontWeight(.medium)
                        }
                        .padding()
                        .background(Color.green)
                        .cornerRadius(12)
                        .shadow(radius: 10)
                        .padding()
                        .transition(.move(edge: .bottom).combined(with: .opacity))
                    }
                }
            }
        )
        .navigationTitle("Settings")
        .onAppear {
            loadInitialData()
            viewModel.fetchSettings(); viewModel.fetchAIProviders(); viewModel.fetchVoices(language: learningLanguage)
        }
                        .onChange(of: ttsVoice) { _, newValue in
            TTSSynthesizerManager.shared.preferredVoice = newValue
        }
        .onChange(of: learningLanguage) { _, newValue in
            viewModel.fetchVoices(language: newValue)
        }
        .onChange(of: authViewModel.user) { _, user in
            if let user = user {
                username = user.username
                email = user.email
                timezone = user.timezone ?? ""
                aiEnabled = user.aiEnabled ?? false
                wordOfDayEmailEnabled = user.wordOfDayEmailEnabled ?? false
                selectedProvider = user.aiProvider ?? ""
                selectedModel = user.aiModel ?? ""
            }
        }
        .onChange(of: viewModel.learningPrefs) { _, prefs in
            if let prefs = prefs {
                learningLanguage = authViewModel.user?.preferredLanguage ?? "italian"
                currentLevel = authViewModel.user?.currentLevel ?? "A1"
                ttsVoice = prefs.ttsVoice ?? ""
                TTSSynthesizerManager.shared.preferredVoice = ttsVoice
                focusOnWeakAreas = prefs.focusOnWeakAreas
                freshQuestionRatio = prefs.freshQuestionRatio
                knownQuestionPenalty = prefs.knownQuestionPenalty
                weakAreaBoost = prefs.weakAreaBoost
                reviewIntervalDays = prefs.reviewIntervalDays
                dailyGoal = prefs.dailyGoal ?? 10
            }
        }
    }

    private func loadInitialData() {
        if let user = authViewModel.user {
            username = user.username
            email = user.email
            timezone = user.timezone ?? ""
            learningLanguage = user.preferredLanguage ?? "italian"
            currentLevel = user.currentLevel ?? "A1"
            aiEnabled = user.aiEnabled ?? false
            wordOfDayEmailEnabled = user.wordOfDayEmailEnabled ?? false
            selectedProvider = user.aiProvider ?? ""
            selectedModel = user.aiModel ?? ""
        }
        if let prefs = viewModel.learningPrefs {
            ttsVoice = prefs.ttsVoice ?? ""
        }
    }

        private var ttsTestText: String {
        switch learningLanguage.lowercased() {
        case "italian", "it": return "Questa è una prova della voce selezionata."
        case "spanish", "es": return "Esta es una prueba de la voz seleccionada."
        case "french", "fr": return "Ceci est un test de la voix sélectionnée."
        case "german", "de": return "Dies ist ein Test der ausgewählten Stimme."
        default: return "This is a test of the selected voice."
        }
    }

    private func saveChanges() {
        var userUpdate = UserUpdateRequest()
        userUpdate.username = username
        userUpdate.email = email
        userUpdate.timezone = timezone
        userUpdate.preferredLanguage = learningLanguage
        userUpdate.currentLevel = currentLevel
        userUpdate.aiEnabled = aiEnabled
        userUpdate.wordOfDayEmailEnabled = wordOfDayEmailEnabled
        userUpdate.aiProvider = selectedProvider.isEmpty ? nil : selectedProvider
        userUpdate.aiModel = selectedModel.isEmpty ? nil : selectedModel
        userUpdate.apiKey = apiKey.isEmpty ? nil : apiKey

        let prefs = UserLearningPreferences(
            focusOnWeakAreas: focusOnWeakAreas,
            freshQuestionRatio: freshQuestionRatio,
            knownQuestionPenalty: knownQuestionPenalty,
            reviewIntervalDays: reviewIntervalDays,
            weakAreaBoost: weakAreaBoost,
            dailyReminderEnabled: dailyReminderEnabled,
            ttsVoice: ttsVoice,
            dailyGoal: dailyGoal
        )

        viewModel.saveChanges(userUpdate: userUpdate, prefs: prefs)

        // Show success message after a short delay to ensure save completes
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
            if viewModel.error == nil {
                withAnimation {
                    showSuccessMessage = true
                }
                // Hide after 3 seconds
                DispatchQueue.main.asyncAfter(deadline: .now() + 3) {
                    withAnimation {
                        showSuccessMessage = false
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func settingsSection<Content: View>(title: String, icon: String, id: String, color: Color = .primary, @ViewBuilder content: () -> Content) -> some View {
        let isExpanded = expandedSections.contains(id)

        VStack(spacing: 0) {
            Button(action: {
                if isExpanded {
                    expandedSections.remove(id)
                } else {
                    expandedSections.insert(id)
                }
            }) {
                HStack {
                    Image(systemName: icon)
                        .foregroundColor(color == .primary ? .blue : color)
                        .frame(width: 24)
                    Text(title)
                        .foregroundColor(color)
                        .fontWeight(.medium)
                    Spacer()
                    Image(systemName: isExpanded ? "chevron.up" : "chevron.down")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
                .padding()
                .background(Color(.systemBackground))
            }

            if isExpanded {
                Divider().padding(.horizontal)
                VStack(alignment: .leading) {
                    content()
                }
                .padding()
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(Color(.systemBackground))
            }
        }
        .cornerRadius(12)
        .shadow(color: Color.black.opacity(0.05), radius: 5, x: 0, y: 2)
        .overlay(RoundedRectangle(cornerRadius: 12).stroke(Color.gray.opacity(0.1), lineWidth: 1))
    }

    @ViewBuilder
    private func settingsField(label: String, text: Binding<String>, required: Bool = false) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(label)
                    .font(.subheadline)
                    .fontWeight(.medium)
                if required {
                    Text("*").foregroundColor(.red)
                }
            }
            TextField(label, text: text)
                .padding(10)
                .background(Color(.secondarySystemBackground))
                .cornerRadius(8)
        }
    }

    @ViewBuilder
    private func dataButton(title: String, icon: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack {
                Label(title, systemImage: icon)
                Spacer()
                Image(systemName: "chevron.right").font(.caption)
            }
            .foregroundColor(.red)
            .padding(.vertical, 8)
        }
    }

    @ViewBuilder
    private func sliderSetting(label: String, value: Binding<Float>, range: ClosedRange<Float>, step: Float, unit: String = "", multiplier: Float = 1.0) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(label)
                    .font(.subheadline)
                    .fontWeight(.medium)
                Spacer()
                Text("\(Int(value.wrappedValue * multiplier))\(unit)")
                    .font(.caption)
                    .foregroundColor(.blue)
                    .fontWeight(.bold)
            }
            Slider(value: value, in: range, step: step)
        }
    }
}
