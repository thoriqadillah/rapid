import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts as QL
import "../components"
import "../components/download" as Download
import ".."

// DownloadService and DownloadDialog are Python-registered context properties,
// invisible to qmllint's static analysis.
// qmllint disable unqualified
Layout {
    id: root
    property string type: ''

    readonly property int progressColumnWidth: 300
    readonly property int speedColumnWidth: 120
    readonly property int sizeColumnWidth: 120
    readonly property int etaColumnWidth: 120

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

    QL.ColumnLayout {
        anchors.fill: parent
        anchors.leftMargin: Theme.spacingPageLeft
        anchors.rightMargin: Theme.spacingPageRight
        anchors.topMargin: Theme.spacingPageTop

        spacing: 0

        Rectangle {
            QL.Layout.fillWidth: true
            implicitHeight: headerRow.implicitHeight + Theme.spacingSm * 2
            color: Theme.colorBase

            QL.RowLayout {
                id: headerRow
                anchors {
                    fill: parent
                    leftMargin: Theme.spacingLg
                    rightMargin: Theme.spacingLg
                }
                spacing: Theme.spacingMd

                Text {
                    text: qsTr("Name")
                    color: Theme.colorTextMuted
                    font.pixelSize: Theme.textSize
                    QL.Layout.fillWidth: true
                }

                Text {
                    text: qsTr("Progress")
                    color: Theme.colorTextMuted
                    font.pixelSize: Theme.textSize
                    QL.Layout.preferredWidth: root.progressColumnWidth
                }

                Text {
                    text: qsTr("Speed")
                    color: Theme.colorTextMuted
                    font.pixelSize: Theme.textSize
                    QL.Layout.preferredWidth: root.speedColumnWidth
                }

                Text {
                    text: qsTr("Size")
                    color: Theme.colorTextMuted
                    font.pixelSize: Theme.textSize
                    QL.Layout.preferredWidth: root.sizeColumnWidth
                }

                Text {
                    text: qsTr("Estimation")
                    color: Theme.colorTextMuted
                    font.pixelSize: Theme.textSize
                    QL.Layout.preferredWidth: root.etaColumnWidth
                }
            }
        }

        Item {
            QL.Layout.fillWidth: true
            QL.Layout.fillHeight: true
            QL.Layout.topMargin: Theme.spacingSm

            ListView {
                id: list
                anchors.fill: parent
                clip: true
                boundsBehavior: Flickable.StopAtBounds
                model: DownloadService
                Controls.ScrollBar.vertical: Controls.ScrollBar {
                    policy: Controls.ScrollBar.AsNeeded
                }
                delegate: Download.DownloadItem {
                    width: ListView.view.width
                    progressColumnWidth: root.progressColumnWidth
                    speedColumnWidth: root.speedColumnWidth
                    sizeColumnWidth: root.sizeColumnWidth
                    etaColumnWidth: root.etaColumnWidth
                }

                add: Transition {
                    NumberAnimation { property: "opacity"; from: 0; to: 1; duration: 300 }
                    NumberAnimation { property: "y"; from: -Theme.spacingXs; duration: 300; easing.type: Easing.OutCubic }
                }
            }

            Text {
                visible: list.count === 0
                anchors.centerIn: parent
                text: qsTr("No downloads yet")
                color: Theme.colorTextMuted
                font.pixelSize: Theme.textSize
            }
        }
    }
}
