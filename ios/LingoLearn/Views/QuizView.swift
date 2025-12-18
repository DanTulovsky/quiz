import SwiftUI

struct QuizView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel: QuizViewModel
    @State private var reportReason = ""
    @State private var selectedConfidence: Int? = nil

    @StateObject private var ttsManager = TTSSynthesizerManager.shared

    init(question: Question? = nil, questionType: String? = nil, isDaily: Bool = false) {
        _viewModel = StateObject(wrappedValue: QuizViewModel(question: question, questionType: questionType, isDaily: isDaily))
    }

    private func stringValue(_ v: JSONValue?) -> String? {
        guard let v else { return nil }
        if case .string(let s) = v { return s }
        return nil
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
        ScrollView {
            ScrollViewReader { proxy in
                VStack(spacing: 20) {
                    if let error = ttsManager.errorMessage {
                        Text(error)
                            .font(.caption)
                            .foregroundColor(.red)
                            .padding()
                            .background(Color.red.opacity(0.1))
                            .cornerRadius(8)
                    }

                    if viewModel.isLoading && viewModel.question == nil {
                        ProgressView()
                            .padding(.top, 50)
                    }

                    if let error = viewModel.error {
                        Text("Error: \(error.localizedDescription)")
                            .foregroundColor(.red)
                            .padding()
                            .background(Color.red.opacity(0.1))
                            .cornerRadius(8)
                    }

                    if let message = viewModel.generatingMessage {
                        VStack {
                            Text(message)
                                .padding()
                            Button("Try Again") {
                                viewModel.getQuestion()
                            }
                            .buttonStyle(.bordered)
                        }
                    } else if let question = viewModel.question {
                        questionCard(question)

                        optionsList(question)

                        if let response = viewModel.answerResponse {
                            feedbackSection(response)
                        }

                        actionButtons()

                        footerButtons()
                    } else {
                        VStack(spacing: 20) {
                            Image(systemName: "questionmark.circle")
                                .font(.system(size: 60))
                                .foregroundColor(.blue)
                            Text("Ready to test your knowledge?")
                                .font(.title2)
                            Button("Start Quiz") {
                                viewModel.getQuestion()
                            }
                            .buttonStyle(.borderedProminent)
                            .controlSize(.large)
                        }
                        .padding(.top, 100)
                    }
                }
                .padding()
                Color.clear.frame(height: 1).id("bottom")

                    .onChange(of: viewModel.selectedAnswerIndex) { old, val in
                        if val != nil {
                            withAnimation {
                                proxy.scrollTo("bottom", anchor: .bottom)
                            }
                        }
                    }
                    .onChange(of: viewModel.answerResponse) { old, response in
                        if response != nil {
                            withAnimation {
                                proxy.scrollTo("bottom", anchor: .bottom)
                            }
                        }
                    }
            }
        }
        .navigationBarTitleDisplayMode(.inline)
        .navigationBarBackButtonHidden(true)
        .toolbar {
            ToolbarItem(placement: .navigationBarLeading) {
                Button(action: { dismiss() }) {
                    HStack(spacing: 4) {
                        Image(systemName: "chevron.left")
                            .font(.system(size: 17, weight: .semibold))
                        Text("Back")
                            .font(.system(size: 17))
                    }
                    .foregroundColor(.blue)
                }
            }
        }
        .sheet(isPresented: $viewModel.showReportModal) {
            reportSheet
        }
        .sheet(isPresented: $viewModel.showMarkKnownModal) {
            markKnownSheet
        }
        .onAppear {
            if viewModel.question == nil { viewModel.getQuestion() }
        }
    }

    @ViewBuilder
    private func questionCard(_ question: Question) -> some View {
        VStack(alignment: .leading, spacing: 15) {
            HStack {
                BadgeView(text: question.type.replacingOccurrences(of: "_", with: " ").uppercased(), color: .indigo)
                Spacer()
                BadgeView(text: "\(question.language.uppercased()) - \(question.level)", color: .blue)
            }

            if let passage = stringValue(question.content["passage"]) {
                VStack(alignment: .trailing) {
                    TTSButton(text: passage, language: question.language)
                    Text(passage)
                        .font(.body)
                        .lineSpacing(4)
                }
                .padding()
                .background(Color(.systemBackground))
                .cornerRadius(12)
                .overlay(RoundedRectangle(cornerRadius: 12).stroke(Color.gray.opacity(0.2), lineWidth: 1))
            }

            if let sentence = stringValue(question.content["sentence"]) {
                let targetWord = stringValue(question.content["question"])
                highlightedText(sentence, targetWord: targetWord)
                    .font(.title3)
                    .fontWeight(.medium)
            } else if let questionText = stringValue(question.content["question"]) ?? stringValue(question.content["prompt"]) {
                Text(questionText)
                    .font(.title3)
                    .fontWeight(.medium)
            }

            if question.type == "vocabulary", let targetWord = stringValue(question.content["question"]) {
                Text("What does **\(targetWord)** mean in this context?")
            }
        }
        .padding()
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(.systemBackground))
        .cornerRadius(16)
        .shadow(color: Color.black.opacity(0.05), radius: 8, x: 0, y: 4)
        .overlay(RoundedRectangle(cornerRadius: 16).stroke(Color.gray.opacity(0.1), lineWidth: 1))
    }

    @ViewBuilder
    private func highlightedText(_ fullText: String, targetWord: String?) -> some View {
        if let targetWord = targetWord, let range = fullText.range(of: targetWord, options: .caseInsensitive) {
            let before = String(fullText[..<range.lowerBound])
            let word = String(fullText[range])
            let after = String(fullText[range.upperBound...])

            Text("\(Text(before))\(Text(word).foregroundColor(.blue).fontWeight(.bold))\(Text(after))")
        } else {
            Text(fullText)
        }
    }

    @ViewBuilder
    private func optionsList(_ question: Question) -> some View {
        if let options = stringArrayValue(question.content["options"]) {
            VStack(spacing: 12) {
                ForEach(Array(options.enumerated()), id: \.offset) { idx, option in
                    optionButton(option: option, index: idx)
                }
            }
        }
    }

    @ViewBuilder
    private func optionButton(option: String, index: Int) -> some View {
        let isSelected = viewModel.selectedAnswerIndex == index
        let isCorrect = viewModel.answerResponse?.correctAnswerIndex == index
        let isUserIncorrect = viewModel.answerResponse != nil && viewModel.answerResponse?.userAnswerIndex == index && !isCorrect

        Button(action: {
            if viewModel.answerResponse == nil {
                viewModel.selectedAnswerIndex = index
            }
        }) {
            HStack {
                if viewModel.answerResponse != nil {
                    if isCorrect {
                        Image(systemName: "check.circle.fill")
                            .foregroundColor(.green)
                    } else if isUserIncorrect {
                        Image(systemName: "x.circle.fill")
                            .foregroundColor(.red)
                    }
                }

                Text(option)
                    .font(.body)
                    .foregroundColor(isUserIncorrect ? .red : (isCorrect ? .green : (isSelected ? .white : .primary)))

                Spacer()
            }
            .padding()
            .frame(maxWidth: .infinity)
            .background(
                isUserIncorrect ? Color.red.opacity(0.1) :
                (isCorrect ? Color.green.opacity(0.1) :
                (isSelected ? Color.blue : Color.blue.opacity(0.05)))
            )
            .cornerRadius(12)
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(isUserIncorrect ? Color.red : (isCorrect ? Color.green : Color.blue.opacity(0.2)), lineWidth: 1)
            )
        }
        .disabled(viewModel.answerResponse != nil)
    }

    @ViewBuilder
    private func feedbackSection(_ response: AnswerResponse) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Image(systemName: response.isCorrect ? "checkmark.circle.fill" : "xmark.circle.fill")
                    .foregroundColor(response.isCorrect ? .green : .red)
                Text(response.isCorrect ? "Correct!" : "Incorrect")
                    .font(.headline)
                    .foregroundColor(response.isCorrect ? .green : .red)
            }

            Text(response.explanation)
                .font(.subheadline)
                .foregroundColor(.secondary)
                .fixedSize(horizontal: false, vertical: true)
        }
        .padding()
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(response.isCorrect ? Color.green.opacity(0.05) : Color.red.opacity(0.05))
        .cornerRadius(12)
    }

    @ViewBuilder
    private func actionButtons() -> some View {
        if let _ = viewModel.answerResponse {
            Button("Next Question") {
                viewModel.selectedAnswerIndex = nil
                viewModel.getQuestion()
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.large)
            .frame(maxWidth: .infinity)
        } else {
            Button("Submit Answer") {
                if let idx = viewModel.selectedAnswerIndex {
                    viewModel.submitAnswer(userAnswerIndex: idx)
                }
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.large)
            .disabled(viewModel.selectedAnswerIndex == nil || viewModel.isLoading)
            .frame(maxWidth: .infinity)
        }
    }

    @ViewBuilder
    private func footerButtons() -> some View {
        HStack(spacing: 20) {
            Button(action: { viewModel.showReportModal = true }) {
                Label(viewModel.isReported ? "Reported" : "Report issue", systemImage: "flag")
                    .font(.caption)
            }
            .disabled(viewModel.isReported)
            .foregroundColor(.secondary)

            Spacer()

            Button(action: { viewModel.showMarkKnownModal = true }) {
                Label("Adjust frequency", systemImage: "slider.horizontal.3")
                    .font(.caption)
            }
            .foregroundColor(.secondary)
        }
        .padding(.top, 10)
    }

    private var reportSheet: some View {
        NavigationView {
            Form {
                Section(header: Text("Why are you reporting this question?")) {
                    TextEditor(text: $reportReason)
                        .frame(minHeight: 100)
                }

                Button("Submit Report") {
                    viewModel.reportQuestion(reason: reportReason)
                }
                .disabled(viewModel.isSubmittingAction)
            }
            .navigationTitle("Report Issue")
            .navigationBarItems(trailing: Button("Cancel") { viewModel.showReportModal = false })
        }
    }

    private var markKnownSheet: some View {
        NavigationView {
            VStack(spacing: 20) {
                Text("Choose how often you want to see this question in future quizzes: 1–2 show it more, 3 no change, 4–5 show it less.")
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
                        .background(selectedConfidence == level ? Color.teal : Color.teal.opacity(0.1))
                        .foregroundColor(selectedConfidence == level ? .white : .teal)
                        .cornerRadius(12)
                    }
                }
                .padding(.horizontal)

                Spacer()

                Button("Save Preference") {
                    if let confidence = selectedConfidence {
                        viewModel.markQuestionKnown(confidence: confidence)
                    }
                }
                .buttonStyle(.borderedProminent)
                .tint(.teal)
                .disabled(selectedConfidence == nil || viewModel.isSubmittingAction)
                .padding()
            }
            .navigationTitle("Adjust Frequency")
            .navigationBarItems(trailing: Button("Cancel") { viewModel.showMarkKnownModal = false })
        }
    }
}
