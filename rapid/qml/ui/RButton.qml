import QtQuick
import QtQuick.Controls as Controls
import ".."

Controls.Button {
    id: root

    property bool secondary: false
    property url iconSource: ""
    property int iconSize: Theme.iconSm
    property color iconColor: secondary ? Theme.textBody : Theme.textOnPrimary
    property int cornerRadius: Qt.platform.os === "linux" ? Theme.radiusSm : Theme.radiusMd

    icon.source: root.iconSource
    icon.width: root.iconSize
    icon.height: root.iconSize
    icon.color: root.iconColor

    hoverEnabled: true
    padding: Theme.spaceMd
    spacing: Theme.spaceXs
    opacity: enabled ? 1 : 0.5
    palette.buttonText: root.iconColor

    background: Rectangle {
        radius: root.cornerRadius
        border.width: root.secondary ? 1 : 0
        border.color: Theme.border
        color: root.secondary
            ? (root.hovered ? Theme.buttonSurfaceHover : Theme.buttonSurface)
            : (root.hovered ? Theme.primaryHover : Theme.primary)
    }
}
