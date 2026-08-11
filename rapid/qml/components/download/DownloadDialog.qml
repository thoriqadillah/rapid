pragma ComponentBehavior: Bound
import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import "../.."
import "../../ui"

Window {
    id: root

    property Window owner: null
    property string defaultDir: ""
    property var resolvedUris: []
    property var options: ({})
    property var errors: ({})
    property bool isFetching: false

    minimumWidth: 480
    height: scroll.height + dialogButtons.height + Theme.spacingLg + Theme.spacingLg
    visible: false
    title: qsTr("New download")
    color: Theme.colorBackground

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
        DownloadService.download(resolvedUris, options)
        root.close()
    }

    function reset() {
        links.text = ""
        saveDir.text = root.defaultDir
    }

    function resolve() {
        root.isFetching = true
        DownloadService.resolve(links.text)
    }

    Connections {
        target: DownloadService
        function onResolved(uris, errors) {
            root.resolvedUris = uris
            root.errors = errors
            root.isFetching = false
            links.error = errors.url ?? ""
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
                wrapMode: TextEdit.Wrap
                Layout.fillWidth: true

                onTextChanged: {
                    resolveTimer.restart()
                    root.errors = ({})
                    root.resolvedUris = []
                    links.error = ""
                }

                RButton { iconSource: "../../icons/MdiLightContentPaste.svg"; iconOnly: true }
            }

            ColumnLayout {
                visible: root.resolvedUris.length > 0
                spacing: Theme.spacingSm

                Repeater {
                    model: root.resolvedUris

                    ResolverUri {
                        width: parent.width
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

                    RButton { iconSource: "../../icons/MdiLightFolder.svg"; onClicked: root.pickFolder(); iconOnly: true }
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
            enabled: !Object.keys(root.errors).length && root.resolvedUris.length > 0 || !root.isFetching
            onClicked: root.submit()
        }
    }
}
