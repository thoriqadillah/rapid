import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import "../.."
import "../../ui"

// Clipboard and DownloadDialog are Python-registered context properties,
// invisible to qmllint's static analysis.
// qmllint disable unqualified
Rectangle {
    id: root

    visible: Clipboard.url.length > 0
    implicitHeight: Theme.touchTarget
    color: Qt.rgba(Theme.colorInfo.r, Theme.colorInfo.g, Theme.colorInfo.b, 0.15)
    radius: Theme.radiusSm

    signal download

    RowLayout {
        anchors.fill: parent
        anchors.leftMargin: Theme.spacingMd
        anchors.rightMargin: Theme.spacingMd
        spacing: Theme.spacingSm

        Controls.Button {
            icon.source: "qrc:/icons/MdiLightContentPaste.svg"
            icon.width: Theme.iconSm
            icon.height: Theme.iconSm
            icon.color: Theme.colorInfo
            enabled: false
            padding: 0
            display: Controls.AbstractButton.IconOnly
            background: null
        }

        Text {
            Layout.fillWidth: true
            text: qsTr("There is one url in the clipboard")
            color: Theme.colorInfo
            font.pixelSize: Theme.textSize
        }

        RButton {
            iconSource: "qrc:/icons/MdiLightDownload.svg"
            iconSize: Theme.iconSm
            link: true
            text: qsTr("Download")
            variant: RButton.InfoVariant
            onClicked: download()
        }

        RButton {
            iconSource: "qrc:/icons/MaterialSymbolsLightCloseRounded.svg"
            iconSize: Theme.iconSm
            link: true
            iconOnly: true
            variant: RButton.InfoVariant
            onClicked: Clipboard.copy("")
        }
    }
}
