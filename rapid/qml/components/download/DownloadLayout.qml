pragma ComponentBehavior: Bound
import QtQuick
import QtQuick.Layouts as QL
import ".."
import "../.."

// DownloadService is a Python-registered context property, invisible to
// qmllint's static analysis.
// qmllint disable unqualified
Layout {
    id: root

    function countFor(key: string) : string {
        const value = DownloadService.counts[key] ?? 0;
        return value > 0 ? String(value) : "";
    }

    sidebarContent: Component {
        Sidebar {
            id: sidebar

            SidebarSection {
                heading: qsTr("LIBRARY")
                currentDestination: sidebar.currentDestination
                onActivated: destination => sidebar.activate(destination)
                items: [
                    {
                        destination: "",
                        label: qsTr("All downloads"),
                        count: root.countFor("all"),
                        iconSource: "qrc:/icons/MdiLightDownload.svg",
                        iconColor: Theme.colorText
                    }
                ]
            }

            SidebarSection {
                heading: qsTr("CATEGORIES")
                currentDestination: sidebar.currentDestination
                onActivated: destination => sidebar.activate(destination)
                topMargin: Theme.spacingMd
                items: [
                    {
                        destination: "audio",
                        label: qsTr("Audio"),
                        count: root.countFor("audio"),
                        iconSource: "qrc:/icons/MdiSquareRounded.svg",
                        iconColor: Theme.colorCategoryAudio,
                        categoryItem: true
                    },
                    {
                        destination: "application",
                        label: qsTr("Application"),
                        count: root.countFor("application"),
                        iconSource: "qrc:/icons/MdiSquareRounded.svg",
                        iconColor: Theme.colorCategoryApplication,
                        categoryItem: true
                    },
                    {
                        destination: "compressed",
                        label: qsTr("Compressed"),
                        count: root.countFor("compressed"),
                        iconSource: "qrc:/icons/MdiSquareRounded.svg",
                        iconColor: Theme.colorCategoryCompressed,
                        categoryItem: true
                    },
                    {
                        destination: "document",
                        label: qsTr("Document"),
                        count: root.countFor("document"),
                        iconSource: "qrc:/icons/MdiSquareRounded.svg",
                        iconColor: Theme.colorCategoryDocument,
                        categoryItem: true
                    },
                    {
                        destination: "image",
                        label: qsTr("Images"),
                        count: root.countFor("image"),
                        iconSource: "qrc:/icons/MdiSquareRounded.svg",
                        iconColor: Theme.colorCategoryImage,
                        categoryItem: true
                    },
                    {
                        destination: "video",
                        label: qsTr("Videos"),
                        count: root.countFor("video"),
                        iconSource: "qrc:/icons/MdiSquareRounded.svg",
                        iconColor: Theme.colorCategoryVideo,
                        categoryItem: true
                    }
                ]
            }

            Item {
                QL.Layout.fillHeight: true
                QL.Layout.minimumHeight: Theme.spacingMd
            }

            SidebarItem {
                destination: "settings"
                label: qsTr("Settings")
                iconSource: "qrc:/icons/MdiLightSettings.svg"
                iconColor: Theme.colorText
                selected: sidebar.currentDestination === destination
                onActivated: sidebar.activate(destination)
                QL.Layout.fillWidth: true
            }
        }
    }
}
