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
    property string title: (root.modelData.title || root.modelData.filename || root.modelData.url) ?? ""
    property string url: root.modelData.url ?? ""
    property string category: root.modelData.category || "unknown"
    property int size: root.modelData.size || 0
    property bool checked: root.modelData.checked || false
    property bool closable: false
    property bool editing: false
    property bool readonly: false
    property bool loading: modelData.loading || false

    signal deleteClicked()
    signal saved(var newUri)

    function categoryFromName(name) {
        const ext = name.includes(".") ? name.split(".").pop().toLowerCase() : ""
        const categories = {
            audio: "mp3 wav flac ogg m4a aac wma opus",
            video: "mp4 mkv webm avi mov m4v flv wmv mpg mpeg 3gp",
            image: "png jpg jpeg gif bmp svg webp ico tiff avif",
            document: "pdf doc docx txt md epub rtf odt xls xlsx csv ppt pptx json xml html",
            compressed: "zip rar 7z gz tar bz2 xz zst iso",
            application: "exe msi apk dmg deb rpm bin jar",
        }
        for (const category of Object.keys(categories)) {
            if (categories[category].split(" ").includes(ext)) return category
        }
        return "unknown"
    }

    function updateCategory(name) {
        const category = root.categoryFromName(name)
        root.category = category
    }

    function saveTitle() {
        const name = titleEdit.text.trim()
        if (name.length == 0) {
            root.editing = false
            return
        }

        root.title = name
        root.category = root.categoryFromName(name)

        const data = root.modelData
        data.title = root.title
        data.filename = root.title
        data.category = root.category
        root.saved(data)
        root.editing = false
    }

    function undoChange() {
        root.title = root.modelData.title
        root.category = root.modelData.category
        titleEdit.text = root.modelData.title
        root.editing = false
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

    readonly property string categoryName: {
        switch (root.category) {
            case "audio": return "Audio";
            case "application": return "Application";
            case "compressed": return "Compressed";
            case "document": return "Document";
            case "image": return "Image";
            case "video": return "Video";
            default: return "Unknown";
        }
    }

    radius: Theme.radiusSm
    color: Theme.colorSurface
    border.color: root.loading ? Theme.colorBorder : Theme.categoryColor(root.category)
    implicitWidth: (root.loading ? loadingIndicator.implicitWidth + Theme.spacingSm : row.implicitWidth) + Theme.spacingMd * 2
    implicitHeight: (root.loading ? loadingIndicator.implicitHeight + Theme.spacingSm : row.implicitHeight) + Theme.spacingMd * 2
    Layout.fillWidth: true

    opacity: 0
    transform: Translate { id: slide; y: -Theme.spacingLg }

    function exit() {
        if (root.opacity === 1.0) exitAnimation.start()
    }

    function enter() {
        enterAnimation.start()
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

    Controls.Button {
        id: loadingIndicator
        anchors.centerIn: parent
        icon.source: "qrc:/icons/MdiLightLoading.svg"
        icon.width: Theme.touchTarget
        icon.height: Theme.touchTarget
        icon.color: Theme.colorTextMuted
        enabled: false
        visible: root.loading
        padding: 0
        background: null

        RotationAnimation on rotation {
            running: root.loading
            from: 0
            to: 360
            duration: 1000
            loops: Animation.Infinite
        }
    }

    RowLayout {
        id: row
        anchors.fill: parent
        anchors.leftMargin: Theme.spacingMd
        anchors.rightMargin: Theme.spacingMd
        anchors.topMargin: Theme.spacingMd
        anchors.bottomMargin: Theme.spacingMd
        spacing: Theme.spacingMd
        visible: !root.loading

        Controls.Button {
            id: categoryIcon
            icon.source: root.categoryIcon
            icon.width: height
            icon.height: height
            icon.color: Theme.categoryColor(root.category)
            enabled: false
            padding: Theme.spacingSm
            background: Rectangle {
                color: Theme.categoryColor(root.category)
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
                visible: !root.editing
                text: root.title
                color: Theme.colorText
                elide: Text.ElideMiddle
                Layout.fillWidth: true
                font.pixelSize: Theme.textSize
            }

            Controls.TextField {
                property int borderHeight: 1

                id: titleEdit
                visible: root.editing
                text: root.title
                selectByMouse: true
                Layout.fillWidth: true
                color: Theme.colorText
                font.pixelSize: Theme.textSize
                implicitHeight: Theme.textSize + Theme.spacingXs - titleEdit.borderHeight
                verticalAlignment: Text.AlignVCenter
                padding: 0
                Keys.onReturnPressed: root.saveTitle()
                Keys.onEnterPressed: root.saveTitle()
                onTextChanged: root.updateCategory(titleEdit.text)

                background: Rectangle {
                    id: activeBorder
                    color: "transparent"
                    Rectangle {
                        anchors.left: parent.left
                        anchors.right: parent.right
                        anchors.bottom: parent.bottom
                        height: titleEdit.borderHeight
                        color: titleEdit.activeFocus ? Theme.colorPrimary : Theme.colorTextMuted
                    }
                }
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
                    text: [DownloadService.formatSize(root.size), root.categoryName].filter(v => Boolean(v) && v !== "—").join(" · ")
                    color: Theme.colorTextMuted
                    font.pixelSize: Theme.textSizeSm
                    Layout.fillWidth: true
                }
            }
        }

        RowLayout {
            spacing: Theme.spacingXs

            RButton {
                visible: root.editing
                Layout.alignment: Qt.AlignVCenter
                iconSource: "qrc:/icons/MdiLightUndoVariant.svg"
                tooltip: "Undo change"
                iconOnly: true
                onClicked: root.undoChange()
            }

            RButton {
                visible: root.editing
                Layout.alignment: Qt.AlignVCenter
                iconSource: "qrc:/icons/MdiLightCheck.svg"
                tooltip: "Save"
                foregroundColor: Theme.colorSuccess
                iconOnly: true
                onClicked: root.saveTitle()
            }

            RButton {
                visible: !root.editing && !root.readonly
                Layout.alignment: Qt.AlignVCenter
                iconSource: "qrc:/icons/MdiLightPencil.svg"
                tooltip: "Edit"
                iconOnly: true
                onClicked: {
                    root.editing = true
                    Qt.callLater(() => {
                        titleEdit.forceActiveFocus()
                        titleEdit.selectAll()
                    })
                }
            }

            RButton {
                visible: root.closable && !root.editing
                Layout.alignment: Qt.AlignVCenter
                iconSource: "qrc:/icons/MaterialSymbolsLightCloseRounded.svg"
                tooltip: "Remove from download list"
                foregroundColor: Theme.colorDanger
                iconOnly: true
                onClicked: root.deleteClicked()
            }
        }

    }
}
