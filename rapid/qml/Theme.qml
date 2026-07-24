pragma Singleton
import QtQuick

QtObject {
    readonly property SystemPalette palette: SystemPalette {}

    // Colors — driven by the OS palette so light/dark just works
    readonly property color bgColor: palette.window
    readonly property color surface: palette.base
    readonly property color surfaceAlt: palette.alternateBase
    readonly property color borderColor: palette.mid

    readonly property color primary: palette.highlight
    readonly property color primaryHover: Qt.darker(palette.highlight, 1.1)
    readonly property color primaryTint: Qt.lighter(palette.highlight, 1.6)
    readonly property color textOnPrimary: palette.highlightedText

    readonly property color buttonSurface: palette.button
    readonly property color buttonSurfaceHover: Qt.lighter(palette.button, 1.5)
    readonly property color buttonText: palette.buttonText

    readonly property color textPrimary: palette.windowText
    readonly property color textBody: palette.text
    readonly property color textMuted: Qt.rgba(palette.text.r, palette.text.g, palette.text.b, 0.6)
    readonly property color textFaint: Qt.rgba(palette.text.r, palette.text.g, palette.text.b, 0.4)

    // Category colors — semantic file-type accents, kept fixed regardless of theme
    readonly property color categoryVideo: "#ee7b57"
    readonly property color categoryDocs: "#5b9bd5"
    readonly property color categoryMusic: "#4fb783"
    readonly property color categoryImages: "#b98bd6"

    // Border radius
    readonly property int radiusSm: 4
    readonly property int radiusMd: 8
    readonly property int radiusLg: 12
    readonly property int radiusXl: 14
    readonly property int radiusPill: 20
    readonly property int radiusSheet: 22

    // Spacing
    readonly property int spaceXs: 4
    readonly property int spaceSm: 8
    readonly property int spacingMd: 12
    readonly property int spaceLg: 16
    readonly property int spaceXl: 24

    // Adaptive layout — values are device-independent pixels
    readonly property int breakpointMedium: 600
    readonly property int breakpointExpanded: 840
    readonly property int touchTarget: 36

    // Icons
    readonly property int iconSm: 16
    readonly property int iconMd: 24
    readonly property int iconLg: 32
    readonly property int iconXl: 48
}
