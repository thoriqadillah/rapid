pragma Singleton
import QtQuick

QtObject {
    readonly property SystemPalette palette: SystemPalette {}

    // Colors — driven by the OS palette so light/dark just works
    readonly property color colorBackground: palette.base
    readonly property color colorBase: palette.window
    readonly property color colorBorder: palette.mid
    readonly property color colorPrimary: palette.highlight
    readonly property color colorPrimaryHover: Qt.darker(palette.highlight, 1.1)

    readonly property color colorSurface: palette.button

    // button colors
    readonly property color colorButtonBase: Theme.colorSurface
    readonly property color colorButtonBaseHover: Qt.lighter(palette.button, 1.5)

    // text colors
    readonly property color colorText: palette.text
    readonly property color colorTextInverted: palette.highlightedText
    readonly property color colorTextMuted: Qt.rgba(palette.text.r, palette.text.g, palette.text.b, 0.6)

    // input colors
    readonly property color colorInputBackground: Theme.colorSurface

    readonly property color colorCategoryVideo: "#ee7b57"
    readonly property color colorCategoryDocs: "#5b9bd5"
    readonly property color colorCategoryAudio: "#4fb783"
    readonly property color colorCategoryImages: "#b98bd6"

    // Text size
    readonly property int textSize: 14
    readonly property int textSizeSm: 10

    // Border radius
    readonly property int radiusSm: 4
    readonly property int radiusMd: 8
    readonly property int radiusLg: 12
    readonly property int radiusXl: 14
    readonly property int radiusPill: 20
    readonly property int radiusSheet: 22

    // Spacing
    readonly property int spacingXs: 4
    readonly property int spacingSm: 8
    readonly property int spacingMd: 12
    readonly property int spacingLg: 16
    readonly property int spacingXl: 24

    // page spacing
    readonly property int spacingPageLeft: Theme.spacingSm
    readonly property int spacingPageRight: Theme.spacingSm
    readonly property int spacingPageTop: Theme.spacingSm
    readonly property int spacingPageBottom: Theme.spacingSm

    // Adaptive layout — values are device-independent pixels
    readonly property int breakpointMd: 600
    readonly property int breakpointLg: 840
    readonly property int touchTarget: 36

    // Icons
    readonly property int iconXs: 10
    readonly property int iconSm: 16
    readonly property int iconMd: 20
    readonly property int iconLg: 32
    readonly property int iconXl: 48
}
