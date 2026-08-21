import QtQuick
import QtQuick.Layouts
import "../.."
import "../../ui"

// DownloadService is a Python-registered context property, invisible to
// qmllint's static analysis.
// qmllint disable unqualified
Rectangle {
    id: root

    required property var status
    required property var totalLength
    required property var completedLength
    required property var downloadSpeed

    property int minimumWidth: 70

    readonly property bool isComplete: root.status === "complete"
    readonly property bool isPaused: root.status === "paused"
    readonly property bool isError: root.status === "error"

    readonly property string displayText: {
        if (root.isComplete) return qsTr("Finished")
        if (root.isPaused) return qsTr("Paused")
        if (root.isError) return qsTr("Error")
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

    readonly property color statusColor: {
        if (root.isComplete) return Theme.colorSuccess
        if (root.isError) return Theme.colorDanger
        return Theme.colorText
    }

    readonly property bool showPill: root.isComplete || root.isError || root.isPaused

    color: "transparent"
    implicitWidth: pill.implicitWidth
    implicitHeight: pill.implicitHeight
    Layout.minimumWidth: root.minimumWidth

    Rectangle {
        id: pill
        visible: root.showPill
        anchors.left: parent.left
        anchors.verticalCenter: parent.verticalCenter
        width: Math.max(label.implicitWidth + Theme.spacingMd * 2, root.minimumWidth)
        implicitHeight: label.implicitHeight + Theme.spacingXs * 2
        radius: Theme.radiusPill
        color: "transparent"
        border.width: 1
        border.color: root.statusColor

        Text {
            id: label
            anchors.centerIn: parent
            text: root.displayText
            color: root.statusColor
            font.pixelSize: Theme.textSizeSm
        }
    }

    Text {
        visible: !root.showPill
        text: root.displayText
        anchors.left: parent.left
        anchors.verticalCenter: parent.verticalCenter
        font.pixelSize: Theme.textSize
        color: root.statusColor
    }
}
