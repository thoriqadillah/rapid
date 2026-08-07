import QtQuick
import QtQuick.Controls as Controls
import ".."

Controls.Button {
    id: root

    enum Variant {
        BaseVariant,
        PrimaryVariant,
        SecondaryVariant,
        GhostVariant
    }

    property int variant: RButton.BaseVariant
    property bool outlined: false
    property url iconSource: ""
    property int iconSize: Theme.iconMd
    property color foregroundColor: root.variant === RButton.PrimaryVariant
        ? Theme.colorTextInverted
        : Theme.colorText
    property int cornerRadius: Qt.platform.os === "linux" ? Theme.radiusSm : Theme.radiusMd

    readonly property color backgroundColor: {
        switch (root.variant) {
        case RButton.SecondaryVariant:
            return Theme.colorButtonBase
        case RButton.BaseVariant:
            return Theme.colorButtonBase
        case RButton.GhostVariant:
            return 'transparent'
        default:
            return Theme.colorPrimary
        }
    }

    readonly property color hoveredColor: {
        switch (root.variant) {
        case RButton.SecondaryVariant:
        case RButton.GhostVariant:
        case RButton.BaseVariant:
            return Theme.colorButtonBaseHover
        default:
            return Theme.colorPrimaryHover
        }
    }

    icon.source: root.iconSource
    icon.width: root.iconSize
    icon.height: root.iconSize
    icon.color: root.foregroundColor

    FontMetrics {
        id: buttonFontMetrics
        font: root.font
    }

    hoverEnabled: true
    horizontalPadding: root.text == "" ? Theme.spacingMd : Theme.spacingXl
    verticalPadding: Theme.spacingXs
    spacing: Theme.spacingXs
    implicitHeight: Math.max(
        Theme.touchTarget,
        Math.max(buttonFontMetrics.height, Theme.iconSm) + Theme.spacingMd * 2,
        root.implicitContentHeight + root.verticalPadding * 2
    )
    opacity: enabled ? 1 : 0.5
    palette.buttonText: root.foregroundColor

    background: Rectangle {
        radius: root.cornerRadius
        border.width: root.outlined ? 1 : 0
        border.color: root.outlined ? Theme.colorBorder : "transparent"
        color: root.hovered
            ? root.hoveredColor
            : root.backgroundColor
    }
}
