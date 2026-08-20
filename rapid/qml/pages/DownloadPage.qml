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

    readonly property int nameColumnMinWidth: 100
    readonly property int progressColumnWidth: 300
    readonly property int speedColumnWidth: 120
    readonly property int sizeColumnWidth: 120
    readonly property int etaColumnWidth: 120

    onDestinationSelected: destination => {
        if (destination === Navigation.settingsPage)
            Navigation.push(destination);
        else
            type = destination;
    }

    Connections {
        target: root
        function onAddClicked() {
            DownloadDialog.openFor(root.Window.window);
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

                Item {
                    QL.Layout.preferredWidth: Theme.iconXs
                    QL.Layout.minimumWidth: Theme.iconXs
                    QL.Layout.maximumWidth: Theme.iconXs
                }

                Text {
                    text: qsTr("Name")
                    color: Theme.colorTextMuted
                    font.pixelSize: Theme.textSize
                    QL.Layout.fillWidth: true
                    QL.Layout.minimumWidth: root.nameColumnMinWidth
                }

                Text {
                    text: qsTr("Progress")
                    color: Theme.colorTextMuted
                    font.pixelSize: Theme.textSize
                    QL.Layout.preferredWidth: root.progressColumnWidth
                    QL.Layout.minimumWidth: root.progressColumnWidth
                    QL.Layout.maximumWidth: root.progressColumnWidth
                }

                Text {
                    text: qsTr("Speed")
                    color: Theme.colorTextMuted
                    font.pixelSize: Theme.textSize
                    QL.Layout.preferredWidth: root.speedColumnWidth
                    QL.Layout.minimumWidth: root.speedColumnWidth
                    QL.Layout.maximumWidth: root.speedColumnWidth
                }

                Text {
                    text: qsTr("Size")
                    color: Theme.colorTextMuted
                    font.pixelSize: Theme.textSize
                    QL.Layout.preferredWidth: root.sizeColumnWidth
                    QL.Layout.minimumWidth: root.sizeColumnWidth
                    QL.Layout.maximumWidth: root.sizeColumnWidth
                }

                Text {
                    text: qsTr("Estimation")
                    color: Theme.colorTextMuted
                    font.pixelSize: Theme.textSize
                    QL.Layout.preferredWidth: root.etaColumnWidth
                    QL.Layout.minimumWidth: root.etaColumnWidth
                    QL.Layout.maximumWidth: root.etaColumnWidth
                }
            }
        }

        Item {
            QL.Layout.fillWidth: true
            QL.Layout.fillHeight: true
            QL.Layout.topMargin: Theme.spacingSm

            ListView {
                id: list
                property bool contextMenuOpen: false
                property string contextMenuGid: ""
                property string expandedGid: ""
                signal contextMenuRequested(var data)
                anchors.fill: parent
                clip: true
                boundsBehavior: Flickable.StopAtBounds
                model: DownloadService
                Controls.ScrollBar.vertical: Controls.ScrollBar {
                    policy: Controls.ScrollBar.AsNeeded
                }
                delegate: Download.DownloadItem {
                    width: ListView.view.width
                    nameColumnMinWidth: root.nameColumnMinWidth
                    progressColumnWidth: root.progressColumnWidth
                    speedColumnWidth: root.speedColumnWidth
                    sizeColumnWidth: root.sizeColumnWidth
                    etaColumnWidth: root.etaColumnWidth

                    menuOpen: list.contextMenuOpen
                    highlighted: list.contextMenuGid === gid
                    expanded: list.expandedGid === gid
                    onToggleExpanded: list.expandedGid = list.expandedGid === gid ? "" : gid
                    onRemoveRequested: function (gid) {
                        if (list.expandedGid === gid) {
                            list.expandedGid = "";
                            DownloadService.purge(gid);
                        } else {
                            DownloadService.purge(gid);
                        }
                    }
                    onContextMenuRequested: function (data) {
                        list.contextMenuRequested(data);
                    }
                }

                onContextMenuRequested: function (data) {
                    list.contextMenuOpen = true;
                    list.contextMenuGid = data.gid;
                    itemMenu.gid = data.gid;
                    itemMenu.status = data.status;
                    itemMenu.fileDir = data.fileDir;
                    itemMenu.resolved = data.resolved;
                    itemMenu.popup();
                }

                add: Transition {
                    NumberAnimation {
                        property: "opacity"
                        from: 0
                        to: 1
                        duration: 300
                    }
                    NumberAnimation {
                        property: "y"
                        from: -Theme.spacingXs
                        duration: 300
                        easing.type: Easing.OutCubic
                    }
                }
                remove: Transition {
                    ParallelAnimation {
                        NumberAnimation {
                            property: "opacity"
                            to: 0
                            duration: 300
                        }
                        NumberAnimation {
                            property: "y"
                            to: -height
                            duration: 300
                            easing.type: Easing.InCubic
                        }
                    }
                }
            }

            Column {
                visible: list.count === 0
                anchors.centerIn: parent
                spacing: Theme.spacingMd

                Controls.Button {
                    anchors.horizontalCenter: parent.horizontalCenter
                    enabled: false
                    padding: 0
                    display: Controls.AbstractButton.IconOnly
                    icon.source: "qrc:/icons/MdiLightFormatAlignBottom.svg"
                    icon.width: Theme.iconXl
                    icon.height: Theme.iconXl
                    icon.color: Theme.colorTextMuted
                    background: null
                }

                Text {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: qsTr("No downloads yet")
                    color: Theme.colorTextMuted
                    font.pixelSize: Theme.textSize
                }
            }
        }
    }

    Download.DownloadItemContextMenu {
        id: itemMenu
        onClosed: {
            list.contextMenuOpen = false;
            list.contextMenuGid = "";
        }
    }
}
