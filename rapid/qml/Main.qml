import QtQuick
import QtQuick.Controls
import "ui"
import "components"

ApplicationWindow {
    width: 640
    height: 480
    visible: true
    title: "Rapid downloader"
    color: Theme.bg

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
        width: parent.width
        onAddClicked: console.log("add clicked")
    }
}
