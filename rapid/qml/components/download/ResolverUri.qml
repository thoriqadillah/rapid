import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import "../.."
import "../../ui"

// DownloadService is a Python-registered context property, invisible to
// qmllint's static analysis.
// qmllint disable unqualified
Rectangle {
    id: root

    required property var modelData
    property string title: root.modelData.title || root.modelData.filename || root.modelData.url
    property string url: root.modelData.url
    property string category: root.modelData.category || "unknown"
    property int size: root.modelData.size || 0
    property bool checked: root.modelData.checked || false
    property bool closable: false

    signal editClicked()
    signal deleteClicked()

    readonly property color categoryColor: {
        switch (root.category) {
            case "audio": return Theme.colorCategoryAudio;
            case "application": return Theme.colorCategoryApplication;
            case "compressed": return Theme.colorCategoryCompressed;
            case "video": return Theme.colorCategoryVideo;
            case "image": return Theme.colorCategoryImage;
            default: return Theme.colorCategoryUnknown;
        }
    }

    readonly property string categoryIcon: {
        switch (root.category) {
            case "audio": return "qrc:/icons/MdiLightMusic.svg";
            case "application": return "qrc:/icons/MdiLightConsole.svg";
            case "compressed": return "qrc:/icons/MdiLightViewModule.svg";
            case "document": return "qrc:/icons/MdiLightBook.svg";
            case "image": return "qrc:/icons/MdiLightPicture.svg";
            case "video": return "qrc:/icons/MdiLightFilmstrip.svg";
            default: return "qrc:/icons/MdiLightHelpCircle.svg";
        }
    }

    radius: Theme.radiusSm
    color: Theme.colorSurface
    border.color: root.categoryColor
    implicitWidth: row.implicitWidth + Theme.spacingMd * 2
    implicitHeight: row.implicitHeight + Theme.spacingMd * 2
    Layout.fillWidth: true

    opacity: 0
    transform: Translate { id: slide; y: -Theme.spacingLg }

    function exit() {
        if (root.opacity === 1.0) exitAnimation.start()
    }

    ParallelAnimation {
        id: enterAnimation
        running: false
        NumberAnimation { target: root; property: "opacity"; to: 1; duration: 300 }
        NumberAnimation { target: slide; property: "y"; to: 0; duration: 300; easing.type: Easing.OutCubic }
    }

    ParallelAnimation {
        id: exitAnimation
        running: false
        NumberAnimation { target: root; property: "opacity"; to: 0; duration: 300; easing.type: Easing.InCubic }
        NumberAnimation { target: slide; property: "y"; to: Theme.spacingLg; duration: 300; easing.type: Easing.InCubic }
    }

    Component.onCompleted: enterAnimation.start()

    RowLayout {
        id: row
        anchors.fill: parent
        anchors.leftMargin: Theme.spacingMd
        anchors.rightMargin: Theme.spacingMd
        anchors.topMargin: Theme.spacingMd
        anchors.bottomMargin: Theme.spacingMd
        spacing: Theme.spacingMd

        Controls.Button {
            id: categoryIcon
            icon.source: root.categoryIcon
            icon.width: height
            icon.height: height
            icon.color: root.categoryColor
            enabled: false
            padding: Theme.spacingSm
            background: Rectangle {
                color: root.categoryColor
                opacity: 0.15
                radius: Theme.radiusSm
            }
            implicitHeight: Theme.touchTarget
            Layout.fillHeight: true
            Layout.preferredWidth: height
        }

        ColumnLayout {
            Layout.fillWidth: true

            Text {
                text: root.title
                color: Theme.colorText
                elide: Text.ElideMiddle
                Layout.fillWidth: true
                font.pixelSize: Theme.textSize
            }

            ColumnLayout {
                Layout.fillWidth: true
                spacing: Theme.spacingXs

                Text {
                    text: root.url
                    color: Theme.colorTextMuted
                    font.pixelSize: Theme.textSizeSm
                    elide: Text.ElideMiddle
                    Layout.fillWidth: true
                }

                Text {
                    text: [DownloadService.formatSize(root.size), root.category].filter(Boolean).join(" · ")
                    color: Theme.colorTextMuted
                    font.pixelSize: Theme.textSizeSm
                    Layout.fillWidth: true
                }
            }
        }

        RowLayout {
            spacing: Theme.spacingXs

            RButton {
                Layout.alignment: Qt.AlignVCenter
                iconSource: "qrc:/icons/MdiLightPencil.svg"
                iconOnly: true
                onClicked: root.editClicked()
            }

            RButton {
                visible: root.closable
                Layout.alignment: Qt.AlignVCenter
                iconSource: "qrc:/icons/MaterialSymbolsLightCloseRounded.svg"
                foregroundColor: Theme.colorDanger
                iconOnly: true
                onClicked: root.deleteClicked()
            }
        }

    }
}
