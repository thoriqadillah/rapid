pragma Singleton

import QtQuick

QtObject {
    id: root

    // Set this once from the Router component on completion
    property var router: null

    function navigate(route) {
        if (!router) return
        router.navigate(route)
    }

    readonly property string downloadPage: "download"
    readonly property string settingsPage: "settings"
}
