pragma ComponentBehavior: Bound
import QtQuick
import QtQuick.Layouts
import ".."

ColumnLayout {
    id: section

    property string heading: ""
    property var items: []
    property string currentDestination: ""
    property real topMargin: 0
    signal activated(string destination)

    Layout.topMargin: section.topMargin
    spacing: Theme.spacingXs

    SidebarLabel {
        visible: section.heading !== ""
        text: section.heading
        Layout.bottomMargin: Theme.spacingXs
    }

    Repeater {
        model: section.items

        delegate: SidebarItem {
            id: delegateItem

            required property var modelData

            destination: modelData.destination
            label: modelData.label ?? ""
            count: modelData.count ?? ""
            iconSource: modelData.iconSource ?? ""
            iconColor: modelData.iconColor ?? Theme.colorTextMuted
            categoryItem: modelData.categoryItem ?? false
            selected: section.currentDestination === delegateItem.destination
            onActivated: section.activated(delegateItem.destination)
            Layout.fillWidth: true
        }
    }
}
