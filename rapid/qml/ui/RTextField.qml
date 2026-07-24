import QtQuick
import QtQuick.Controls as Controls
import ".."

Controls.TextField {
    id: root

    property url prefixIcon: ""
    property url suffixIcon: ""
    property int iconSize: Theme.iconSm
    property color iconColor: Theme.colorTextMuted
    readonly property int cornerRadius: Qt.platform.os === "linux" ? Theme.radiusSm : Theme.radiusMd

    selectByMouse: true
    color: Theme.colorText
    placeholderTextColor: Theme.colorTextMuted
    padding: Theme.spacingMd
    leftPadding: Theme.spacingMd + (prefixIconButton.visible ? root.iconSize + Theme.spacingXs : 0)
    rightPadding: Theme.spacingMd + (suffixIconButton.visible ? root.iconSize + Theme.spacingXs : 0)
    implicitHeight: Math.max(
        Theme.touchTarget,
        root.contentHeight + root.topPadding + root.bottomPadding
    )

    background: Rectangle {
        radius: root.cornerRadius
        color: Theme.colorInputBackground
        border.width: 1
        border.color: root.activeFocus ? Theme.colorPrimary : Theme.colorBorder
    }

    Controls.Button {
        id: prefixIconButton
        anchors.left: root.left
        anchors.leftMargin: Theme.spacingMd
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
        anchors.rightMargin: Theme.spacingMd
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
