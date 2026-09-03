import QtQuick
import QtQuick.Controls as Controls
import ".."

Controls.Switch {
    id: root

    property color activeColor: Theme.colorPrimary
    property color inactiveColor: Theme.colorBorder

    hoverEnabled: true

    MouseArea {
        anchors.fill: parent
        acceptedButtons: Qt.NoButton
        cursorShape: Qt.PointingHandCursor
    }

    implicitWidth: 36
    implicitHeight: 20
    padding: 0
    contentItem: Item {}
    indicator: Rectangle {
        width: 36
        height: 20
        radius: Theme.radiusPill
        color: root.checked ? root.activeColor : root.inactiveColor
        border.width: 1
        border.color: Qt.darker(root.checked ? root.activeColor : root.inactiveColor, 1.2)
        anchors.verticalCenter: parent.verticalCenter

        Rectangle {
            x: root.checked ? parent.width - width - 2 : 2
            y: 2
            width: 16
            height: 16
            radius: Theme.radiusPill
            color: Theme.colorText
            Behavior on x {
                NumberAnimation {
                    duration: 120
                }
            }
        }
    }
}
