import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import "../.."
import "../../ui"

// Confirmation dialog for deleting a download: shows the display name and a
// "delete files from disk" switch, then emits deleteConfirmed(bool) with the
// switch state.
RDialog {
    id: root

    property string displayName: ""

    signal deleteConfirmed(bool deleteFromDisk)

    title: qsTr("Delete download")
    Shortcut {
        sequence: "Return"
        onActivated: {
            root.close();
            root.deleteConfirmed(deleteFromDiskSwitch.checked);
        }
    }
    Shortcut {
        sequence: "Enter"
        onActivated: {
            root.close();
            root.deleteConfirmed(deleteFromDiskSwitch.checked);
        }
    }

    Text {
        text: qsTr("Remove %1 forever?").arg(root.displayName || qsTr("this download"))
        color: Theme.colorText
        font.pixelSize: Theme.textSize
        wrapMode: Text.WordWrap
        Layout.fillWidth: true
    }

    footer: RowLayout {
        spacing: Theme.spacingMd

        RowLayout {
            spacing: Theme.spacingSm
            RSwitch {
                id: deleteFromDiskSwitch
                checked: true
            }
            Controls.Label {
                text: qsTr("From disk")
                color: Theme.colorText
                font.pixelSize: Theme.textSize
            }
        }
        Item {
            Layout.fillWidth: true
        }
        RButton {
            text: qsTr("Cancel")
            onClicked: root.close()
        }
        RButton {
            text: qsTr("Delete")
            variant: RButton.DangerVariant
            onClicked: {
                root.close();
                root.deleteConfirmed(deleteFromDiskSwitch.checked);
            }
        }
    }
}
