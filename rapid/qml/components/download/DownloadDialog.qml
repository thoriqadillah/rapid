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
    }

    function pickFolder() {
        const dir = Dialogs.pickFolder(saveDir.text)
        if (dir) saveDir.text = dir
    }

    function pickTorrent() {
        const file = Dialogs.pick_torrent()
    }

    function buildOptions() {
        const o = {}
        if (saveDir.text) o.dir = saveDir.text
        return o
    }

    function submit() {
        const uris = links.text.split(/\s+/).filter(s => s.length > 0)
        Aria2.add_uris(uris, buildOptions())
        root.close()
    }

    function reset() {
        links.text = ""
        saveDir.text = root.defaultDir
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

            ColumnLayout {
                Layout.fillWidth: true
                Layout.maximumWidth: parent.width
                spacing: Theme.spacingSm

                Controls.Label { text: qsTr("URL"); color: Theme.colorText; font.pixelSize: Theme.textSize }
                RowLayout {
                    Layout.fillWidth: true
                    Layout.maximumWidth: parent.width
                    spacing: Theme.spacingMd

                    RTextField {
                        id: links
                        placeholderText: qsTr("https://example.com")
                        selectByMouse: true
                        wrapMode: TextEdit.Wrap
                        Layout.fillWidth: true
                    }
                    RButton { iconSource: "../../icons/MdiLightContentPaste.svg" }
                }
            }

            ColumnLayout {
                Layout.fillWidth: true
                Layout.maximumWidth: parent.width
                spacing: Theme.spacingMd

                Controls.Label { text: qsTr("Destination"); color: Theme.colorText; font.pixelSize: Theme.textSize }
                RowLayout {
                    Layout.fillWidth: true
                    Layout.maximumWidth: parent.width
                    spacing: Theme.spacingMd

                    RTextField {
                        id: saveDir
                        text: root.defaultDir
                        Layout.fillWidth: true
                    }
                    RButton { iconSource: "../../icons/MdiLightFolder.svg"; onClicked: root.pickFolder() }
                }

                RTextField {
                    id: fileName
                    Layout.fillWidth: true
                    prefixIcon: "../icons/MdiLightFile.svg"
                    placeholderText: qsTr("File name")
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
        }
        RButton {
            text: qsTr("Download")
            variant: RButton.PrimaryVariant
            onClicked: root.submit()
        }
    }
}
