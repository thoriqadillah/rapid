pragma Singleton
import QtQuick

QtObject {

    readonly property color colorBackground: "#1e1f29"
    readonly property color colorSurface: "#282A36"
    readonly property color colorBorder: Qt.darker("#44475A", 1.1)

    readonly property color colorPrimary: "#BD93F9"
    readonly property color colorPrimaryHover: Qt.darker(Theme.colorPrimary, 1.1)
    readonly property color colorAccent: "#FF79C6"
    readonly property color colorAccentHover: Qt.darker(Theme.colorAccent, 1.1)
    readonly property color colorDanger: "#FF5555"
    readonly property color colorDangerHover: Qt.darker(Theme.colorDanger, 1.1)
    readonly property color colorSuccess: "#50FA7B"
    readonly property color colorSuccessHover: Qt.darker(Theme.colorSuccess, 1.1)
    readonly property color colorWarning: "#FFB86C"
    readonly property color colorWarningHover: Qt.darker(Theme.colorWarning, 1.1)
    readonly property color colorInfo: "#8BE9FD"
    readonly property color colorInfoHover: Qt.darker(Theme.colorInfo, 1.1)


    // button colors
    readonly property color colorButtonBase: Theme.colorSurface
    readonly property color colorButtonBaseHover: Qt.lighter(Theme.colorButtonBase, 1.1)

    // text colors
    readonly property color colorText: "#F8F8F2"
    readonly property color colorTextInverted: "#F8F8F2"
    readonly property color colorTextMuted: Qt.rgba(Theme.colorText.r, Theme.colorText.g, Theme.colorText.b, 0.6)

    // input colors
    readonly property color colorInputBackground: Theme.colorSurface

    readonly property color colorCategoryAudio: "#50FA7B"
    readonly property color colorCategoryApplication: "#F1FA8C"
    readonly property color colorCategoryCompressed: "#FF5555"
    readonly property color colorCategoryDocument: "#8BE9FD"
    readonly property color colorCategoryImage: "#BD93F9"
    readonly property color colorCategoryVideo: "#FFB86C"
    readonly property color colorCategoryUnknown: "#F8F8F2"

    function categoryColor(category: string): color {
        switch (category) {
            case "audio": return colorCategoryAudio
            case "application": return colorCategoryApplication
            case "compressed": return colorCategoryCompressed
            case "document": return colorCategoryDocument
            case "image": return colorCategoryImage
            case "video": return colorCategoryVideo
            default: return colorCategoryUnknown
        }
    }

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
    readonly property int iconSm: 18
    readonly property int iconMd: 24
    readonly property int iconLg: 32
    readonly property int iconXl: 48
}
