import SwiftUI

@main
struct MarvinTimeTrackerApp: App {
    @State private var viewModel = TrackingViewModel()

    var body: some Scene {
        WindowGroup {
            Group {
                if viewModel.isOnboarded {
                    TimerView(viewModel: viewModel)
                } else {
                    OnboardingView(viewModel: viewModel)
                }
            }
            .task {
                await viewModel.observePushTokens()
            }
        }
    }
}
