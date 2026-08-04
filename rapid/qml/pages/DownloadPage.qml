import QtQuick
import "../components"
import ".."

Layout {
    id: root
    property string type: ''
    property string searchText: ''

    onDestinationSelected: destination => {
        if (destination === Navigation.settingsPage) Navigation.navigate(destination)
        else type = destination
    }

    onSearchTextChanged: root.searchText = searchText

    Item {
        anchors.fill: parent

        Text {
            anchors.centerIn: parent
            text: qsTr("All Downloads: ") + root.type + " - search: " + root.searchText
            color: Theme.colorText
            font.pixelSize: 18
        }
    }
}
