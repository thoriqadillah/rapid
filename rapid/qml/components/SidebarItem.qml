import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import ".."

Item {
    id: item

    signal activated()

    required property string destination
    required property string label
    property string count: ""
    property url iconSource: ""
    property color iconColor: Theme.colorTextMuted
    property bool categoryItem: false
    property bool selected: false

    implicitHeight: 36

    Rectangle {
        anchors.fill: parent
        radius: Theme.radiusSm
        color: item.selected || mouseArea.containsMouse ? Qt.lighter(Theme.colorBase, 1.4) : "transparent"
    }

    RowLayout {
        anchors {
            fill: parent
            leftMargin: Theme.spacingSm
            rightMargin: Theme.spacingSm
        }
        spacing: Theme.spacingSm

        Controls.Button {
            enabled: false
            opacity: 1
            padding: 0
            display: Controls.AbstractButton.IconOnly
            icon.source: item.iconSource
            icon.width: item.categoryItem ? Theme.iconXs : Theme.iconSm
            icon.height: item.categoryItem ? Theme.iconXs : Theme.iconSm
            icon.color: item.selected && !item.categoryItem ? Theme.colorText : item.iconColor
            background: null
            Layout.preferredWidth: Theme.iconSm
            Layout.preferredHeight: Theme.iconSm
            Layout.alignment: Qt.AlignVCenter
        }

        Text {
            text: item.label
            color: item.selected || mouseArea.containsMouse ? Theme.colorText : Theme.colorTextMuted
            font.pixelSize: Theme.textSize
            elide: Text.ElideRight
            Layout.fillWidth: true
            Layout.alignment: Qt.AlignVCenter
        }

        Text {
            visible: item.count !== ""
            text: item.count
            color: item.selected || mouseArea.containsMouse ? Theme.colorText : Theme.colorTextMuted
            font.pixelSize: Theme.textSizeSm
            Layout.alignment: Qt.AlignVCenter
        }
    }

    MouseArea {
        id: mouseArea

        anchors.fill: parent
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor
        onClicked: item.activated()
    }
}
