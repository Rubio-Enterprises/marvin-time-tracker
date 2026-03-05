import ActivityKit
import Foundation

struct TimeTrackerAttributes: ActivityAttributes {
    struct ContentState: Codable, Hashable {
        var taskTitle: String
        var startedAt: Date
        var isTracking: Bool
    }
}
