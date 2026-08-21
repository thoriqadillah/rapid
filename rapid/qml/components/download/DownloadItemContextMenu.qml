import QtQuick
import QtQuick.Controls as Controls

// DownloadService is a Python-registered context property, invisible to
// qmllint's static analysis.
// qmllint disable unqualified
Controls.Menu {
    id: root

    property string gid: ""
    property string status: ""
    property string fileDir: ""
    property var resolved: null


    Controls.MenuItem {
        text: qsTr("Pause")
        enabled: root.status === "active" || root.status === "waiting"
        onTriggered: DownloadService.pause(root.gid)
    }
    Controls.MenuItem {
        text: qsTr("Resume")
        enabled: root.status === "paused"
        onTriggered: DownloadService.resume(root.gid)
    }
    Controls.MenuItem {
        text: qsTr("Stop")
        enabled: root.status === "active" || root.status === "waiting"
        onTriggered: DownloadService.stop(root.gid)
    }
    Controls.MenuItem {
        text: qsTr("Delete")
        enabled: root.status === "complete" || root.status === "error"
        onTriggered: DownloadService.delete(root.gid)
    }

    Controls.MenuSeparator {}

    Controls.MenuItem {
        text: qsTr("Copy URL")
        enabled: root.resolved && root.resolved.url
        onTriggered: Clipboard.copy(root.resolved.url)
    }
    Controls.MenuItem {
        text: qsTr("Go to file")
        enabled: root.fileDir.length > 0
        onTriggered: Qt.openUrlExternally("file://" + root.fileDir)
    }
}
