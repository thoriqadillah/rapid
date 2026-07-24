import QtQuick
import QtQuick.Controls
import "components"

ApplicationWindow {
    id: window

    width: 640
    height: 480
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

    RHeader {
        anchors {
            top: parent.top
            left: parent.left
            right: parent.right
        }
        compact: window.compact
        onAddClicked: console.log("add clicked")
    }
}
