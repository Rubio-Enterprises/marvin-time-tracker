import Foundation

enum TrackingState: Equatable {
    case idle
    case tracking(taskId: String, title: String, startedAt: Date)

    var isTracking: Bool {
        if case .tracking = self { return true }
        return false
    }

    var taskTitle: String? {
        if case .tracking(_, let title, _) = self { return title }
        return nil
    }

    var startedAt: Date? {
        if case .tracking(_, _, let date) = self { return date }
        return nil
    }

    var taskId: String? {
        if case .tracking(let id, _, _) = self { return id }
        return nil
    }
}
