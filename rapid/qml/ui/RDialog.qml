import QtQuick
import QtQuick.Layouts
import ".."

// Generic window-style dialog (a real Window, not a popup) with an overlay on
// its owner window. Both the body and the action area are slots:
//   - default property `content` : arbitrary body UI (Vue <slot>)
//   - `footer`                    : arbitrary action buttons (any, not just OK/Cancel)

// qmllint disable unqualified
Window {
    id: root

    default property alias content: body.data
    property alias footer: footerArea.data

    property Window owner: null
    property int maxHeight: 0

    signal opened

    readonly property real intrinsicHeight: body.implicitHeight + footerArea.implicitHeight + Theme.spacingLg * 3

    minimumWidth: 500
    height: root.maxHeight > 0 ? Math.min(root.maxHeight, root.intrinsicHeight) : root.intrinsicHeight
    visible: false
    color: Theme.colorBackground

    Shortcut {
        sequence: "Escape"
        onActivated: root.close()
    }

    // qmllint disable missing-property
    function setOverlay(show) {
        const overlay = root.owner ? root.owner.dialogOverlay : null;
        if (overlay)
            overlay.visible = show;
    }
    // qmllint enable missing-property

    function openFor(newOwner) {
        root.owner = newOwner ?? null;
        root.setOverlay(true);
        root.showNormal();
        root.raise();
        root.requestActivate();
        root.opened();
    }

    onVisibleChanged: if (!root.visible) root.setOverlay(false)
    Component.onDestruction: root.setOverlay(false)

    Connections {
        target: root.owner
        enabled: root.owner !== null
        function onClosing() {
            root.close();
        }
    }

    ColumnLayout {
        id: body
        anchors {
            top: parent.top
            left: parent.left
            right: parent.right
            bottom: root.maxHeight > 0 ? footerArea.top : undefined
            margins: Theme.spacingSm
            bottomMargin: Theme.spacingLg
        }
    }

    RowLayout {
        id: footerArea
        spacing: Theme.spacingMd
        layoutDirection: Qt.RightToLeft
        anchors {
            left: parent.left
            right: parent.right
            bottom: parent.bottom
            margins: Theme.spacingMd
        }
    }
}
