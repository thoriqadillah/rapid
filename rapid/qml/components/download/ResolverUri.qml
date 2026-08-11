import QtQuick
import QtQuick.Layouts
import "../.."

RowLayout {
    id: root

    required property var modelData
    property string title: root.modelData.title || root.modelData.filename || root.modelData.url
    property string url: root.modelData.url
    property string kind: root.modelData.kind || "unknown"
    property int size: root.modelData.size || 0
    readonly property color kindColor:
        root.kind === "video" ? Theme.colorCategoryVideo :
        root.kind === "audio" ? Theme.colorCategoryAudio :
        root.kind === "image" ? Theme.colorCategoryImages : Theme.colorCategoryDocs

    function formatSize(bytes) {
        if (!bytes) return ""
        const units = ["B", "KB", "MB", "GB", "TB"]
        let i = 0
        let value = bytes
        while (value >= 1024 && i < units.length - 1) {
            value /= 1024
            i++
        }
        return value.toFixed(value >= 100 || i === 0 ? 0 : 1) + " " + units[i]
    }

    spacing: Theme.spacingMd

    Rectangle {
        implicitWidth: 3
        Layout.fillHeight: true
        color: root.kindColor
        radius: 2
    }

    ColumnLayout {
        Layout.fillWidth: true
        spacing: Theme.spacingSm

        Text {
            text: root.title
            color: Theme.colorText
            elide: Text.ElideMiddle
            Layout.fillWidth: true
            font.pixelSize: Theme.textSize
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingXs

            Text {
                text: root.url
                color: Theme.colorTextMuted
                font.pixelSize: Theme.textSizeSm
                elide: Text.ElideMiddle
                Layout.fillWidth: true
            }

            Text {
                text: [root.formatSize(root.size), root.kind].filter(Boolean).join(" · ")
                color: Theme.colorTextMuted
                font.pixelSize: Theme.textSizeSm
                Layout.fillWidth: true
            }
        }

    }
}
