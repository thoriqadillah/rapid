pragma Singleton

import QtQuick

QtObject {
    id: root

    // Set this once from the Router component on completion
    property var router: null
    readonly property string currentRoute: router?.currentItem?.objectName ?? ""

    function replace(route) {
        if (!router) return
        const comp = router.routes[route]
        if (!comp) return
        if (root.currentRoute === route) return

        router.replace(null, comp, { objectName: route })
    }

    function push(route) {
        if (!router) return
        const comp = router.routes[route]
        if (!comp) return

        router.push(comp, { objectName: route })
    }

    function back() {
        if (!router || router.depth <= 1) return
        router.pop()
    }

    readonly property string downloadPage: "download"
    readonly property string settingsPage: "settings"
}
