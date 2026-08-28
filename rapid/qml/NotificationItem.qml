import QtQuick
import QtQuick.Controls as Controls
import "."

Controls.Popup {
    id: root

    property string title: ""
    property string message: ""
    property color typeColor: Theme.colorInfo
    property bool positionAnimationEnabled: false

    signal dismissed

    closePolicy: Controls.Popup.NoAutoClose
    x: window.width - width - Theme.spacingPageRight
    width: 400
    height: contentItem.implicitHeight + Theme.spacingMd * 2
    topPadding: Theme.spacingSm
    bottomPadding: Theme.spacingSm

    Behavior on y {
        enabled: root.positionAnimationEnabled

        NumberAnimation {
            duration: 200
            easing.type: Easing.OutCubic
        }
    }

    onOpened: dismissTimer.start()
    onClosed: dismissed()

    Timer {
        id: dismissTimer
        interval: 3000
        onTriggered: root.close()
    }

    enter: Transition {
        ParallelAnimation {
            NumberAnimation {
                property: "opacity"
                from: 0
                to: 1
                duration: 250
                easing.type: Easing.OutCubic
            }
            NumberAnimation {
                property: "x"
                from: window.width
                to: root.x
                duration: 250
                easing.type: Easing.OutCubic
            }
        }
    }

    exit: Transition {
        ParallelAnimation {
            NumberAnimation {
                property: "opacity"
                to: 0
                duration: 200
                easing.type: Easing.InCubic
            }
            NumberAnimation {
                property: "x"
                to: window.width
                duration: 200
                easing.type: Easing.InCubic
            }
        }
    }

    background: Rectangle {
        radius: Theme.radiusSm
        color: Theme.colorSurface
        border.width: 1
        border.color: Theme.colorBorder
    }

    contentItem: Item {
        implicitHeight: col.implicitHeight

        Rectangle {
            id: border
            width: 3
            anchors {
                top: parent.top
                bottom: parent.bottom
                left: parent.left
            }
            color: root.typeColor
            radius: width / 2
        }

        Column {
            id: col
            anchors {
                verticalCenter: parent.verticalCenter
                left: parent.left
                right: parent.right
                leftMargin: Theme.spacingMd + border.width
                rightMargin: Theme.spacingMd
            }
            spacing: Theme.spacingXs

            Text {
                text: root.title
                color: Theme.colorText
                font.pixelSize: Theme.textSize
                font.weight: Font.Medium
                elide: Text.ElideRight
                width: parent.width
            }

            Text {
                text: root.message
                color: Theme.colorText
                font.pixelSize: Theme.textSize
                wrapMode: Text.WordWrap
                width: parent.width
            }
        }
    }

    MouseArea {
        anchors.fill: col
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor
        onClicked: root.close()
        onEntered: dismissTimer.stop()
        onExited: dismissTimer.restart()
    }
}
