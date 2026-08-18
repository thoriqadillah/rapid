import QtQuick
import QtQuick.Layouts
import QtQuick.Controls as Controls
import "../.."
import "../../ui"

// DownloadService is a Python-registered context property, invisible to
// qmllint's static analysis.
// qmllint disable unqualified
Rectangle {
    id: root

    // Role values from DownloadService (a QAbstractListModel).
    required property var gid
    required property var status
    required property var resolved
    required property var files
    required property var totalLength
    required property var completedLength
    required property var downloadSpeed
    required property var errorMessage

    property int progressColumnWidth: 250
    property int speedColumnWidth: 125
    property int sizeColumnWidth: 125
    property int etaColumnWidth: 125

    color: rowMouse.containsMouse ? Theme.colorSurface : "transparent"
    implicitHeight: row.implicitHeight + Theme.spacingSm * 2

    readonly property string displayName: {
        const res = root.resolved
        if (res && (res.title || res.filename)) return res.title || res.filename
        const paths = Array.isArray(root.files) ? root.files : []
        const path = paths.length > 0 ? paths[0].path : ""
        if (path) return path.split("/").pop() || path
        return root.gid
    }

    readonly property double percent: {
        const total = Number(root.totalLength) || 0
        if (!total) return 0
        return Math.min(1, (Number(root.completedLength) || 0) / total)
    }

    readonly property bool isError: root.status === "error"
    readonly property bool isPaused: root.status === "paused"
    readonly property bool isComplete: root.status === "complete"

    readonly property string statusText: {
        if (root.isError) return qsTr("Error")
        if (root.isPaused) return qsTr("Paused")
        if (root.isComplete) return qsTr("Finished")
        return ""
    }

    readonly property string speedText: {
        if (root.isError) return "—"
        const speed = Number(root.downloadSpeed) || 0
        return speed > 0 ? DownloadService.formatSize(speed) + "/s" : "—"
    }

    readonly property string sizeText: {
        const total = Number(root.totalLength) || 0
        if (!total) return "—"
        const done = Number(root.completedLength) || 0
        return DownloadService.formatSize(done) + " / " + DownloadService.formatSize(total)
    }

    readonly property string etaText: {
        if (root.statusText) return root.statusText
        const total = Number(root.totalLength) || 0
        const done = Number(root.completedLength) || 0
        const speed = Number(root.downloadSpeed) || 0
        const remaining = total - done
        if (remaining <= 0 || speed <= 0) return "—"
        const secs = remaining / speed
        if (secs < 60) return qsTr("%1s").arg(Math.ceil(secs))
        const mins = Math.floor(secs / 60)
        if (mins < 60) return qsTr("%1m").arg(mins)
        return qsTr("%1h %2m").arg(Math.floor(mins / 60)).arg(mins % 60)
    }

    MouseArea {
        id: rowMouse
        anchors.fill: parent
        hoverEnabled: true
    }

    Rectangle {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        height: 1
        color: Theme.colorBorder
        opacity: 0.4
    }

    RowLayout {
        id: row
        anchors.fill: parent
        anchors.leftMargin: Theme.spacingLg
        anchors.rightMargin: Theme.spacingLg
        spacing: Theme.spacingMd

        Controls.Button {
            id: icon

            enabled: false
            opacity: 1
            padding: 0
            display: Controls.AbstractButton.IconOnly
            icon.source: "qrc:/icons/MdiSquareRounded.svg"
            icon.width: Theme.iconXs
            icon.height: Theme.iconXs
            icon.color: Theme.categoryColor(root.resolved.category)
            background: null
            Layout.preferredWidth: Theme.iconXs
            Layout.preferredHeight: Theme.iconXs
            Layout.alignment: Qt.AlignVCenter
        }

        Text {
            text: root.displayName
            color: Theme.colorText
            font.pixelSize: Theme.textSize
            elide: Text.ElideMiddle
            Layout.fillWidth: true
        }

        RowLayout {
            Layout.preferredWidth: root.progressColumnWidth
            Layout.minimumWidth: root.progressColumnWidth
            Layout.maximumWidth: root.progressColumnWidth
            spacing: Theme.spacingSm

            Rectangle {
                Layout.fillWidth: true
                implicitHeight: 10
                radius: height / 2
                color: Theme.colorSurface

                Rectangle {
                    width: parent.width * root.percent
                    height: parent.height
                    radius: height / 2
                    color: root.isError ? Theme.colorDanger : Theme.categoryColor(root.resolved.category)
                }
            }

            Text {
                text: qsTr("%1%").arg(Math.round(root.percent * 100))
                color: Theme.colorTextMuted
                font.pixelSize: Theme.textSize
                Layout.preferredWidth: percentMetrics.advanceWidth("100%")
                Layout.minimumWidth: percentMetrics.advanceWidth("100%")
                Layout.maximumWidth: percentMetrics.advanceWidth("100%")
            }

            FontMetrics {
                id: percentMetrics
                font.pixelSize: Theme.textSize
            }
        }

        Text {
            text: root.speedText
            color: Theme.colorText
            font.pixelSize: Theme.textSize
            elide: Text.ElideRight
            Layout.preferredWidth: root.speedColumnWidth
            Layout.minimumWidth: root.speedColumnWidth
            Layout.maximumWidth: root.speedColumnWidth
        }

        Text {
            text: root.sizeText
            color: Theme.colorText
            font.pixelSize: Theme.textSize
            Layout.preferredWidth: root.sizeColumnWidth
            Layout.minimumWidth: root.sizeColumnWidth
            Layout.maximumWidth: root.sizeColumnWidth
        }

        Text {
            text: root.etaText
            color: root.isError ? Theme.colorDanger : Theme.colorText
            font.pixelSize: Theme.textSize
            Layout.preferredWidth: root.etaColumnWidth
            Layout.minimumWidth: root.etaColumnWidth
            Layout.maximumWidth: root.etaColumnWidth
        }
    }
}
