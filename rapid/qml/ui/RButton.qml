import QtQuick
import QtQuick.Controls as Controls
import ".."

Controls.Button {
    id: root

    property bool ghost: false
    property bool secondary: false
    property bool outlined: false
    property url iconSource: ""
    property int iconSize: Theme.iconSm
    property color iconColor: ghost ? Theme.textBody : Theme.textOnPrimary
    property int cornerRadius: Qt.platform.os === "linux" ? Theme.radiusSm : Theme.radiusMd

    icon.source: root.iconSource
    icon.width: root.iconSize
    icon.height: root.iconSize
    icon.color: root.iconColor

    hoverEnabled: true
    padding: Theme.spacingMd
    spacing: Theme.spaceXs
    implicitHeight: Math.max(
        Theme.touchTarget,
        root.implicitContentHeight + root.topPadding + root.bottomPadding
    )
    opacity: enabled ? 1 : 0.5
    palette.buttonText: root.iconColor

    background: Rectangle {
        radius: root.cornerRadius
        border.width: root.outlined ? 1 : 0
        border.color: root.outlined ? Theme.borderColor : ''
        color: root.ghost
            ? (root.hovered ? Theme.buttonSurfaceHover : Theme.buttonSurface)
            : (root.hovered ? Theme.primaryHover : Theme.primary)
    }
}
