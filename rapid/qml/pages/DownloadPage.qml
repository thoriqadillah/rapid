import QtQuick
import "../components"
import ".."

// DownloadDialog is a Python-registered context property, invisible to
// qmllint's static analysis.
// qmllint disable unqualified
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
