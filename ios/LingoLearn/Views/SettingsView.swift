import SwiftUI

struct SettingsView: View {
    @StateObject private var viewModel = SettingsViewModel()
    @State private var username = ""
    @State private var email = ""
    @State private var language: Language = .en
    @State private var level: Level = .a1

    var body: some View {
        Form {
            Section(header: Text("Profile")) {
                TextField("Username", text: $username)
                TextField("Email", text: $email)
            }
            
            Section(header: Text("Language")) {
                Picker("Language", selection: $language) {
                    ForEach(Language.allCases, id: \.self) { lang in
                        Text(lang.rawValue.uppercased()).tag(lang)
                    }
                }
                Picker("Level", selection: $level) {
                    ForEach(Level.allCases, id: \.self) { lvl in
                        Text(lvl.rawValue).tag(lvl)
                    }
                }
            }
            
            Button("Save") {
                viewModel.updateUser(username: username, email: email, language: language, level: level)
            }
        }
        .navigationTitle("Settings")
        .onAppear {
            // Load user data here
        }
    }
}

extension Language: CaseIterable {
    public static var allCases: [Language] {
        return [.en, .es, .fr, .de, .it]
    }
}

extension Level: CaseIterable {
    public static var allCases: [Level] {
        return [.a1, .a2, .b1, .b2, .c1, .c2]
    }
}
