import QtQuick
import "../components"
import ".."

Layout {
    id: root
    property string type: ''

    onDestinationSelected: destination => {
        if (destination === Navigation.settingsPage) Navigation.push(destination)
        else type = destination
    }

    Connections {
        target: root
        function onAddClicked() {
            DownloadDialog.openFor(root.Window.window)
        }
    }

    Item {
        anchors.fill: parent

        Text {
            anchors.centerIn: parent
            text: qsTr("All Downloads: ") + root.type + " - search: " + root.searchText
            color: Theme.colorText
            font.pixelSize: Theme.textSize
        }
    }
}
