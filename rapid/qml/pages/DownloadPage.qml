import QtQuick
import ".."

Item {
    property string routeName: "all"
    property string filters: "all"

    Text {
        anchors.centerIn: parent
        text: qsTr("All Downloads: ") + parent.filters
        color: Theme.colorText
        font.pixelSize: 18
    }
}
