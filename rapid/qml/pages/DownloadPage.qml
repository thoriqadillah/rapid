import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Effects
import QtQuick.Layouts
import "../components/download" as Download
import ".."

// DownloadService, DownloadFilter and DownloadDialog are Python-registered
// context properties, invisible to qmllint's static analysis.
// qmllint disable unqualified
Download.DownloadLayout {
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

    onTypeChanged: {
        DownloadFilter.setCategory(type)
        list.expandedGid = ""
    }

    onSearchTextChanged: {
        DownloadFilter.setSearch(searchText)
        list.expandedGid = ""
    }

    Connections {
        target: root
        function onAddClicked() {
            DownloadDialog.openFor(root.Window.window);
        }
    }

    Connections {
        target: DownloadService
        function onDownloadCompleted(gid) {
            NotificationService.success(qsTr("Download complete"), DownloadService.downloadName(gid), true);
        }

        function onDownloadFailed(gid, errorMessage) {
            NotificationService.error(qsTr("Download failed"), errorMessage || DownloadService.downloadName(gid), true);
        }
    }

    Rectangle {
        anchors.fill: parent
        anchors.leftMargin: Theme.spacingPageLeft
        anchors.rightMargin: Theme.spacingPageRight
        anchors.topMargin: Theme.spacingPageTop
        anchors.bottomMargin: Theme.spacingPageBottom
        radius: Theme.radiusSm
        color: "transparent"
        border.width: 2
        border.color: Theme.colorSurface
        clip: true
        layer.enabled: true
        layer.effect: MultiEffect {
            maskEnabled: true
            maskSource: tableMask
        }

        Rectangle {
            id: tableMask
            anchors.fill: parent
            radius: parent.radius
            color: "white"
            visible: false
            layer.enabled: true
        }

        ColumnLayout {
            anchors.fill: parent
            spacing: 0

            Rectangle {
                Layout.fillWidth: true
                implicitHeight: headerRow.implicitHeight + Theme.spacingSm * 2
                color: Theme.colorSurface
                radius: Theme.radiusMd
                Rectangle {
                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.bottom: parent.bottom
                    height: parent.radius
                    color: parent.color
                }

                RowLayout {
                    id: headerRow
                    anchors {
                        fill: parent
                        leftMargin: Theme.spacingLg
                        rightMargin: Theme.spacingLg
                    }
                    spacing: Theme.spacingMd

                    Item {
                        Layout.preferredWidth: Theme.iconXs
                        Layout.minimumWidth: Theme.iconXs
                        Layout.maximumWidth: Theme.iconXs
                    }

                    Text {
                        text: qsTr("Name")
                        color: Theme.colorTextMuted
                        font.pixelSize: Theme.textSize
                        Layout.fillWidth: true
                        Layout.minimumWidth: root.nameColumnMinWidth
                    }

                    Text {
                        text: qsTr("Progress")
                        color: Theme.colorTextMuted
                        font.pixelSize: Theme.textSize
                        Layout.preferredWidth: root.progressColumnWidth
                        Layout.minimumWidth: root.progressColumnWidth
                        Layout.maximumWidth: root.progressColumnWidth
                    }

                    Text {
                        text: qsTr("Speed")
                        color: Theme.colorTextMuted
                        font.pixelSize: Theme.textSize
                        Layout.preferredWidth: root.speedColumnWidth
                        Layout.minimumWidth: root.speedColumnWidth
                        Layout.maximumWidth: root.speedColumnWidth
                    }

                    Text {
                        text: qsTr("Size")
                        color: Theme.colorTextMuted
                        font.pixelSize: Theme.textSize
                        Layout.preferredWidth: root.sizeColumnWidth
                        Layout.minimumWidth: root.sizeColumnWidth
                        Layout.maximumWidth: root.sizeColumnWidth
                    }

                    Text {
                        text: qsTr("Estimation")
                        color: Theme.colorTextMuted
                        font.pixelSize: Theme.textSize
                        Layout.preferredWidth: root.etaColumnWidth
                        Layout.minimumWidth: root.etaColumnWidth
                        Layout.maximumWidth: root.etaColumnWidth
                    }
                }
            }

            Item {
                Layout.fillWidth: true
                Layout.fillHeight: true

                ListView {
                    id: list
                    property bool contextMenuOpen: false
                    property string contextMenuGid: ""
                    property string expandedGid: ""
                    signal contextMenuRequested(var data)
                    anchors.fill: parent
                    clip: true
                    boundsBehavior: Flickable.StopAtBounds
                    model: DownloadFilter
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
                        onRemoveRequested: function (gid, deleteFromDisk) {
                            if (list.expandedGid === gid) {
                                list.expandedGid = "";
                                DownloadService.delete(gid, deleteFromDisk);
                            } else {
                                DownloadService.delete(gid, deleteFromDisk);
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
                        itemMenu.displayName = data.displayName || "";
                        itemMenu.popup();
                    }

                    add: Transition {
                        NumberAnimation {
                            property: "opacity"
                            from: 0
                            to: 1
                            duration: 150
                        }
                        NumberAnimation {
                            property: "x"
                            from: -Theme.spacingXl
                            duration: 150
                            easing.type: Easing.OutCubic
                        }
                    }
                    remove: Transition {
                        ParallelAnimation {
                            NumberAnimation {
                                property: "opacity"
                                to: 0
                                duration: 150
                            }
                            NumberAnimation {
                                property: "x"
                                to: Theme.spacingXl
                                duration: 150
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
    }

    Download.DownloadItemContextMenu {
        id: itemMenu
        onDeleteRequested: function (gid) {
            list.contextMenuOpen = false;
            list.contextMenuGid = "";
            deleteConfirm.openFor(root.Window.window);
        }
        onClosed: {
            list.contextMenuOpen = false;
            list.contextMenuGid = "";
        }
    }

    Download.DeleteConfirmationDialog {
        id: deleteConfirm
        displayName: itemMenu.displayName || ""
        onDeleteConfirmed: function (deleteFromDisk) {
            DownloadService.delete(itemMenu.gid, deleteFromDisk);
        }
    }
}
