import QtQuick
import QtQuick.Layouts
import "../.."
import "../../ui"

// Expandable "Advanced options" accordion for the download dialog. Holds
// editable HTTP headers and cookies passed to the resolver and downloader.
ColumnLayout {
    id: root

    spacing: Theme.spacingMd
    property bool expanded: false
    property alias headers: headersEditor
    property alias cookies: cookiesEditor

    RButton {
        Layout.fillWidth: true
        variant: RButton.GhostVariant
        iconSource: "qrc:/icons/MdiLightChevronDown.svg"
        iconSize: Theme.iconMd
        text: qsTr("Advanced options")
        horizontalPadding: Theme.spacingSm
        verticalPadding: Theme.spacingXs
        onClicked: root.expanded = !root.expanded
    }

    ColumnLayout {
        Layout.fillWidth: true
        visible: root.expanded
        spacing: Theme.spacingMd

        KVEditor {
            id: headersEditor
            Layout.fillWidth: true
            title: qsTr("Headers")
        }

        KVEditor {
            id: cookiesEditor
            Layout.fillWidth: true
            title: qsTr("Cookies")
        }
    }
}
