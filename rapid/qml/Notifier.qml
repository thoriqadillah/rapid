pragma Singleton
import QtQuick

QtObject {
    id: root

    signal notificationRequested(string type, string title, string message)

    function error(title, message) {
        root.notificationRequested("error", title, message)
    }

    function success(title, message) {
        root.notificationRequested("success", title, message)
    }

    function info(title, message) {
        root.notificationRequested("info", title, message)
    }
}
