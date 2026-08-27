import QtQuick
import QtQuick.Layouts
import "../.."
import "../../ui"

// DownloadService is a Python-registered context property, invisible to
// qmllint's static analysis.
// qmllint disable unqualified
Rectangle {
    id: root

    // Role values from DownloadService (a QAbstractListModel), forwarded by the parent item.
    required property var gid
    required property var status
    required property var category
    required property var resolved
    required property var files
    required property var totalLength
    required property var completedLength
    required property var errorMessage

    signal removeRequested(var gid)

    color: "transparent"
    implicitHeight: content.implicitHeight + Theme.spacingMd + Theme.spacingLg
    Layout.fillWidth: true

    readonly property double percent: {
        const total = Number(root.totalLength) || 0
        if (!total) return 0
        return Math.min(1, (Number(root.completedLength) || 0) / total)
    }

    readonly property string displayName: {
        const res = root.resolved;
        if (res && (res.title || res.filename))
            return res.title || res.filename;
        const paths = Array.isArray(root.files) ? root.files : [];
        const path = paths.length > 0 ? paths[0].path : "";
        if (path)
            return path.split("/").pop() || path;
        return root.gid;
    }

    readonly property string fileDir: {
        const paths = root.files
        const path = paths.length > 0 ? paths[0].path : ""
        if (!path) return ""
        const idx = path.lastIndexOf("/")
        return idx > 0 ? path.slice(0, idx) : ""
    }

    readonly property bool canPause: root.status === "active" || root.status === "waiting"
    readonly property bool canResume: root.status === "paused"

    readonly property string statusText: {
        switch (root.status) {
            case "removed":
                return "Stopped"
            default:
                return root.status.charAt(0).toUpperCase() + root.status.slice(1)
        }
    }

    ColumnLayout {
        id: content
        property int columnWidth: 100
        anchors.fill: parent
        anchors.leftMargin: Theme.spacingLg
        anchors.rightMargin: Theme.spacingLg
        anchors.topMargin: Theme.spacingMd
        spacing: Theme.spacingMd

        ResolverUri {
            modelData: root.resolved
            category: root.category
            readonly: true
        }

        DownloadSpeedSample {
            gid: root.gid
            category: root.category
            Layout.topMargin: Theme.spacingSm
            Layout.bottomMargin: Theme.spacingSm
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingMd

            Text {
                text: qsTr("Progress")
                color: Theme.colorTextMuted
                font.pixelSize: Theme.textSize
                Layout.minimumWidth: content.columnWidth
            }

            Rectangle {
                Layout.fillWidth: true
                implicitHeight: 10
                radius: height / 2
                border.width: 1
                border.color: Theme.colorBorder
                color: Theme.colorSurface

                Rectangle {
                    width: parent.width * root.percent
                    height: parent.height
                    radius: height / 2
                    color: Theme.categoryColor(root.category)
                }
            }

            Text {
                text: qsTr("%1%").arg(Math.round(root.percent * 100))
                color: Theme.colorTextMuted
                font.pixelSize: Theme.textSize
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingMd

            Text {
                text: qsTr("File location")
                color: Theme.colorTextMuted
                font.pixelSize: Theme.textSize
                Layout.minimumWidth: content.columnWidth
            }

            Text {
                text: root.fileDir || "—"
                color: Theme.colorText
                font.pixelSize: Theme.textSize
                elide: Text.ElideMiddle
                Layout.fillWidth: true
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingMd

            Text {
                text: qsTr("MIME type")
                color: Theme.colorTextMuted
                font.pixelSize: Theme.textSize
                Layout.minimumWidth: content.columnWidth
            }

            Text {
                text: root.resolved.mimeType || "—"
                color: Theme.colorText
                font.pixelSize: Theme.textSize
                Layout.fillWidth: true
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingMd

            Text {
                text: qsTr("Status")
                color: Theme.colorTextMuted
                font.pixelSize: Theme.textSize
                Layout.minimumWidth: content.columnWidth
            }

            Text {
                text: root.statusText
                color: Theme.colorText
                font.pixelSize: Theme.textSize
                Layout.fillWidth: true
            }
        }

        RowLayout {
            visible: !!root.errorMessage && root.status === "error"
            Layout.fillWidth: true
            spacing: Theme.spacingMd

            Text {
                text: qsTr("Error")
                color: Theme.colorTextMuted
                font.pixelSize: Theme.textSize
                Layout.minimumWidth: content.columnWidth
            }

            Text {
                text: root.errorMessage || ""
                color: Theme.colorDanger
                font.pixelSize: Theme.textSize
                Layout.fillWidth: true
                wrapMode: Text.Wrap
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingSm

            RButton {
                variant: RButton.SecondaryVariant
                text: root.canResume ? qsTr("Resume") : qsTr("Pause")
                enabled: root.canPause || root.canResume
                iconSource: root.canResume ? "qrc:/icons/MdiLightPlay.svg" : "qrc:/icons/MdiLightPause.svg"
                onClicked: {
                    if (root.canResume) DownloadService.resume(root.gid)
                    else DownloadService.pause(root.gid)
                }
            }

            RButton {
                text: qsTr("Stop")
                enabled: root.canPause
                onClicked: DownloadService.stop(root.gid)
                iconSource: "qrc:/icons/MdiLightStop.svg"
            }

            Item {
                Layout.fillWidth: true
            }

            RButton {
                variant: RButton.DangerVariant
                iconOnly: true
                tooltip: qsTr("Remove the file forever")
                onClicked: deleteConfirm.openFor(root.Window.window)
                iconSource: "qrc:/icons/MaterialSymbolsLightDeleteForeverOutlineSharp.svg"
            }
        }
    }

    RDialog {
        id: deleteConfirm
        title: qsTr("Delete download")

        Text {
            text: qsTr("Remove %1 forever?").arg(root.displayName)
            color: Theme.colorText
            font.pixelSize: Theme.textSize
            wrapMode: Text.WordWrap
            Layout.fillWidth: true
        }

        footer: RowLayout {
            spacing: Theme.spacingMd

            RButton {
                text: qsTr("Cancel")
                onClicked: deleteConfirm.close()
            }
            RButton {
                text: qsTr("Delete")
                variant: RButton.DangerVariant
                onClicked: {
                    deleteConfirm.close();
                    root.removeRequested(root.gid);
                }
            }
        }
    }
}
