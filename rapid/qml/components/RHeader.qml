import QtQuick
import ".."
import "../ui"

Rectangle {
    id: root

    signal addClicked()
    property alias searchText: searchField.text

    height: Math.max(searchField.implicitHeight, addButton.implicitHeight)
        + Theme.spaceSm * 2
    color: Theme.bg

    Rectangle {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        height: 1
        color: Theme.border
    }

    Row {
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        anchors.rightMargin: Theme.spaceSm
        spacing: Theme.spaceSm

        RTextField {
            id: searchField
            width: 400
            prefixIcon: "../icons/MdiMagnify.svg"
            placeholderText: "Search"
        }

        RButton {
            id: addButton
            height: parent.height
            text: "Add"
            iconSource: "../icons/MdiPlus.svg"
            onClicked: root.addClicked()
        }
    }
}
