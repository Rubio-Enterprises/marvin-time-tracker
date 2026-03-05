import Foundation

struct MarvinTask: Identifiable, Codable, Hashable {
    let id: String
    let title: String

    enum CodingKeys: String, CodingKey {
        case id = "_id"
        case title
    }
}
