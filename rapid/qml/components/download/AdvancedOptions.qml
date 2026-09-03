import QtQuick
import QtQuick.Layouts
import "../.."
import "../../ui"

// Expandable "Advanced options" accordion for the download dialog. Holds
// editable HTTP headers and cookies passed to the resolver and downloader.
ColumnLayout {
    id: root

    property bool expanded: false
    property alias headers: headersEditor
    property alias cookies: cookiesEditor
    spacing: 0

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

    Item {
        id: advancedWrapper
        Layout.topMargin: root.expanded ? Theme.spacingMd : 0
        Layout.fillWidth: true
        Layout.preferredHeight: root.expanded ? advancedContent.implicitHeight : 0
        clip: true

        Behavior on Layout.preferredHeight {
            NumberAnimation {
                duration: 100
            }
        }

        ColumnLayout {
            id: advancedContent
            width: parent.width
            opacity: root.expanded ? 1 : 0
            y: root.expanded ? 0 : -root.implicitHeight
            spacing: Theme.spacingMd

            Behavior on opacity {
                NumberAnimation {
                    duration: 100
                }
            }
            Behavior on y {
                NumberAnimation {
                    duration: 100
                }
            }

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
}
