import QtQuick
import QtQuick.Layouts
import ".."
import "../ui"

Rectangle {
    id: root

    signal addClicked()
    property alias searchText: searchField.text
    property bool compact: width < Theme.breakpointMedium

    implicitHeight: headerLayout.implicitHeight + Theme.spaceSm * 2
    height: implicitHeight
    color: Theme.bgColor

    Rectangle {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        height: 1
        color: Theme.borderColor
    }

    GridLayout {
        id: headerLayout

        anchors {
            left: parent.left
            right: parent.right
            verticalCenter: parent.verticalCenter
            leftMargin: Theme.spaceSm
            rightMargin: Theme.spaceSm
        }
        columns: root.compact ? 1 : 2
        rowSpacing: Theme.spaceSm
        columnSpacing: Theme.spaceSm

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
            visible: !root.compact
            ghost: true
            Layout.fillWidth: root.compact
            Layout.minimumHeight: Theme.touchTarget
            iconSource: "../icons/MdiPlus.svg"
            onClicked: root.addClicked()
        }
    }
}
