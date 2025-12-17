import SwiftUI

struct WordOfTheDayView: View {
    @StateObject private var viewModel = WordOfTheDayViewModel()
    
    private var displayFormatter: DateFormatter {
        let formatter = DateFormatter()
        formatter.dateStyle = .full
        return formatter
    }
    
    var body: some View {
        ScrollView {
            VStack(spacing: 25) {
                // Header with Today button
                HStack {
                    Text("Word of the Day")
                        .font(.title)
                        .fontWeight(.bold)
                    Spacer()
                    Button(action: { viewModel.fetchToday() }) {
                        Label("Today", systemImage: "calendar")
                            .font(.subheadline)
                            .foregroundColor(.blue)
                    }
                }
                .padding(.top)
                
                // Date Navigation
                HStack(spacing: 20) {
                    Button(action: { viewModel.previousDay() }) {
                        Image(systemName: "chevron.left")
                            .font(.title2)
                            .padding(10)
                            .background(Color.blue.opacity(0.1))
                            .clipShape(Circle())
                    }
                    
                    Text(displayFormatter.string(from: viewModel.currentDate))
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .frame(maxWidth: .infinity)
                    
                    Button(action: { viewModel.nextDay() }) {
                        Image(systemName: "chevron.right")
                            .font(.title2)
                            .padding(10)
                            .background(Color.blue.opacity(0.1))
                            .clipShape(Circle())
                    }
                    .disabled(Calendar.current.isDateInToday(viewModel.currentDate))
                }
                
                if viewModel.isLoading {
                    ProgressView()
                        .padding(.top, 50)
                } else if let wotd = viewModel.wordOfTheDay {
                    wordCard(wotd)
                    
                    Text("Use arrows to navigate between days")
                        .font(.caption)
                        .foregroundColor(.secondary)
                        .padding(.top)
                } else if let error = viewModel.error {
                    VStack(spacing: 15) {
                        Image(systemName: "exclamationmark.triangle")
                            .font(.largeTitle)
                            .foregroundColor(.red)
                        Text(error.localizedDescription)
                            .multilineTextAlignment(.center)
                        Button("Retry") { viewModel.fetchWordOfTheDay() }
                            .buttonStyle(.bordered)
                    }
                    .padding(.top, 50)
                }
            }
            .padding()
        }
        .navigationBarTitleDisplayMode(.inline)
        .onAppear {
            viewModel.fetchWordOfTheDay()
        }
    }
    
    @ViewBuilder
    private func wordCard(_ wotd: WordOfTheDayDisplay) -> some View {
        VStack(spacing: 25) {
            // Word and Translation
            VStack(spacing: 10) {
                Text(wotd.word)
                    .font(.system(size: 48, weight: .bold))
                    .foregroundColor(.primary)
                
                Text(wotd.translation)
                    .font(.title2)
                    .italic()
                    .foregroundColor(.secondary)
            }
            .padding(.top, 10)
            
            // Example Sentence Inner Card
            VStack(alignment: .leading, spacing: 12) {
                Text(wotd.sentence)
                    .font(.title3)
                    .lineSpacing(4)
            }
            .padding()
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Color(.systemBackground))
            .cornerRadius(12)
            .shadow(color: Color.black.opacity(0.05), radius: 5, x: 0, y: 2)
            .overlay(RoundedRectangle(cornerRadius: 12).stroke(Color.blue.opacity(0.2), lineWidth: 1))
            
            // Explanation Inner Card
            if let explanation = wotd.explanation {
                VStack(alignment: .leading, spacing: 12) {
                    Text(explanation)
                        .font(.body)
                        .foregroundColor(.primary)
                }
                .padding()
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(Color(.systemBackground))
                .cornerRadius(12)
                .shadow(color: Color.black.opacity(0.05), radius: 5, x: 0, y: 2)
                .overlay(RoundedRectangle(cornerRadius: 12).stroke(Color.blue.opacity(0.2), lineWidth: 1))
            }
            
            // Badges
            HStack(spacing: 10) {
                BadgeView(text: wotd.language.uppercased(), color: .gray)
                if let level = wotd.level {
                    BadgeView(text: level.uppercased(), color: .gray)
                }
                BadgeView(text: "VOCABULARY", color: .gray)
            }
            .padding(.bottom, 10)
        }
        .padding(25)
        .frame(maxWidth: .infinity)
        .background(Color(.systemBackground))
        .cornerRadius(20)
        .overlay(
            RoundedRectangle(cornerRadius: 20)
                .stroke(Color.blue.opacity(0.5), lineWidth: 2)
        )
        .shadow(color: Color.black.opacity(0.1), radius: 15, x: 0, y: 10)
    }
}
