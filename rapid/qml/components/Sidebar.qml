pragma ComponentBehavior: Bound
import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import ".."

Rectangle {
    id: root

    signal destinationSelected(string destination)

    property string currentDestination: ""
    property bool open: true

    color: Theme.colorBase
    implicitWidth: 200
    width: open ? implicitWidth : 0
    clip: true

    Behavior on width {
        NumberAnimation {
            duration: 200
            easing.type: Easing.InOutQuad
        }
    }

    component SectionLabel: Text {
        color: Theme.colorTextMuted
        font.pixelSize: Theme.textSizeSm
        font.letterSpacing: 1
        text: ""
    }

    component SidebarItem: Item {
        id: item

        required property string destination
        required property string label
        property string count: ""
        property url iconSource: ""
        property color iconColor: Theme.colorTextMuted
        property bool categoryItem: false

        readonly property bool selected: root.currentDestination === destination

        implicitHeight: 36

        Rectangle {
            anchors.fill: parent
            radius: Theme.radiusSm
            color: item.selected || itemMouseArea.containsMouse ? Qt.lighter(root.color, 1.4) : "transparent"
        }

        RowLayout {
            anchors {
                fill: parent
                leftMargin: Theme.spacingSm
                rightMargin: Theme.spacingSm
            }
            spacing: Theme.spacingSm

            Controls.Button {
                enabled: false
                opacity: 1
                padding: 0
                display: Controls.AbstractButton.IconOnly
                icon.source: item.iconSource
                icon.width: item.categoryItem ? Theme.iconXs : Theme.iconSm
                icon.height: item.categoryItem ? Theme.iconXs : Theme.iconSm
                icon.color: item.selected && !item.categoryItem ? Theme.colorText : item.iconColor
                background: null
                Layout.preferredWidth: Theme.iconSm
                Layout.preferredHeight: Theme.iconSm
                Layout.alignment: Qt.AlignVCenter
            }

            Text {
                text: item.label
                color: item.selected || itemMouseArea.containsMouse ? Theme.colorText : Theme.colorTextMuted
                font.pixelSize: Theme.textSize
                elide: Text.ElideRight
                Layout.fillWidth: true
                Layout.alignment: Qt.AlignVCenter
            }

            Text {
                visible: item.count !== ""
                text: item.count
                color: item.selected || itemMouseArea.containsMouse ? Theme.colorText : Theme.colorTextMuted
                font.pixelSize: Theme.textSizeSm
                Layout.alignment: Qt.AlignVCenter
            }
        }

        MouseArea {
            id: itemMouseArea

            anchors.fill: parent
            hoverEnabled: true
            cursorShape: Qt.PointingHandCursor
            onClicked: {
                root.currentDestination = item.destination;
                root.destinationSelected(item.destination);
            }
        }
    }

    Rectangle {
        anchors.right: parent.right
        width: 1
        height: parent.height
        color: Theme.colorBorder
    }

    ColumnLayout {
        anchors {
            fill: parent
            topMargin: Theme.spacingSm
            bottomMargin: Theme.spacingPageBottom
            leftMargin: Theme.spacingPageLeft
            rightMargin: Theme.spacingPageRight
        }
        spacing: Theme.spacingXs

        SectionLabel {
            text: qsTr("LIBRARY")
            Layout.bottomMargin: Theme.spacingXs
        }

        SidebarItem {
            destination: ""
            label: qsTr("All Downloads")
            count: "24"
            iconSource: "../icons/MdiLightDownload.svg"
            iconColor: Theme.colorText
            Layout.fillWidth: true
        }

        SidebarItem {
            destination: "scheduled"
            label: qsTr("Scheduled")
            count: "3"
            iconSource: "../icons/MdiLightClock.svg"
            iconColor: Theme.colorText
            Layout.fillWidth: true
        }

        SectionLabel {
            text: qsTr("CATEGORIES")
            Layout.topMargin: Theme.spacingMd
            Layout.bottomMargin: Theme.spacingXs
        }

        SidebarItem {
            destination: "audio"
            label: qsTr("Audios")
            iconSource: "../icons/MdiSquareRounded.svg"
            iconColor: Theme.colorCategoryAudio
            Layout.fillWidth: true
            categoryItem: true
        }

        SidebarItem {
            destination: "document"
            label: qsTr("Documents")
            iconSource: "../icons/MdiSquareRounded.svg"
            iconColor: Theme.colorCategoryDocs
            Layout.fillWidth: true
            categoryItem: true
        }

        SidebarItem {
            destination: "image"
            label: qsTr("Images")
            iconSource: "../icons/MdiSquareRounded.svg"
            iconColor: Theme.colorCategoryImages
            Layout.fillWidth: true
            categoryItem: true
        }

        SidebarItem {
            destination: "video"
            label: qsTr("Videos")
            iconSource: "../icons/MdiSquareRounded.svg"
            iconColor: Theme.colorCategoryVideo
            Layout.fillWidth: true
            categoryItem: true
        }

        Item {
            Layout.fillHeight: true
            Layout.minimumHeight: Theme.spacingMd
        }

        SidebarItem {
            destination: "settings"
            label: qsTr("Settings")
            iconSource: "../icons/MdiLightSettings.svg"
            iconColor: Theme.colorText
            Layout.fillWidth: true
        }
    }
}
