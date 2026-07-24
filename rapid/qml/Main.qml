import QtQuick
import QtQuick.Controls
import "components"

ApplicationWindow {
    id: window

    width: 640
    height: 480
    visible: true
    title: "Rapid downloader"
    color: Theme.bgColor

    readonly property bool compact: width < Theme.breakpointMedium
    readonly property bool medium: width >= Theme.breakpointMedium
        && width < Theme.breakpointExpanded
    readonly property bool expanded: width >= Theme.breakpointExpanded

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
