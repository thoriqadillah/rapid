import QtQuick
import "../components"
import ".."

Layout {
    id: root
    property string searchText: ''

    onSearchTextChanged: root.searchText = searchText

    Item {
        anchors.fill: parent

        Text {
            anchors.centerIn: parent
            text: qsTr("Settings")
            color: Theme.colorText
            font.pixelSize: 18
        }
    }
}
