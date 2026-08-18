pragma ComponentBehavior: Bound
import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import "../.."
import "../../ui"

// DownloadService and Clipboard are Python-registered context properties,
// invisible to qmllint's static analysis.
// qmllint disable unqualified
Window {
    id: root

    property Window owner: null
    property string defaultDir: ""
    property var resolvedUris: []
    property var options: ({})
    property var errors: ({})
    property var previousResolvedUri: []
    property bool isFetching: false
    property bool isEditing: false

    minimumWidth: 480
    height: scroll.height + dialogButtons.height + Theme.spacingLg + Theme.spacingLg
    visible: false
    title: qsTr("New download")
    color: Theme.colorBackground

// Click empty dialog space to drop focus from any text field.
    MouseArea {
        anchors.fill: parent
        z: -1
        onClicked: root.contentItem.forceActiveFocus()
    }

    // When linked to an owner (e.g. the main window), exit when it exits.
    Connections {
        target: root.owner
        enabled: root.owner !== null
        function onClosing() {
            root.close()
        }
    }

    // Spawn as its own window. `owner` is the window to die with (null = independent).
    function openFor(newOwner) {
        root.owner = newOwner ?? null
        root.show()
        root.resolve()
    }

    Timer {
        id: resolveTimer
        interval: 300
        onTriggered: root.resolve()
    }

    function pickFolder() {
        const dir = DownloadService.pickFolder(saveDir.text)
        if (dir) saveDir.text = dir
    }

    function submit() {
        DownloadService.download(resolvedUris)
        root.close()
    }

    function reset() {
        links.text = ""
        saveDir.text = root.defaultDir
        root.resolvedUris = []
        root.errors = ({})
        root.previousResolvedUri = []
        root.isFetching = false
        resolveTimer.stop()
        exitTimer.stop()
    }

    onClosing: root.reset()

    function removeUri(id) {
        root.resolvedUris = root.resolvedUris.filter((_, idx) => idx !== id)
    }

    function resolve() {
        if (links.text === "") return
        root.isFetching = true
        DownloadService.resolve(links.text)
    }

    function startTransition(newUris) {
        if (root.resolvedUris.length > 0) {
            root.previousResolvedUri = newUris
            uriBlock.leave()
            return
        }

        root.resolvedUris = newUris
    }

    Connections {
        target: DownloadService
        function onResolved(uris, errors) {
            root.errors = errors
            root.isFetching = false
            links.error = errors.url ?? ""
            root.startTransition(uris)
        }
    }

    Controls.ScrollView {
        id: scroll
        anchors {
            top: parent.top
            left: parent.left
            right: parent.right
            margins: Theme.spacingSm
        }
        clip: true
        Controls.ScrollBar.horizontal.policy: Controls.ScrollBar.AlwaysOff
        Controls.ScrollBar.vertical.policy: Controls.ScrollBar.AsNeeded

        ColumnLayout {
            id: form
            width: scroll.availableWidth
            spacing: Theme.spacingLg

            RTextField {
                id: links
                label: qsTr("URL")
                placeholderText: qsTr("https://example.com")
                selectByMouse: true
                wrapMode: TextEdit.NoWrap
                Layout.fillWidth: true

                onTextChanged: {
                    resolveTimer.restart()
                    root.errors = ({})
                    links.error = ""
                    if (links.text === "") root.startTransition([])
                }

                RButton {
                    iconSource: "qrc:/icons/MdiLightContentPaste.svg"
                    iconOnly: true
                    enabled: Clipboard.text.length > 0
                    onClicked: links.text = Clipboard.text
                    tooltip: qsTr("Paste from clipboard")
                }
            }

            ColumnLayout {
                id: uriBlock
                visible: root.resolvedUris.length > 0
                spacing: Theme.spacingSm
                property bool exiting: false

                function leave() {
                    uriBlock.exiting = true
                    exitTimer.restart()
                }

                Timer {
                    id: exitTimer
                    interval: 310
                    onTriggered: {
                        root.resolvedUris = root.previousResolvedUri
                        root.previousResolvedUri = []
                        uriBlock.exiting = false
                    }
                }

                Repeater {
                    id: uriRepeater
                    model: root.resolvedUris

                    ResolverUri {
                        required property int index
                        closable: root.resolvedUris.length > 1
                        onDeleteClicked: root.removeUri(index)
                        onSaved: (newUri) => {
                            root.resolvedUris[index] = newUri
                        }
                        onEditingChanged: () => {
                            root.isEditing = root.resolvedUris.some((_, i) => uriRepeater.itemAt(i)?.editing)
                        }

                        Connections {
                            target: uriBlock
                            function onExitingChanged() {
                                if (uriBlock.exiting) exit()
                                uriBlock.exiting = false
                            }
                        }
                    }
                }
            }

            ColumnLayout {
                id: saveDirLayout
                implicitWidth: scroll.availableWidth
                spacing: Theme.spacingMd
                RTextField {
                    id: saveDir
                    label: qsTr("Destination")
                    text: root.defaultDir
                    Layout.fillWidth: true

                    RButton {
                        iconSource: "qrc:/icons/MdiLightFolder.svg"
                        iconOnly: true
                        tooltip: qsTr("Pick folder")
                        onClicked: root.pickFolder()
                    }
                }
            }

        }
    }

    Controls.DialogButtonBox {
        id: dialogButtons
        padding: 0
        background: Rectangle { color: Theme.colorBackground }
        anchors {
            left: parent.left
            right: parent.right
            bottom: parent.bottom
            margins: Theme.spacingSm
        }
        alignment: Qt.AlignRight
        RButton {
            text: qsTr("Cancel")
            onClicked: root.close()
            enabled: !root.isFetching
        }
        RButton {
            id: downloadButton
            text: qsTr("Download")
            variant: RButton.PrimaryVariant
            enabled: links.text !== "" && !Object.keys(root.errors).length && root.resolvedUris.length > 0 && !root.isFetching && !root.isEditing
            onClicked: root.submit()
        }
    }
}
