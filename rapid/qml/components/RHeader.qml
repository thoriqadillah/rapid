import QtQuick
import QtQuick.Layouts
import ".."
import "../ui"

Rectangle {
    id: root

    signal addClicked()
    property alias searchText: searchField.text
    property bool compact: width < Theme.breakpointMd

    implicitHeight: headerLayout.implicitHeight + Theme.spacingSm * 2
    height: implicitHeight
    color: Theme.colorBackground

    Rectangle {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        height: 1
        color: Theme.colorBorder
    }

    GridLayout {
        id: headerLayout

        anchors {
            left: parent.left
            right: parent.right
            verticalCenter: parent.verticalCenter
            leftMargin: Theme.spacingSm
            rightMargin: Theme.spacingSm
        }
        rowSpacing: Theme.spacingSm
        columnSpacing: Theme.spacingSm

        RTextField {
            id: searchField

            Layout.fillWidth: true
            Layout.minimumWidth: 0
            Layout.minimumHeight: Theme.touchTarget
            prefixIcon: "../icons/MdiMagnify.svg"
            placeholderText: "Search"
        }

        RButton {
            id: addButton
            text: !root.compact ? "Add" : ""
            variant: RButton.PrimaryVariant
            Layout.minimumHeight: Theme.touchTarget
            iconSource: "../icons/MdiPlus.svg"
            onClicked: root.addClicked()
        }
    }
}
