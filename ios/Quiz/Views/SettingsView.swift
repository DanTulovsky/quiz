import SwiftUI
import Combine
import UserNotifications

// swiftlint:disable file_length
// swiftlint:disable:next type_body_length
struct SettingsView: View {
    @StateObject private var viewModel = SettingsViewModel()
    @EnvironmentObject var authViewModel: AuthenticationViewModel

    // Account Info State
    @State private var username = ""
    @State private var email = ""
    @State private var timezone = ""

    // All available IANA timezone identifiers
    private var commonTimezones: [String] {
        TimeZone.knownTimeZoneIdentifiers.sorted()
    }

    // Learning Preferences State
    @State private var learningLanguage: String = ""
    @State private var currentLevel: String = ""
    @State private var ttsVoice: String = ""
    @State private var focusOnWeakAreas = true
    @State private var freshQuestionRatio: Float = 0.2
    @State private var knownQuestionPenalty: Float = 0.1
    @State private var weakAreaBoost: Float = 2.0
    @State private var reviewIntervalDays: Int = 7
    @State private var dailyGoal: Int = 10

    // Expanded States
    @State private var expandedSections: Set<String> = []

    // AI Settings State
    @State private var aiEnabled = false
    @State private var selectedProvider: String = ""
    @State private var selectedModel: String = ""
    @State private var apiKey: String = ""

    // Notifications State
    @State private var wordOfDayEmailEnabled = false
    @State private var dailyReminderEnabled = true
    @State private var wordOfDayIOSNotifyEnabled = false
    @State private var dailyReminderIOSNotifyEnabled = false

    // Theme State
    @AppStorage("app_theme") private var appTheme: String = "system"
    @AppStorage("app_font_size") private var appFontSize: String = "M"

    // Success feedback
    @State private var showSuccessMessage = false

    private func formatTimezone(_ timezone: String) -> String {
        let cityName = timezone.split(separator: "/").last?.replacingOccurrences(of: "_", with: " ") ?? timezone
        return "\(cityName) (\(timezone.split(separator: "/").first ?? ""))"
    }

    var body: some View {
        scrollContent
            .navigationTitle("Settings")
            .modifier(bodyModifiers)
    }

    private var bodyModifiers: some ViewModifier {
        SettingsBodyModifiers(
            viewModel: viewModel,
            authViewModel: authViewModel,
            learningLanguage: $learningLanguage,
            ttsVoice: $ttsVoice,
            username: $username,
            email: $email,
            timezone: $timezone,
            aiEnabled: $aiEnabled,
            wordOfDayEmailEnabled: $wordOfDayEmailEnabled,
            selectedProvider: $selectedProvider,
            selectedModel: $selectedModel,
            currentLevel: $currentLevel,
            focusOnWeakAreas: $focusOnWeakAreas,
            freshQuestionRatio: $freshQuestionRatio,
            knownQuestionPenalty: $knownQuestionPenalty,
            weakAreaBoost: $weakAreaBoost,
            reviewIntervalDays: $reviewIntervalDays,
            dailyGoal: $dailyGoal,
            wordOfDayIOSNotifyEnabled: $wordOfDayIOSNotifyEnabled,
            dailyReminderIOSNotifyEnabled: $dailyReminderIOSNotifyEnabled,
            loadInitialData: loadInitialData
        )
    }

    @ViewBuilder
    private var scrollContent: some View {
        ScrollView {
            VStack(spacing: 16) {
                if viewModel.isLoading && viewModel.learningPrefs == nil {
                    ProgressView()
                        .padding(.top, 50)
                } else {
                    mainContent
                }
            }
            .padding()
        }
        .overlay(successOverlay)
    }

    @ViewBuilder
    private var successOverlay: some View {
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
                .background(AppTheme.Colors.successGreen)
                .cornerRadius(AppTheme.CornerRadius.button)
                .shadow(radius: 10)
                .padding()
                .transition(.move(edge: .bottom).combined(with: .opacity))
            }
        }
    }

    @ViewBuilder
    private var mainContent: some View {
        settingsSection(title: "Theme", icon: "paintbrush", id: "theme") {
            VStack(alignment: .leading, spacing: 20) {
                VStack(alignment: .leading, spacing: 8) {
                    Text("Choose your preferred color theme and mode").font(.caption)
                        .foregroundColor(.secondary)

                    Toggle(
                        isOn: Binding(
                            get: {
                                appTheme == "light"
                                    || (appTheme == "system"
                                            && UITraitCollection.current.userInterfaceStyle
                                            == .light)
                            },
                            set: { newValue in
                                appTheme = newValue ? "light" : "dark"
                            }
                        )
                    ) {
                        Text("Light mode")
                            .font(.subheadline)
                    }
                }

                VStack(alignment: .leading, spacing: 8) {
                    Text("Font Size").font(.subheadline).fontWeight(.medium)
                    HStack(spacing: 8) {
                        ForEach(["S", "M", "L", "XL"], id: \.self) { size in
                            Button(action: {
                                appFontSize = size
                            }, label: {
                                Text(size)
                                    .font(
                                        size == "S"
                                            ? .caption
                                            : (size == "M"
                                                ? .subheadline
                                                : (size == "L" ? .body : .title3))
                                    )
                                    .fontWeight(appFontSize == size ? .bold : .regular)
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 8)
                                    .padding(.horizontal, 4)
                                    .background(
                                        appFontSize == size
                                            ? AppTheme.Colors.primaryBlue
                                            : AppTheme.Colors.primaryBlue.opacity(0.1)
                                    )
                                    .foregroundColor(
                                        appFontSize == size
                                            ? .white : AppTheme.Colors.primaryBlue
                                    )
                                    .cornerRadius(AppTheme.CornerRadius.badge)
                            })
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
                        ForEach(commonTimezones, id: \.self) { timezone in
                            Text(formatTimezone(timezone)).tag(timezone)
                        }
                    }
                    .pickerStyle(.menu)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(8)
                    .background(AppTheme.Colors.secondaryBackground)
                    .cornerRadius(AppTheme.CornerRadius.badge)
                }
            }
        }

        settingsSection(title: "Learning Preferences", icon: "target", id: "learning") {
            VStack(alignment: .leading, spacing: 20) {
                VStack(alignment: .leading, spacing: 8) {
                    Text("Learning Language").font(.subheadline).fontWeight(.medium)
                    Picker("Language", selection: $learningLanguage) {
                        if viewModel.availableLanguages.isEmpty {
                            Text("Loading...").tag("")
                        } else {
                            ForEach(viewModel.availableLanguages) { language in
                                Text(language.name.capitalized).tag(language.name)
                            }
                        }
                    }
                    .pickerStyle(.menu)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(8)
                    .background(AppTheme.Colors.secondaryBackground)
                    .cornerRadius(AppTheme.CornerRadius.badge)
                }

                VStack(alignment: .leading, spacing: 8) {
                    Text("Current Level").font(.subheadline).fontWeight(.medium)
                    Picker("Level", selection: $currentLevel) {
                        if viewModel.availableLevels.isEmpty {
                            Text("Loading...").tag("")
                        } else {
                            ForEach(viewModel.availableLevels, id: \.self) { level in
                                let description = viewModel.levelDescriptions[level] ?? level
                                Text("\(level) - \(description)").tag(level)
                            }
                        }
                    }
                    .pickerStyle(.menu)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(8)
                    .background(AppTheme.Colors.secondaryBackground)
                    .cornerRadius(AppTheme.CornerRadius.badge)
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
                                    let displayName =
                                        voice.displayName ?? voice.shortName ?? voice
                                        .name ?? identifier
                                    Text(displayName).tag(identifier)
                                }
                            }
                        }
                        .pickerStyle(.menu)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(4)
                        .background(Color(.secondarySystemBackground))
                        .cornerRadius(8)

                        TTSButton(
                            text: ttsTestText, language: learningLanguage,
                            voiceIdentifier: ttsVoice
                        )
                        .padding(8)
                        .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                        .cornerRadius(AppTheme.CornerRadius.badge)
                    }
                }

                Toggle(isOn: $focusOnWeakAreas) {
                    Label("Focus on weak areas", systemImage: "lightbulb")
                        .font(.subheadline)
                }

                sliderSetting(
                    label: "Fresh question ratio", value: $freshQuestionRatio,
                    range: 0...1, step: 0.1, unit: "%", multiplier: 100)

                sliderSetting(
                    label: "Known question penalty", value: $knownQuestionPenalty,
                    range: 0...1, step: 0.1, unit: "x")

                sliderSetting(
                    label: "Weak area boost", value: $weakAreaBoost, range: 1...5,
                    step: 0.5, unit: "x")

                VStack(alignment: .leading, spacing: 8) {
                    Text("Review interval (days): \(reviewIntervalDays)").font(
                        .subheadline)
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
                Text("Stay on track with your learning goals.").font(.caption)
                    .foregroundColor(.secondary)

                Button(action: { viewModel.sendTestEmail() }, label: {
                    Text("Test Email")
                        .font(.subheadline)
                        .fontWeight(.medium)
                        .foregroundColor(AppTheme.Colors.primaryBlue)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, AppTheme.Spacing.buttonVerticalPadding)
                        .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                        .cornerRadius(AppTheme.CornerRadius.button)
                })
                .disabled(email.isEmpty)

                Divider().padding(.vertical, 5)

                Toggle(isOn: $dailyReminderIOSNotifyEnabled) {
                    Label("Daily Reminder iOS Notifications", systemImage: "bell.badge.fill")
                }
                .onChange(of: dailyReminderIOSNotifyEnabled) { _, newValue in
                    updateIOSNotificationPreference(field: "daily_reminder_ios_notify_enabled", value: newValue)
                }
                Text("Receive push notifications on your iOS device for daily reminders.").font(.caption)
                    .foregroundColor(.secondary)

                Button(action: { viewModel.sendTestIOSNotification(notificationType: "daily_reminder") }, label: {
                    Text("Test Notification")
                        .font(.subheadline)
                        .fontWeight(.medium)
                        .foregroundColor(AppTheme.Colors.primaryBlue)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, AppTheme.Spacing.buttonVerticalPadding)
                        .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                        .cornerRadius(AppTheme.CornerRadius.button)
                })

                if let testResult = viewModel.testNotificationResults["daily_reminder"] {
                    Text(testResult)
                        .font(.caption)
                        .foregroundColor(
                            testResult.contains("Success")
                                ? AppTheme.Colors.successGreen
                                : AppTheme.Colors.errorRed
                        )
                        .padding(.top, 4)
                }

                Divider().padding(.vertical, 5)

                Toggle(isOn: $wordOfDayIOSNotifyEnabled) {
                    Label("Word of the Day iOS Notifications", systemImage: "bell.badge.fill")
                }
                .onChange(of: wordOfDayIOSNotifyEnabled) { _, newValue in
                    updateIOSNotificationPreference(field: "word_of_day_ios_notify_enabled", value: newValue)
                }
                Text("Receive push notifications on your iOS device with your Word of the Day.").font(.caption)
                    .foregroundColor(.secondary)

                Button(action: { viewModel.sendTestIOSNotification(notificationType: "word_of_day") }, label: {
                    Text("Test Notification")
                        .font(.subheadline)
                        .fontWeight(.medium)
                        .foregroundColor(AppTheme.Colors.primaryBlue)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, AppTheme.Spacing.buttonVerticalPadding)
                        .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                        .cornerRadius(AppTheme.CornerRadius.button)
                })

                if let testResult = viewModel.testNotificationResults["word_of_day"] {
                    Text(testResult)
                        .font(.caption)
                        .foregroundColor(
                            testResult.contains("Success")
                                ? AppTheme.Colors.successGreen
                                : AppTheme.Colors.errorRed
                        )
                        .padding(.top, 4)
                }
            }
        }

        settingsSection(title: "Word of the Day Emails", icon: "envelope", id: "wotd") {
            VStack(alignment: .leading, spacing: 15) {
                Toggle(isOn: $wordOfDayEmailEnabled) {
                    Label("Daily Email Delivery", systemImage: "envelope.fill")
                }
                Button(action: { viewModel.sendTestEmail() }, label: {
                    Text("Test Email")
                        .font(.subheadline)
                        .fontWeight(.medium)
                        .foregroundColor(AppTheme.Colors.primaryBlue)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, AppTheme.Spacing.buttonVerticalPadding)
                        .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                        .cornerRadius(AppTheme.CornerRadius.button)
                })
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
                                if let models = viewModel.aiProviders.first(where: {
                                    $0.code == selectedProvider
                                })?.models {
                                    ForEach(models, id: \.code) { model in
                                        Text(model.name).tag(model.code)
                                    }
                                }
                            }
                            .pickerStyle(.menu)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(8)
                            .background(AppTheme.Colors.secondaryBackground)
                            .cornerRadius(AppTheme.CornerRadius.badge)

                            Text("Endpoint URL").font(.subheadline).fontWeight(.medium)
                            Text(
                                viewModel.aiProviders.first(where: {
                                    $0.code == selectedProvider
                                })?.url ?? "N/A"
                            )
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
                                    .foregroundColor(AppTheme.Colors.successGreen)
                                    .font(.caption)
                            }
                        }
                        .font(.subheadline).fontWeight(.medium)
                        FormSecureField(
                            placeholder: "Enter API Key (Optional if saved)",
                            text: $apiKey,
                            showPasswordToggle: false
                        )

                        Button(action: {
                            viewModel.testAI(
                                provider: selectedProvider, model: selectedModel,
                                apiKey: apiKey)
                        }, label: {
                            Label("Test AI Connection", systemImage: "bolt.fill")
                                .font(.subheadline)
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 10)
                                .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                                .cornerRadius(AppTheme.CornerRadius.badge)
                        })
                        .disabled(selectedProvider.isEmpty || selectedModel.isEmpty)

                        if let testResult = viewModel.testResult {
                            Text(testResult)
                                .font(.caption)
                                .foregroundColor(
                                    testResult.contains("Success") ? .green : .red)
                        }
                    }
                }
            }
        }

        settingsSection(
            title: "Data Management", icon: "tray.full", id: "data", color: .red
        ) {
            VStack(alignment: .leading, spacing: 15) {
                Text("Destructive actions cannot be undone.").font(.caption)
                    .foregroundColor(.secondary)

                dataButton(
                    title: "Clear All Stories", icon: "book.closed",
                    action: { viewModel.clearStories() })
                dataButton(
                    title: "Clear AI History", icon: "bubble.left.and.bubble.right",
                    action: { viewModel.clearAIChats() })
                dataButton(
                    title: "Clear Translation History", icon: "arrow.left.and.right",
                    action: { viewModel.clearTranslationHistory() })

                Divider().padding(.vertical, 5)

                Button(action: { viewModel.resetAccount() }, label: {
                    Label("Reset All Progress", systemImage: "exclamationmark.triangle")
                        .foregroundColor(.white)
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(Color.red)
                        .cornerRadius(10)
                })
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
            .background(AppTheme.Colors.primaryBlue)
            .foregroundColor(.white)
            .cornerRadius(AppTheme.CornerRadius.button)
        }
        .padding(.top, 10)
        .disabled(viewModel.isLoading)

        if let error = viewModel.error {
            Text("Error: \(error.localizedDescription)")
                .foregroundColor(.red)
                .font(.caption)
                .padding()
                .background(AppTheme.Colors.errorRed.opacity(0.1))
                .cornerRadius(AppTheme.CornerRadius.badge)
        }

        // Logout Button
        Button(action: {
            authViewModel.logout()
        }, label: {
            HStack {
                Image(systemName: "arrow.right.square")
                Text("Logout")
            }
            .font(.headline)
            .frame(maxWidth: .infinity)
            .padding()
            .background(AppTheme.Colors.errorRed)
            .foregroundColor(.white)
            .cornerRadius(AppTheme.CornerRadius.button)
        })
        .padding(.top, 20)
    }

    private func loadInitialData() {
        if let user = authViewModel.user {
            username = user.username
            email = user.email
            timezone = user.timezone ?? ""
            // Don't set language/level/provider/model here - validation handlers will set valid values when data loads
            // This prevents warnings from Pickers rendering with invalid selections before data is available
            aiEnabled = user.aiEnabled ?? false
            wordOfDayEmailEnabled = user.wordOfDayEmailEnabled ?? false
            // Validation handlers will set these from user preferences when data loads
        }
        if let prefs = viewModel.learningPrefs {
            ttsVoice = prefs.ttsVoice ?? ""
            wordOfDayIOSNotifyEnabled = prefs.wordOfDayIosNotifyEnabled ?? false
            dailyReminderIOSNotifyEnabled = prefs.dailyReminderIosNotifyEnabled ?? false
        }
    }

    private var ttsTestText: String {
        switch learningLanguage.lowercased() {
        case "italian", "it": return "Questa è una prova della voce selezionata."
        case "spanish", "es": return "Esta es una prueba de la voz seleccionada."
        case "french", "fr": return "Ceci est un test de la voix sélectionnée."
        case "german", "de": return "Dies ist ein Test der ausgewählten Stimme."
        case "russian", "ru": return "Это тест выбранного голоса."
        default: return "This is a test of the selected voice."
        }
    }

    private func updateIOSNotificationPreference(field: String, value: Bool) {
        guard var currentPrefs = viewModel.learningPrefs else { return }

        if field == "daily_reminder_ios_notify_enabled" {
            currentPrefs.dailyReminderIosNotifyEnabled = value
        } else if field == "word_of_day_ios_notify_enabled" {
            currentPrefs.wordOfDayIosNotifyEnabled = value
        }

        // Request notification permissions when user enables a notification toggle
        if value {
            UNUserNotificationCenter.current().getNotificationSettings { settings in
                DispatchQueue.main.async {
                    if settings.authorizationStatus != .authorized {
                        NotificationService.shared.requestAuthorization()
                    }
                }
            }
        }

        viewModel.apiService.updateLearningPreferences(prefs: currentPrefs)
            .receive(on: DispatchQueue.main)
            .sink(
                receiveCompletion: { (completion: Subscribers.Completion<APIService.APIError>) in
                    switch completion {
                    case .failure(let error):
                        print("❌ Failed to update iOS notification preference: \(error.localizedDescription)")
                    case .finished:
                        break
                    }
                },
                receiveValue: { [weak viewModel] updatedPrefs in
                    viewModel?.learningPrefs = updatedPrefs
                }
            )
            .store(in: &viewModel.cancellables)
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
            wordOfDayIosNotifyEnabled: wordOfDayIOSNotifyEnabled,
            dailyReminderIosNotifyEnabled: dailyReminderIOSNotifyEnabled,
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
    private func settingsSection<Content: View>(
        title: String, icon: String, id: String, color: Color = .primary,
        @ViewBuilder content: () -> Content
    ) -> some View {
        let isExpanded = expandedSections.contains(id)

        VStack(spacing: 0) {
            Button(action: {
                if isExpanded {
                    expandedSections.remove(id)
                } else {
                    expandedSections.insert(id)
                }
            }, label: {
                HStack {
                    Image(systemName: icon)
                        .foregroundColor(color == .primary ? AppTheme.Colors.primaryBlue : color)
                        .frame(width: 24)
                    Text(title)
                        .foregroundColor(color)
                        .fontWeight(.medium)
                    Spacer()
                    Image(systemName: isExpanded ? "chevron.up" : "chevron.down")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 12)
                .background(Color(.systemBackground))
            })
            if isExpanded {
                Divider().padding(.horizontal)
                VStack(alignment: .leading) {
                    content()
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 12)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(Color(.systemBackground))
                .transition(.opacity.combined(with: .move(edge: .bottom)))
            }
        }
        .animation(.easeInOut(duration: 0.2), value: isExpanded)
        .appCard()
    }

    @ViewBuilder
    private func settingsField(label: String, text: Binding<String>, required: Bool = false)
    -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(label)
                    .font(.subheadline)
                    .fontWeight(.medium)
                if required {
                    Text("*").foregroundColor(.red)
                }
            }
            FormTextField(placeholder: label, text: text)
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

    private struct SettingsBodyModifiers: ViewModifier {
        let viewModel: SettingsViewModel
        let authViewModel: AuthenticationViewModel
        @Binding var learningLanguage: String
        @Binding var ttsVoice: String
        @Binding var username: String
        @Binding var email: String
        @Binding var timezone: String
        @Binding var aiEnabled: Bool
        @Binding var wordOfDayEmailEnabled: Bool
        @Binding var selectedProvider: String
        @Binding var selectedModel: String
        @Binding var currentLevel: String
        @Binding var focusOnWeakAreas: Bool
        @Binding var freshQuestionRatio: Float
        @Binding var knownQuestionPenalty: Float
        @Binding var weakAreaBoost: Float
        @Binding var reviewIntervalDays: Int
        @Binding var dailyGoal: Int
        @Binding var wordOfDayIOSNotifyEnabled: Bool
        @Binding var dailyReminderIOSNotifyEnabled: Bool
        let loadInitialData: () -> Void

        func body(content: Content) -> some View {
            let viewWithAppearance = applyAppearanceModifiers(to: content)
            return applyChangeModifiers(to: viewWithAppearance)
        }

        private func applyAppearanceModifiers<V: SwiftUI.View>(to view: V) -> some View {
            view
                .onAppear {
                    loadInitialData()
                    viewModel.fetchSettings()
                    viewModel.fetchAIProviders()
                    viewModel.fetchLanguages()
                    viewModel.fetchLevels(language: learningLanguage)
                }
                .onChange(of: viewModel.availableLanguages) { _, languages in
                    TTSSynthesizerManager.shared.updateDefaultVoiceCache(languages: languages)
                    viewModel.fetchVoices(language: learningLanguage)
                    viewModel.fetchLevels(language: learningLanguage)
                }
                .onChange(of: ttsVoice) { _, newValue in
                    TTSSynthesizerManager.shared.preferredVoice = newValue
                }
                .onChange(of: viewModel.availableVoices) { _, voices in
                    // When voices load, if ttsVoice is empty or not in the list, set to default
                    if !voices.isEmpty {
                        let currentVoiceId = ttsVoice.isEmpty ? nil : ttsVoice
                        let voiceIds = voices.compactMap { $0.shortName ?? $0.name }
                        if currentVoiceId == nil || !voiceIds.contains(currentVoiceId!) {
                            // Set to default voice from LanguageInfo, or first in list
                            if let defaultVoice = viewModel.getDefaultVoiceIdentifier(
                                for: learningLanguage) {
                                ttsVoice = defaultVoice
                                TTSSynthesizerManager.shared.preferredVoice = defaultVoice
                            } else if let firstVoice = voices.first {
                                let identifier = firstVoice.shortName ?? firstVoice.name ?? ""
                                if !identifier.isEmpty {
                                    ttsVoice = identifier
                                    TTSSynthesizerManager.shared.preferredVoice = identifier
                                }
                            }
                        }
                    }
                }
        }

        private func applyChangeModifiers<V: SwiftUI.View>(to view: V) -> some View {
            let viewWithLanguageModifiers = applyLanguageValidationModifiers(to: view)
            let viewWithAIModifiers = applyAIValidationModifiers(to: viewWithLanguageModifiers)
            return applyUserValidationModifiers(to: viewWithAIModifiers)
        }

        private func applyLanguageValidationModifiers<V: SwiftUI.View>(to view: V) -> some View {
            view
                .onChange(of: viewModel.availableLanguages) { _, languages in
                    validateAndSetLanguage(languages: languages)
                }
                .onChange(of: viewModel.availableLevels) { _, levels in
                    validateAndSetLevel(levels: levels)
                }
                .onChange(of: learningLanguage) { _, newValue in
                    if !viewModel.availableLanguages.isEmpty {
                        viewModel.fetchVoices(language: newValue)
                        viewModel.fetchLevels(language: newValue)
                    }
                }
        }

        private func validateAndSetLanguage(languages: [LanguageInfo]) {
            guard !languages.isEmpty else { return }
            if learningLanguage.isEmpty {
                setLanguageFromPreferences(languages: languages)
            } else {
                validateExistingLanguage(languages: languages)
            }
        }

        private func setLanguageFromPreferences(languages: [LanguageInfo]) {
            let preferredLang = authViewModel.user?.preferredLanguage ?? languages.first?.name ?? ""
            guard !preferredLang.isEmpty else {
                learningLanguage = languages.first?.name ?? ""
                return
            }
            let lowercasedSelection = preferredLang.lowercased()
            if let nameMatch = languages.first(where: { $0.name.lowercased() == lowercasedSelection }) {
                learningLanguage = nameMatch.name
            } else if let codeMatch = languages.first(where: { $0.code.lowercased() == lowercasedSelection }) {
                learningLanguage = codeMatch.name
            } else if languages.contains(where: { $0.name == preferredLang }) {
                learningLanguage = preferredLang
            } else {
                learningLanguage = languages.first?.name ?? ""
            }
        }

        private func validateExistingLanguage(languages: [LanguageInfo]) {
            let lowercasedSelection = learningLanguage.lowercased()
            let exactMatch = languages.first { $0.name == learningLanguage }
            if exactMatch != nil {
                return
            }
            if let nameMatch = languages.first(where: { $0.name.lowercased() == lowercasedSelection }) {
                learningLanguage = nameMatch.name
            } else if let codeMatch = languages.first(where: { $0.code.lowercased() == lowercasedSelection }) {
                learningLanguage = codeMatch.name
            } else {
                learningLanguage = languages.first?.name ?? ""
            }
        }

        private func validateAndSetLevel(levels: [String]) {
            guard !levels.isEmpty else { return }
            if currentLevel.isEmpty {
                let preferredLevel = authViewModel.user?.currentLevel ?? levels.first ?? ""
                if !preferredLevel.isEmpty && levels.contains(preferredLevel) {
                    currentLevel = preferredLevel
                } else {
                    currentLevel = levels.first ?? ""
                }
            } else if !levels.contains(currentLevel) {
                currentLevel = levels.first ?? ""
            }
        }

        private func applyAIValidationModifiers<V: SwiftUI.View>(to view: V) -> some View {
            view
                .onChange(of: viewModel.aiProviders) { _, providers in
                    validateAIProviders(providers: providers)
                }
                .onChange(of: selectedProvider) { _, newProvider in
                    validateSelectedProvider(newProvider: newProvider)
                }
        }

        private func validateAIProviders(providers: [AIProviderInfo]) {
            guard !providers.isEmpty && !selectedProvider.isEmpty else { return }
            if let provider = providers.first(where: { $0.code == selectedProvider }) {
                if !selectedModel.isEmpty {
                    let modelExists = provider.models.contains { $0.code == selectedModel }
                    if !modelExists {
                        selectedModel = ""
                    }
                }
            } else {
                selectedProvider = ""
                selectedModel = ""
            }
        }

        private func validateSelectedProvider(newProvider: String) {
            guard !newProvider.isEmpty else {
                selectedModel = ""
                return
            }
            if let provider = viewModel.aiProviders.first(where: { $0.code == newProvider }) {
                if !selectedModel.isEmpty {
                    let modelExists = provider.models.contains { $0.code == selectedModel }
                    if !modelExists {
                        selectedModel = ""
                    }
                }
            } else {
                selectedModel = ""
            }
        }

        private func applyUserValidationModifiers<V: SwiftUI.View>(to view: V) -> some View {
            view
                .onChange(of: authViewModel.user) { _, user in
                    if let user = user {
                        updateFromUser(user: user)
                    }
                }
                .onChange(of: viewModel.learningPrefs) { _, prefs in
                    if let prefs = prefs {
                        updateFromLearningPreferences(prefs: prefs)
                    }
                }
        }

        private func updateFromUser(user: User) {
            username = user.username
            email = user.email
            timezone = user.timezone ?? ""
            aiEnabled = user.aiEnabled ?? false
            wordOfDayEmailEnabled = user.wordOfDayEmailEnabled ?? false
            if let provider = user.aiProvider, !provider.isEmpty {
                if viewModel.aiProviders.contains(where: { $0.code == provider }) {
                    selectedProvider = provider
                    if let model = user.aiModel, !model.isEmpty,
                       let providerInfo = viewModel.aiProviders.first(where: { $0.code == provider }),
                       providerInfo.models.contains(where: { $0.code == model }) {
                        selectedModel = model
                    } else {
                        selectedModel = ""
                    }
                } else {
                    selectedProvider = ""
                    selectedModel = ""
                }
            } else {
                selectedProvider = ""
                selectedModel = ""
            }
        }

        private func updateFromLearningPreferences(prefs: UserLearningPreferences) {
            let preferredLang = authViewModel.user?.preferredLanguage ?? ""
            if !preferredLang.isEmpty && !viewModel.availableLanguages.isEmpty {
                setLanguageFromPreferences(languages: viewModel.availableLanguages)
            } else if !viewModel.availableLanguages.isEmpty && learningLanguage.isEmpty {
                learningLanguage = viewModel.availableLanguages.first?.name ?? ""
            }
            let preferredLevel = authViewModel.user?.currentLevel ?? ""
            if !preferredLevel.isEmpty && !viewModel.availableLevels.isEmpty {
                if viewModel.availableLevels.contains(preferredLevel) {
                    currentLevel = preferredLevel
                } else {
                    currentLevel = viewModel.availableLevels.first ?? ""
                }
            } else if !viewModel.availableLevels.isEmpty && currentLevel.isEmpty {
                currentLevel = viewModel.availableLevels.first ?? ""
            }
            ttsVoice = prefs.ttsVoice ?? ""
            TTSSynthesizerManager.shared.preferredVoice = ttsVoice
            focusOnWeakAreas = prefs.focusOnWeakAreas
            freshQuestionRatio = prefs.freshQuestionRatio
            knownQuestionPenalty = prefs.knownQuestionPenalty
            weakAreaBoost = prefs.weakAreaBoost
            reviewIntervalDays = prefs.reviewIntervalDays
            dailyGoal = prefs.dailyGoal ?? 10
            wordOfDayIOSNotifyEnabled = prefs.wordOfDayIosNotifyEnabled ?? false
            dailyReminderIOSNotifyEnabled = prefs.dailyReminderIosNotifyEnabled ?? false
        }
    }

    @ViewBuilder
    private func sliderSetting(
        label: String, value: Binding<Float>, range: ClosedRange<Float>, step: Float,
        unit: String = "", multiplier: Float = 1.0
    ) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(label)
                    .font(.subheadline)
                    .fontWeight(.medium)
                Spacer()
                Text("\(Int(value.wrappedValue * multiplier))\(unit)")
                    .font(.caption)
                    .foregroundColor(AppTheme.Colors.primaryBlue)
                    .fontWeight(.bold)
            }
            Slider(value: value, in: range, step: step)
        }
    }
}
