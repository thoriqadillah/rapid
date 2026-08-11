import QtQuick
import QtQuick.Layouts
import ".."
import "../ui"

Rectangle {
    id: root

    signal addClicked()
    signal menuClicked()

    property alias searchText: searchField.text
    property int preferredSearchWidth: 350

    implicitHeight: headerLayout.implicitHeight + Theme.spacingSm * 2
    height: implicitHeight
    color: Theme.colorBackground

    Rectangle {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        height: 1
        color: 'transparent'
    }

    RowLayout {
        id: headerLayout

        anchors {
            left: parent.left
            right: parent.right
            verticalCenter: parent.verticalCenter
            leftMargin: Theme.spacingPageLeft
            rightMargin: Theme.spacingPageRight
        }
        spacing: Theme.spacingSm

        RButton {
            id: menuButton
            iconSource: "../icons/MdiLightMenu.svg"
            variant: RButton.GhostVariant
            iconOnly: true
            Layout.minimumHeight: Theme.touchTarget
            onClicked: root.menuClicked()
        }

        Item {
            Layout.fillWidth: true
        }

        RTextField {
            id: searchField

            Layout.preferredWidth: root.preferredSearchWidth
            Layout.maximumWidth: root.preferredSearchWidth
            Layout.minimumWidth: 0
            Layout.minimumHeight: Theme.touchTarget
            prefixIcon: "../icons/MdiMagnify.svg"
            placeholderText: "Search..."
        }

        RButton {
            id: addButton
            text: "New"
            variant: RButton.PrimaryVariant
            Layout.minimumHeight: Theme.touchTarget
            iconSource: "../icons/MdiPlus.svg"
            onClicked: root.addClicked()
        }
    }
}
