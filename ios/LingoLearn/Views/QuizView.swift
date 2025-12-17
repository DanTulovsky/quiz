import SwiftUI

struct QuizView: View {
    @StateObject private var viewModel: QuizViewModel
    
    init(questionType: String? = nil) {
        _viewModel = StateObject(wrappedValue: QuizViewModel(questionType: questionType))
    }

    var body: some View {
        VStack {
            if let question = viewModel.question {
                Text(question.text)
                    .padding()
                
                if let choices = question.choices {
                    ForEach(choices, id: \.self) { choice in
                        Button(action: {
                            viewModel.answer = choice
                            viewModel.submitAnswer()
                        }) {
                            Text(choice)
                                .padding()
                                .background(Color.gray.opacity(0.2))
                                .cornerRadius(8)
                        }
                    }
                } else {
                    TextField("Your answer", text: $viewModel.answer)
                        .textFieldStyle(RoundedBorderTextFieldStyle())
                        .padding()
                    Button("Submit") {
                        viewModel.submitAnswer()
                    }
                }
                
                if let response = viewModel.answerResponse {
                    Text(response.isCorrect ? "Correct!" : "Incorrect")
                        .foregroundColor(response.isCorrect ? .green : .red)
                    Text(response.feedback)
                        .padding()
                    Button("Next Question") {
                        viewModel.getQuestion()
                    }
                }
            } else {
                Button("Start Quiz") {
                    viewModel.getQuestion()
                }
            }
        }
        .onAppear {
            viewModel.getQuestion()
        }
    }
}
