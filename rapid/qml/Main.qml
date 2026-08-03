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

    Item {
        id: focusScope
        anchors.fill: parent
        focus: true

        MouseArea {
            anchors.fill: parent
            onClicked: focusScope.forceActiveFocus()
        }
    }

    Sidebar {
        id: sidebar

        anchors {
            top: parent.top
            bottom: parent.bottom
            left: parent.left
        }
        onDestinationSelected: destination =>
            console.log("sidebar destination:", destination)
    }

    Header {
        id: header

        anchors {
            top: parent.top
            left: sidebar.right
            right: parent.right
        }
        compact: window.compact
        onAddClicked: console.log("add clicked")
        onMenuClicked: sidebar.open = !sidebar.open
    }
}
