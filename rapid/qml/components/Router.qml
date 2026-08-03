import QtQuick
import QtQuick.Controls

StackView {
    id: root

    // Map of route name (string) → Component
    property var routes: ({})
    property string currentRoute: ""
    property bool animated: false

    clip: true

    // TODO: animate on mobile
    replaceEnter: null
    replaceExit: null

    function navigate(route) {
        const comp = routes[route]
        if (!comp) return
        if (root.currentRoute === route) return

        root.currentRoute = route
        root.replace(null, comp, { routeName: route })
    }
}
