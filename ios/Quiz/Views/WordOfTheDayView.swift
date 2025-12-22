import SwiftUI

struct WordOfTheDayView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel = WordOfTheDayViewModel()
    @State private var showDatePicker = false
    @State private var selectedDate = Date()

    private var dateButtonLabel: String {
        if Calendar.current.isDateInToday(viewModel.currentDate) {
            return "Today"
        } else {
            return DateFormatters.displayMedium.string(from: viewModel.currentDate)
        }
    }

    var body: some View {
        ScrollView {
            VStack(spacing: 25) {
                // Header with Date button
                HStack {
                    Text("Word of the Day")
                        .font(AppTheme.Typography.headingFont)
                    Spacer()
                    Button(action: {
                        selectedDate = viewModel.currentDate
                        showDatePicker = true
                    }, label: {
                        Label(dateButtonLabel, systemImage: "calendar")
                            .font(AppTheme.Typography.subheadlineFont)
                            .foregroundColor(AppTheme.Colors.primaryBlue)
                    })
                }
                .padding(.top)

                // Date Navigation
                HStack(spacing: 20) {
                    Button(action: { viewModel.previousDay() }, label: {
                        Image(systemName: "chevron.left")
                            .font(.title2)
                            .padding(10)
                            .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                            .clipShape(Circle())
                    })

                    Text(DateFormatters.displayFull.string(from: viewModel.currentDate))
                        .font(AppTheme.Typography.subheadlineFont)
                        .foregroundColor(AppTheme.Colors.secondaryText)
                        .frame(maxWidth: .infinity)

                    Button(action: { viewModel.nextDay() }, label: {
                        Image(systemName: "chevron.right")
                            .font(.title2)
                            .padding(10)
                            .background(AppTheme.Colors.primaryBlue.opacity(0.1))
                            .clipShape(Circle())
                    })
                    .disabled(Calendar.current.isDateInToday(viewModel.currentDate))
                }

                if viewModel.isLoading {
                    ProgressView()
                        .padding(.top, 50)
                } else if let wotd = viewModel.wordOfTheDay {
                    wordCard(wotd)

                    Text("Use arrows to navigate between days")
                        .font(AppTheme.Typography.captionFont)
                        .foregroundColor(AppTheme.Colors.secondaryText)
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
        .navigationBarBackButtonHidden(true)
        .toolbar {
            ToolbarItem(placement: .navigationBarLeading) {
                Button(action: { dismiss() }, label: {
                    HStack(spacing: 4) {
                        Image(systemName: "chevron.left")
                            .scaledFont(size: 17, weight: .semibold)
                        Text("Back")
                            .scaledFont(size: 17)
                    }
                    .foregroundColor(.blue)
                })
            }
        }
        .sheet(isPresented: $showDatePicker) {
            NavigationView {
                VStack {
                    DatePicker(
                        "Select Date",
                        selection: $selectedDate,
                        in: ...Date(),
                        displayedComponents: .date
                    )
                    .datePickerStyle(.graphical)
                    .padding()

                    Spacer()
                }
                .navigationTitle("Select Date")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .navigationBarLeading) {
                        Button("Cancel") {
                            showDatePicker = false
                        }
                    }
                    ToolbarItem(placement: .navigationBarTrailing) {
                        Button("Done") {
                            viewModel.selectDate(selectedDate)
                            showDatePicker = false
                        }
                        .fontWeight(.semibold)
                    }
                }
            }
        }
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
                    .scaledFont(size: 48, weight: .bold)
                    .lineLimit(1)
                    .minimumScaleFactor(0.5)
                    .foregroundColor(AppTheme.Colors.primaryText)

                Text(wotd.translation)
                    .font(.title2)
                    .italic()
                    .foregroundColor(AppTheme.Colors.secondaryText)
            }
            .padding(.top, 10)

            // Example Sentence Inner Card
            VStack(alignment: .leading, spacing: 12) {
                Text(wotd.sentence)
                    .font(AppTheme.Typography.headingFont)
                    .lineSpacing(4)
            }
            .appInnerCard()
            .overlay(
                RoundedRectangle(cornerRadius: AppTheme.CornerRadius.innerCard)
                    .stroke(AppTheme.Colors.borderBlue, lineWidth: 1)
            )

            // Explanation Inner Card
            if let explanation = wotd.explanation {
                VStack(alignment: .leading, spacing: 12) {
                    Text(explanation)
                        .font(AppTheme.Typography.bodyFont)
                        .foregroundColor(AppTheme.Colors.primaryText)
                }
                .appInnerCard()
                .overlay(
                    RoundedRectangle(cornerRadius: AppTheme.CornerRadius.innerCard)
                        .stroke(AppTheme.Colors.borderBlue, lineWidth: 1)
                )
            }

            // Badges
            HStack(spacing: 10) {
                BadgeView(text: wotd.language.uppercased(), color: AppTheme.Colors.primaryBlue)
                if let level = wotd.level {
                    BadgeView(text: level.uppercased(), color: AppTheme.Colors.primaryBlue)
                }
                BadgeView(text: "VOCABULARY", color: AppTheme.Colors.accentIndigo)
            }
            .padding(.bottom, 10)
        }
        .appCard()
    }
}
