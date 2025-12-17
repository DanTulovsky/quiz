import SwiftUI

struct DailyView: View {
    @StateObject private var viewModel = DailyViewModel()
    @EnvironmentObject var authViewModel: AuthenticationViewModel

    var body: some View {
        ScrollView {
            ScrollViewReader { proxy in
                VStack(spacing: 20) {
                    if viewModel.isLoading && viewModel.dailyQuestions.isEmpty {
                        ProgressView("Loading Daily Challenge...")
                            .padding(.top, 50)
                    } else if let error = viewModel.error {
                        errorView(error)
                    } else if let question = viewModel.currentQuestion {
                        headerSection

                        questionCard(question.question)

                        optionsList(question.question)

                        if let response = viewModel.answerResponse {
                            feedbackSection(response)

                            Button(action: {
                                viewModel.nextQuestion()
                            }) {
                                Text("Next Question")
                                    .font(.headline)
                                    .frame(maxWidth: .infinity)
                                    .padding()
                                    .background(Color.blue)
                                    .foregroundColor(.white)
                                    .cornerRadius(12)
                            }
                            .padding(.top, 10)
                        }
                    } else if !viewModel.dailyQuestions.isEmpty {
                        completionView
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
        .navigationTitle("Daily Challenge")
        .navigationBarTitleDisplayMode(.inline)
        .onAppear {
            viewModel.fetchDaily()
        }
    }

    private var headerSection: some View {
        VStack(spacing: 12) {
            HStack {
                BadgeView(text: "DAILY CHALLENGE", color: .orange)
                Spacer()
                BadgeView(text: "\(viewModel.currentQuestion?.question.language.uppercased() ?? "") - \(viewModel.currentQuestion?.question.level ?? "")", color: .blue)
            }

            HStack {
                Text(Date(), style: .date)
                    .font(.subheadline)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 8)
                    .background(Color(.systemBackground))
                    .cornerRadius(8)
                    .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.gray.opacity(0.2), lineWidth: 1))

                Spacer()

                BadgeView(text: "\(viewModel.currentQuestionIndex + 1) OF \(viewModel.dailyQuestions.count)", color: .blue)
            }

            ProgressView(value: viewModel.progress)
                .accentColor(.orange)
                .scaleEffect(x: 1, y: 2, anchor: .center)
                .clipShape(RoundedRectangle(cornerRadius: 4))
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(16)
        .shadow(color: Color.black.opacity(0.05), radius: 8, x: 0, y: 4)
    }

    private func questionCard(_ question: Question) -> some View {
        VStack(alignment: .leading, spacing: 15) {
            HStack {
                HStack(spacing: 4) {
                    Circle().fill(Color.blue).frame(width: 6, height: 6)
                    Text(question.type.replacingOccurrences(of: "_", with: " ").capitalized)
                        .font(.caption)
                        .fontWeight(.bold)
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(Color.blue.opacity(0.1))
                .foregroundColor(.blue)
                .cornerRadius(12)

                Spacer()
            }

            let sentence = stringValue(question.content["sentence"])
            let questionText = stringValue(question.content["question"]) ?? stringValue(question.content["prompt"])

            if let sentence = sentence {
                let targetWord = stringValue(question.content["question"])
                highlightedText(sentence, targetWord: targetWord)
                    .font(.title2)
                    .fontWeight(.bold)
                    .lineSpacing(4)
            } else if let questionText = questionText {
                Text(questionText)
                    .font(.title2)
                    .fontWeight(.bold)
            }

            if question.type == "vocabulary", let targetWord = stringValue(question.content["question"]) {
                Text("What does **\(targetWord)** mean in this context?")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }
        }
        .padding()
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(.systemBackground))
        .cornerRadius(16)
        .shadow(color: Color.black.opacity(0.05), radius: 8, x: 0, y: 4)
    }

    private func optionsList(_ question: Question) -> some View {
        VStack(spacing: 12) {
            if let options = stringArrayValue(question.content["options"]) {
                ForEach(Array(options.enumerated()), id: \.offset) { idx, option in
                    optionButton(option: option, index: idx)
                }
            }
        }
    }

    private func optionButton(option: String, index: Int) -> some View {
        let isSelected = viewModel.selectedAnswerIndex == index
        let isCorrect = viewModel.answerResponse?.correctAnswerIndex == index
        let showResults = viewModel.answerResponse != nil

        return Button(action: {
            if !showResults {
                viewModel.submitAnswer(index: index)
            }
        }) {
            HStack {
                if showResults && isCorrect {
                    Image(systemName: "checkmark")
                        .foregroundColor(.green)
                }

                Text(option)
                    .font(.body)
                    .foregroundColor(showResults ? (isCorrect ? .green : .gray.opacity(0.5)) : (isSelected ? .white : .primary))

                Spacer()
            }
            .padding()
            .frame(maxWidth: .infinity)
            .background(isSelected && !showResults ? Color.blue : Color.gray.opacity(0.05))
            .cornerRadius(12)
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(isSelected && !showResults ? Color.blue : Color.clear, lineWidth: 1)
            )
        }
        .disabled(showResults)
    }

    private func feedbackSection(_ response: DailyAnswerResponse) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: response.isCorrect ? "checkmark" : "xmark")
                    .foregroundColor(response.isCorrect ? .green : .red)
                Text(response.isCorrect ? "Correct!" : "Incorrect")
                    .font(.headline)
                    .foregroundColor(response.isCorrect ? .green : .red)
            }

            Text(response.explanation)
                .font(.subheadline)
                .foregroundColor(.primary)
                .fixedSize(horizontal: false, vertical: true)
        }
        .padding()
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(response.isCorrect ? Color.green.opacity(0.05) : Color.red.opacity(0.05))
        .cornerRadius(12)
        .overlay(RoundedRectangle(cornerRadius: 12).stroke(response.isCorrect ? Color.green.opacity(0.2) : Color.red.opacity(0.2), lineWidth: 1))
    }

    private var completionView: some View {
        VStack(spacing: 20) {
            Image(systemName: "trophy.fill")
                .font(.system(size: 80))
                .foregroundColor(.orange)

            Text("Daily Challenge Complete!")
                .font(.title)
                .fontWeight(.bold)

            Text("You've finished all your questions for today. Great job!")
                .multilineTextAlignment(.center)
                .foregroundColor(.secondary)

            Button("Back to Home") {
                // This would ideally pop back or switch tabs
            }
            .buttonStyle(.borderedProminent)
        }
        .padding(.top, 50)
    }

    private func errorView(_ error: Error) -> some View {
        VStack(spacing: 15) {
            Image(systemName: "exclamationmark.triangle")
                .font(.largeTitle)
                .foregroundColor(.red)
            Text("Error: \(error.localizedDescription)")
                .multilineTextAlignment(.center)
            Button("Retry") {
                viewModel.fetchDaily()
            }
            .buttonStyle(.bordered)
        }
        .padding()
    }

    // Helpers
    private func stringValue(_ v: JSONValue?) -> String? {
        guard let v else { return nil }
        if case .string(let s) = v { return s }
        return nil
    }

    private func stringArrayValue(_ v: JSONValue?) -> [String]? {
        guard let v else { return nil }
        guard case .array(let arr) = v else { return nil }
        return arr.compactMap { item -> String? in
            if case .string(let s) = item { return s }
            return nil
        }
    }

    private func highlightedText(_ fullText: String, targetWord: String?) -> some View {
        if let targetWord = targetWord, let range = fullText.range(of: targetWord, options: .caseInsensitive) {
            let before = String(fullText[..<range.lowerBound])
            let word = String(fullText[range])
            let after = String(fullText[range.upperBound...])

            return Text("\(Text(before))\(Text(word).foregroundColor(.blue).fontWeight(.bold))\(Text(after))")
        } else {
            return Text(fullText)
        }
    }
}
