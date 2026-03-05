import SwiftUI

struct TaskPickerSheet: View {
    @Bindable var viewModel: TrackingViewModel
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            Group {
                if viewModel.isLoading {
                    ProgressView("Loading tasks...")
                } else if viewModel.todayTasks.isEmpty {
                    ContentUnavailableView(
                        "No Tasks Today",
                        systemImage: "checklist",
                        description: Text("Add tasks in Marvin to see them here.")
                    )
                } else {
                    List(viewModel.todayTasks) { task in
                        Button {
                            Task {
                                await viewModel.startTracking(task: task)
                                dismiss()
                            }
                        } label: {
                            Text(task.title)
                                .foregroundStyle(.primary)
                        }
                    }
                }
            }
            .navigationTitle("Today's Tasks")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
            }
            .task {
                await viewModel.loadTodayTasks()
            }
        }
    }
}
