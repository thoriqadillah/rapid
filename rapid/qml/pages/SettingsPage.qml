import QtQuick
import ".."

Item {
    property string routeName: "settings"

    Text {
        anchors.centerIn: parent
        text: qsTr("Settings")
        color: Theme.colorText
        font.pixelSize: 18
    }
}
