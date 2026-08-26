import QtQuick
import QtQuick.Controls as Controls
import ".."

Controls.Button {
    id: root

    enum Variant {
        BaseVariant,
        PrimaryVariant,
        SecondaryVariant,
        GhostVariant,
        WarningVariant,
        DangerVariant
    }

    property int variant: RButton.BaseVariant
    property bool outlined: false
    property url iconSource: ""
    property int iconSize: Theme.iconMd
    property bool iconOnly: false
    property string tooltip: ""
    property color foregroundColor: (root.variant === RButton.PrimaryVariant
        || root.variant === RButton.DangerVariant
        || (root.outlined && root.hovered))
        ? Theme.colorTextInverted
        : Theme.colorText
    property int cornerRadius: Theme.radiusSm

    readonly property color backgroundColor: {
        switch (root.variant) {
        case RButton.SecondaryVariant:
            return Theme.colorAccent
        case RButton.BaseVariant:
            return Theme.colorButtonBase
        case RButton.GhostVariant:
            return 'transparent'
        case RButton.WarningVariant:
            return Theme.colorWarning
        case RButton.DangerVariant:
            return Theme.colorDanger
        default:
            return Theme.colorPrimary
        }
    }

    readonly property color variantColor: {
        switch (root.variant) {
        case RButton.DangerVariant:
            return Theme.colorDanger
        case RButton.SecondaryVariant:
            return Theme.colorAccent
        case RButton.PrimaryVariant:
            return Theme.colorPrimary
        case RButton.WarningVariant:
            return Theme.colorWarning
        default:
            return Theme.colorBorder
        }
    }

    readonly property color hoveredColor: {
        switch (root.variant) {
        case RButton.SecondaryVariant:
            return Theme.colorAccentHover
        case RButton.DangerVariant:
            return Theme.colorDangerHover
        case RButton.GhostVariant:
        case RButton.BaseVariant:
            return Theme.colorButtonBaseHover
        case RButton.WarningVariant:
            return Theme.colorWarningHover
        default:
            return Theme.colorPrimaryHover
        }
    }

    icon.source: root.iconSource
    icon.width: root.iconSize
    icon.height: root.iconSize
    icon.color: root.hovered && root.enabled ? Theme.colorTextInverted : root.foregroundColor

    FontMetrics {
        id: buttonFontMetrics
        font: root.font
    }

    hoverEnabled: true

    MouseArea {
        anchors.fill: parent
        acceptedButtons: Qt.NoButton
        cursorShape: Qt.PointingHandCursor
    }
    horizontalPadding: root.text == "" && root.iconOnly ? Theme.spacingMd : Theme.spacingXl
    verticalPadding: Theme.spacingXs
    spacing: Theme.spacingXs
    implicitHeight: Math.max(
        Theme.touchTarget,
        root.implicitContentHeight + root.verticalPadding * 2
    )
    opacity: enabled ? 1 : 0.5
    palette.buttonText: root.hovered && root.enabled ? Theme.colorTextInverted : root.foregroundColor

    background: Rectangle {
        radius: root.cornerRadius
        border.width: root.outlined ? 1 : 0
        border.color: root.outlined ? root.variantColor : "transparent"
        color: root.outlined
            ? (root.hovered && root.enabled ? root.variantColor : "transparent")
            : (root.hovered && root.enabled ? root.hoveredColor : root.backgroundColor)
    }

    Controls.ToolTip {
        visible: root.hovered && root.tooltip !== ""
        delay: 500
        text: root.tooltip
        background: Rectangle {
            border.width: 1
            border.color: Theme.colorBorder
            radius: Theme.radiusSm
            color: Theme.colorSurface
        }
    }
}
