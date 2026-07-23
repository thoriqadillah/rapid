import QtQuick
import QtQuick.Controls as Controls
import ".."

Controls.TextField {
    id: root

    property url prefixIcon: ""
    property url suffixIcon: ""
    property int iconSize: Theme.iconSm
    property color iconColor: Theme.textFaint
    readonly property int cornerRadius: Qt.platform.os === "linux" ? Theme.radiusSm : Theme.radiusMd

    selectByMouse: true
    color: Theme.textBody
    placeholderTextColor: Theme.textFaint
    padding: Theme.spaceMd
    leftPadding: Theme.spaceMd + (prefixIconButton.visible ? root.iconSize + Theme.spaceXs : 0)
    rightPadding: Theme.spaceMd + (suffixIconButton.visible ? root.iconSize + Theme.spaceXs : 0)

    background: Rectangle {
        radius: root.cornerRadius
        color: Theme.surface
        border.width: 1
        border.color: root.activeFocus ? Theme.primary : Theme.border
    }

    Controls.Button {
        id: prefixIconButton
        anchors.left: root.left
        anchors.leftMargin: Theme.spaceMd
        anchors.verticalCenter: root.verticalCenter
        width: root.iconSize
        height: root.iconSize
        visible: root.prefixIcon.toString() !== ""
        enabled: false
        opacity: 1
        padding: 0
        display: Controls.AbstractButton.IconOnly
        icon.source: root.prefixIcon
        icon.width: root.iconSize
        icon.height: root.iconSize
        icon.color: root.iconColor
        background: null
    }

    Controls.Button {
        id: suffixIconButton
        anchors.right: root.right
        anchors.rightMargin: Theme.spaceMd
        anchors.verticalCenter: root.verticalCenter
        width: root.iconSize
        height: root.iconSize
        visible: root.suffixIcon.toString() !== ""
        enabled: false
        opacity: 1
        padding: 0
        display: Controls.AbstractButton.IconOnly
        icon.source: root.suffixIcon
        icon.width: root.iconSize
        icon.height: root.iconSize
        icon.color: root.iconColor
        background: null
    }
}
