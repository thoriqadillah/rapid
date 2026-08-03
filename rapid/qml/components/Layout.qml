pragma ComponentBehavior: Bound
import QtQuick

Item {
    id: root

    // --- Slots ---
    signal destinationSelected(string destination)
    property Component headerContent: null
    property Component sidebarContent: null

    property Component defaultHeaderComponent: Component {
        Header {}
    }

    property Component defaultSidebarComponent: Component {
        Sidebar {
            onDestinationSelected: destination => root.destinationSelected(destination)
        }
    }


    property bool sidebarOpen: true
    default property alias content: contentArea.data
    readonly property alias header: headerLoader.item
    readonly property alias sidebar: sidebarLoader.item

    // ---- Sidebar slot ----
    Loader {
        id: sidebarLoader
        sourceComponent: root.sidebarContent ?? root.defaultSidebarComponent
        onLoaded: item.open = Qt.binding(() => root.sidebarOpen)
        anchors {
            top: parent.top
            bottom: parent.bottom
            left: parent.left
        }

    }

    // ---- Header slot ----
    Loader {
        id: headerLoader
        sourceComponent: root.headerContent ?? root.defaultHeaderComponent
        anchors {
            top: parent.top
            left: sidebarLoader.right
            right: parent.right
        }


        // Forward menuClicked from whichever header is loaded
        Connections {
            target: headerLoader.item
            ignoreUnknownSignals: true
            function onMenuClicked() {
                root.sidebarOpen = !root.sidebarOpen
            }
        }
    }

    // ---- Default content slot ----
    Item {
        id: contentArea

        anchors {
            top: headerLoader.bottom
            left: sidebarLoader.right
            right: parent.right
            bottom: parent.bottom
        }
    }
}
