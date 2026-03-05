import Foundation
import KeychainAccess

struct KeychainService {
    private static let keychain = Keychain(service: "com.strubio.MarvinTimeTracker")

    static var marvinAPIToken: String? {
        get { try? keychain.get("marvinAPIToken") }
        set {
            if let newValue {
                try? keychain.set(newValue, key: "marvinAPIToken")
            } else {
                try? keychain.remove("marvinAPIToken")
            }
        }
    }

    static var serverURL: String? {
        get { try? keychain.get("serverURL") }
        set {
            if let newValue {
                try? keychain.set(newValue, key: "serverURL")
            } else {
                try? keychain.remove("serverURL")
            }
        }
    }

    static var isConfigured: Bool {
        marvinAPIToken != nil && serverURL != nil
    }
}
