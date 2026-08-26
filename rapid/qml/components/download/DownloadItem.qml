import QtQuick
import QtQuick.Layouts
import QtQuick.Controls as Controls
import "../.."
import "../../ui"

// DownloadService is a Python-registered context property, invisible to
// qmllint's static analysis.
// qmllint disable unqualified
Rectangle {
    id: root

    // Role values from DownloadService (a QAbstractListModel).
    required property int index
    required property var gid
    required property var status
    required property var category
    required property var resolved
    required property var files
    required property var totalLength
    required property var completedLength
    required property var downloadSpeed
    required property var errorMessage

    property int nameColumnMinWidth: 100
    property int progressColumnWidth: 250
    property int speedColumnWidth: 125
    property int sizeColumnWidth: 125
    property int etaColumnWidth: 125

    property bool menuOpen: false
    property bool highlighted: false
    property bool expanded: false

    signal contextMenuRequested(var data)
    signal toggleExpanded
    signal removeRequested(var gid)

    color: highlighted || root.expanded || (rowHovered && !menuOpen) ? Theme.colorSurface : (index % 2 !== 0 ? Qt.rgba(Theme.colorSurface.r, Theme.colorSurface.g, Theme.colorSurface.b, 0.25) : "transparent")
    clip: true
    height: row.implicitHeight + Theme.spacingSm * 2 + root.detailHeight

    property real detailHeight: 0
    Behavior on detailHeight {
        NumberAnimation {
            duration: 150
            easing.type: Easing.OutCubic
        }
    }
    onExpandedChanged: root.detailHeight = root.expanded ? detail.implicitHeight : 0

    readonly property bool rowHovered: root.expanded ? rowMouse.containsMouse && rowMouse.mouseY <= row.implicitHeight : rowMouse.containsMouse

    readonly property string displayName: {
        const res = root.resolved;
        if (res && (res.title || res.filename))
            return res.title || res.filename;
        const paths = Array.isArray(root.files) ? root.files : [];
        const path = paths.length > 0 ? paths[0].path : "";
        if (path)
            return path.split("/").pop() || path;
        return root.gid;
    }

    readonly property string fileDir: {
        const paths = root.files;
        const path = paths.length > 0 ? paths[0].path : "";
        if (!path)
            return "";
        const idx = path.lastIndexOf("/");
        return idx > 0 ? path.slice(0, idx) : "";
    }

    readonly property double percent: {
        const total = Number(root.totalLength) || 0;
        if (!total)
            return 0;
        return Math.min(1, (Number(root.completedLength) || 0) / total);
    }

    readonly property bool isError: root.status === "error"
    readonly property bool isPaused: root.status === "paused"
    readonly property bool isComplete: root.status === "complete"

    readonly property string speedText: {
        if (root.isError)
            return "—";
        const speed = Number(root.downloadSpeed) || 0;
        return speed > 0 ? DownloadService.formatSize(speed) + "/s" : "—";
    }

    readonly property string sizeText: {
        const total = Number(root.totalLength) || 0;
        if (!total)
            return "—";
        const done = Number(root.completedLength) || 0;
        return DownloadService.formatSize(done) + " / " + DownloadService.formatSize(total);
    }

    MouseArea {
        id: rowMouse
        anchors.fill: parent
        hoverEnabled: true
        acceptedButtons: Qt.LeftButton | Qt.RightButton
        onClicked: function (mouse) {
            if (mouse.button === Qt.RightButton) {
                root.contextMenuRequested({
                    gid: root.gid,
                    status: root.status,
                    fileDir: root.fileDir,
                    resolved: root.resolved
                });
                return;
            }

            if (mouse.button === Qt.LeftButton) {
                root.toggleExpanded();
            }
        }
    }

    Rectangle {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        height: 1
        color: Theme.colorBorder
        opacity: 0.4
    }

    RowLayout {
        id: row
        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.topMargin: Theme.spacingSm
        anchors.bottomMargin: Theme.spacingSm
        anchors.leftMargin: Theme.spacingLg
        anchors.rightMargin: Theme.spacingLg
        spacing: Theme.spacingMd

        Controls.Button {
            id: icon

            enabled: false
            opacity: 1
            padding: 0
            display: Controls.AbstractButton.IconOnly
            icon.source: "qrc:/icons/MdiSquareRounded.svg"
            icon.width: Theme.iconXs
            icon.height: Theme.iconXs
            icon.color: Theme.categoryColor(root.category)
            background: null
            Layout.preferredWidth: Theme.iconXs
            Layout.preferredHeight: Theme.iconXs
            Layout.alignment: Qt.AlignVCenter
        }

        Text {
            text: root.displayName
            color: Theme.colorText
            font.pixelSize: Theme.textSize
            elide: Text.ElideRight
            verticalAlignment: Text.AlignVCenter
            Layout.fillWidth: true
            Layout.minimumWidth: root.nameColumnMinWidth
        }

        RowLayout {
            Layout.preferredWidth: root.progressColumnWidth
            Layout.minimumWidth: root.progressColumnWidth
            Layout.maximumWidth: root.progressColumnWidth
            spacing: Theme.spacingSm

            Rectangle {
                Layout.fillWidth: true
                implicitHeight: 10
                radius: height / 2
                border.width: 1
                border.color: Theme.colorBorder
                color: Theme.colorSurface

                Rectangle {
                    width: parent.width * root.percent
                    height: parent.height
                    radius: height / 2
                    color: root.isError ? Theme.colorDanger : Theme.categoryColor(root.category)
                }
            }

            Text {
                text: qsTr("%1%").arg(Math.round(root.percent * 100))
                color: Theme.colorTextMuted
                font.pixelSize: Theme.textSize
                verticalAlignment: Text.AlignVCenter
                Layout.preferredWidth: percentMetrics.advanceWidth("100%")
                Layout.minimumWidth: percentMetrics.advanceWidth("100%")
                Layout.maximumWidth: percentMetrics.advanceWidth("100%")
            }

            FontMetrics {
                id: percentMetrics
                font.pixelSize: Theme.textSize
            }
        }

        Text {
            text: root.speedText
            color: Theme.colorText
            font.pixelSize: Theme.textSize
            elide: Text.ElideRight
            verticalAlignment: Text.AlignVCenter
            Layout.preferredWidth: root.speedColumnWidth
            Layout.minimumWidth: root.speedColumnWidth
            Layout.maximumWidth: root.speedColumnWidth
        }

        Text {
            text: root.sizeText
            color: Theme.colorText
            font.pixelSize: Theme.textSize
            elide: Text.ElideRight
            verticalAlignment: Text.AlignVCenter
            Layout.preferredWidth: root.sizeColumnWidth
            Layout.minimumWidth: root.sizeColumnWidth
            Layout.maximumWidth: root.sizeColumnWidth
        }

        DownloadStatus {
            status: root.status
            totalLength: root.totalLength
            completedLength: root.completedLength
            downloadSpeed: root.downloadSpeed
            Layout.preferredWidth: root.etaColumnWidth
            Layout.minimumWidth: root.etaColumnWidth
            Layout.maximumWidth: root.etaColumnWidth
            Layout.fillHeight: true
        }
    }

    Item {
        anchors.top: row.bottom
        anchors.left: parent.left
        anchors.right: parent.right
        height: root.detailHeight
        clip: true

        DownloadItemDetail {
            id: detail
            anchors.top: parent.top
            anchors.left: parent.left
            anchors.right: parent.right
            gid: root.gid
            status: root.status
            resolved: root.resolved
            files: root.files
            totalLength: root.totalLength
            completedLength: root.completedLength
            errorMessage: root.errorMessage
            onRemoveRequested: function (gid) {
                root.removeRequested(gid);
            }
            opacity: root.expanded ? 1 : 0
            transform: Translate {
                id: detailSlide
                y: root.expanded ? 0 : -detail.implicitHeight
                Behavior on y {
                    NumberAnimation {
                        duration: 250
                        easing.type: Easing.OutCubic
                    }
                }
            }
            Behavior on opacity {
                NumberAnimation {
                    duration: 250
                    easing.type: Easing.OutCubic
                }
            }
        }
    }
}
