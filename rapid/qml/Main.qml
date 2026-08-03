import QtQuick
import QtQuick.Controls
import "components"

ApplicationWindow {
    id: window

    width: 1280
    height: 720
    visible: true
    title: "Rapid downloader"
    color: Theme.colorBackground

    readonly property bool compact: width < Theme.breakpointMd
    readonly property bool medium: width >= Theme.breakpointMd
        && width < Theme.breakpointLg
    readonly property bool expanded: width >= Theme.breakpointLg

    Layout {
        anchors.fill: parent

        // No headerContent or sidebarContent set → uses built-in defaults.
        // To swap in a custom header, do:
        //
        //   headerContent: Component {
        //       MyCustomHeader { ... }
        //   }
    }
}
