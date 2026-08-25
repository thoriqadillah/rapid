pragma ComponentBehavior: Bound
import QtQuick
import QtQuick.Layouts
import ".."

Rectangle {
    id: root

    signal destinationSelected(string destination)

    property string currentDestination: ""
    property bool open: true

    function activate(destination) {
        currentDestination = destination;
        root.destinationSelected(destination);
    }

    default property alias content: contentColumn.data

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

    Rectangle {
        anchors.right: parent.right
        width: 1
        height: parent.height
        color: Theme.colorBorder
    }

    ColumnLayout {
        id: contentColumn

        anchors {
            fill: parent
            topMargin: Theme.spacingSm
            bottomMargin: Theme.spacingPageBottom
            leftMargin: Theme.spacingPageLeft
            rightMargin: Theme.spacingPageRight
        }
        spacing: Theme.spacingXs
    }
}
