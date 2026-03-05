import Foundation

struct MarvinAPIClient {
    private let token: String
    private let serverURL: String
    private let marvinBaseURL = "https://serv.amazingmarvin.com/api"

    init(token: String, serverURL: String) {
        self.token = token
        self.serverURL = serverURL
    }

    // MARK: - Direct Marvin API calls (read-only)

    func validateToken() async throws -> Bool {
        var request = URLRequest(url: URL(string: "\(marvinBaseURL)/me")!)
        request.setValue(token, forHTTPHeaderField: "X-API-Token")

        let (_, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else { return false }
        return httpResponse.statusCode == 200
    }

    func todayItems() async throws -> [MarvinTask] {
        var request = URLRequest(url: URL(string: "\(marvinBaseURL)/todayItems")!)
        request.setValue(token, forHTTPHeaderField: "X-API-Token")

        let (data, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            return []
        }

        return (try? JSONDecoder().decode([MarvinTask].self, from: data)) ?? []
    }

    // MARK: - Go server calls (mutations)

    func startTracking(taskId: String, title: String) async throws {
        let url = URL(string: "\(serverURL)/start")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(["taskId": taskId, "title": title])

        let (_, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw APIError.serverError
        }
    }

    func stopTracking(taskId: String? = nil) async throws {
        let url = URL(string: "\(serverURL)/stop")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        if let taskId {
            request.httpBody = try JSONEncoder().encode(["taskId": taskId])
        } else {
            request.httpBody = "{}".data(using: .utf8)
        }

        let (_, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw APIError.serverError
        }
    }

    func fetchStatus() async throws -> ServerStatus {
        let url = URL(string: "\(serverURL)/status")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode(ServerStatus.self, from: data)
    }

    enum APIError: Error {
        case serverError
    }
}

struct ServerStatus: Codable {
    let status: String
    let tracking: Bool
    let taskId: String?
    let taskTitle: String?
    let startedAt: Int64?
    let hasPushToStartToken: Bool
    let hasUpdateToken: Bool
}
