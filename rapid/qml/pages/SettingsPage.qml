import QtQuick
import "../components"
import ".."

Layout {
    id: root

    Item {
        anchors.fill: parent

        Text {
            anchors.centerIn: parent
            text: qsTr("Settings")
            color: Theme.colorText
            font.pixelSize: Theme.textSize
        }
    }
}
