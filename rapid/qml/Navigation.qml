pragma Singleton

import QtQuick
import QtQuick.Controls

QtObject {
    id: root

    // Set this once from the Router component on completion
    property StackView router: null
    readonly property string currentRoute: router?.currentItem?.objectName ?? ""
    property var routes: ({})

    function create(router, routes) {
        if (router === null || routes === null) return
        this.router = router
        this.routes = routes
    }

    function replace(route) {
        if (!router) return
        const comp = root.routes[route]
        if (!comp) return
        if (root.currentRoute === route) return

        router.replace(null, comp, { objectName: route })
    }

    function push(route) {
        if (!router) return
        const comp = root.routes[route]
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
