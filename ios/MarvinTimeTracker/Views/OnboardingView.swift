import SwiftUI

struct OnboardingView: View {
    @Bindable var viewModel: TrackingViewModel

    @State private var serverURL = ""
    @State private var apiToken = ""
    @State private var isValidating = false
    @State private var validationError: String?

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Server URL", text: $serverURL)
                        .textContentType(.URL)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)
                        .keyboardType(.URL)

                    SecureField("Marvin API Token", text: $apiToken)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)
                } header: {
                    Text("Configuration")
                } footer: {
                    Text("Enter your Go relay server URL and Marvin API token.")
                }

                if let error = validationError {
                    Section {
                        Label(error, systemImage: "exclamationmark.triangle")
                            .foregroundStyle(.red)
                    }
                }

                Section {
                    Button {
                        Task { await validate() }
                    } label: {
                        HStack {
                            Text("Connect")
                            if isValidating {
                                Spacer()
                                ProgressView()
                            }
                        }
                    }
                    .disabled(serverURL.isEmpty || apiToken.isEmpty || isValidating)
                }
            }
            .navigationTitle("Setup")
        }
    }

    private func validate() async {
        isValidating = true
        validationError = nil
        defer { isValidating = false }

        let normalizedURL = serverURL.hasSuffix("/")
            ? String(serverURL.dropLast())
            : serverURL

        // Save server URL first so validateToken can use it
        viewModel.saveCredentials(token: apiToken, serverURL: normalizedURL)

        let isValid = await viewModel.validateToken(apiToken)
        if !isValid {
            validationError = "Invalid token or server unreachable"
            viewModel.signOut()
        }
    }
}
