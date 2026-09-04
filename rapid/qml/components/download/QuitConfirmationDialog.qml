import QtQuick
import QtQuick.Layouts
import "../.."
import "../../ui"

// Shown before quitting when downloads are still running: lists the active
// downloads and only quits once the user confirms.
RDialog {
    id: root

    property var activeNames: []

    readonly property string activeNamesHtml: {
        let items = ""
        for (const name of root.activeNames)
            items += "<li style='margin-left:0'>" + name + "</li>"
        return "<ul style='margin-left:-16px;margin-top:0;margin-bottom:0;padding-left:24px'>" + items + "</ul>"
    }

    signal quitConfirmed

    title: qsTr("Quit Rapid?")
    Shortcut {
        sequence: "Return"
        onActivated: {
            root.close();
            root.quitConfirmed();
        }
    }
    Shortcut {
        sequence: "Enter"
        onActivated: {
            root.close();
            root.quitConfirmed();
        }
    }

    ColumnLayout {
        spacing: Theme.spacingMd

        Text {
            text: qsTr("Active download is still downloading. Quitting now will stop the download. The download will be resumed automatically if you open Rapid again, but its possible that at that point the download link already expired")
            color: Theme.colorText
            font.pixelSize: Theme.textSize
            wrapMode: Text.WordWrap
            Layout.fillWidth: true
        }

        Text {
            text: qsTr("The following $1 still downloading:").replace("$1", root.activeNames.length > 1 ? "items are" : "item is")
            color: Theme.colorText
            font.pixelSize: Theme.textSize
            wrapMode: Text.WordWrap
            Layout.fillWidth: true
        }

        Text {
            text: root.activeNamesHtml
            textFormat: Text.RichText
            color: Theme.colorText
            font.pixelSize: Theme.textSize
            wrapMode: Text.WordWrap
            Layout.fillWidth: true
        }
    }

    footer: RowLayout {
        spacing: Theme.spacingMd

        Item {
            Layout.fillWidth: true
        }
        RButton {
            text: qsTr("Cancel")
            onClicked: root.close()
        }
        RButton {
            text: qsTr("Quit anyway")
            variant: RButton.DangerVariant
            onClicked: {
                root.close();
                root.quitConfirmed();
            }
        }
    }
}
