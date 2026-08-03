import QtQuick
import QtQuick.Controls
import "components"
import "pages"

ApplicationWindow {
    id: window

    width: 1280
    height: 720
    visible: true
    title: "Rapid downloader"
    color: Theme.colorBackground

    readonly property bool compact: width < Theme.breakpointMd
    readonly property bool medium: width >= Theme.breakpointMd && width < Theme.breakpointLg
    readonly property bool expanded: width >= Theme.breakpointLg

    Layout {
        anchors.fill: parent

        onDestinationSelected: destination => router.navigate(destination)

        Router {
            id: router
            anchors.fill: parent
            animated: window.compact

            routes: ({
                "all": allDownloadsPage,
                "scheduled": scheduledPage,
                "audio": audioPage,
                "documents": documentsPage,
                "images": imagesPage,
                "video": videoPage,
                "settings": settingsPage
            })

            Component {
                id: allDownloadsPage
                DownloadPage {}
            }
            Component {
                id: scheduledPage
                DownloadPage {
                    filters: "scheduled"
                }
            }
            Component {
                id: audioPage
                DownloadPage {
                    filters: "audio"
                }
            }
            Component {
                id: documentsPage
                DownloadPage {
                    filters: "document"
                }
            }
            Component {
                id: imagesPage
                DownloadPage {
                    filters: "image"
                }
            }
            Component {
                id: videoPage
                DownloadPage {
                    filters: "video"
                }
            }
            Component {
                id: settingsPage
                SettingsPage {}
            }

            Component.onCompleted: navigate("all")
        }
    }
}
