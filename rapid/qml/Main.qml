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

    Router {
        id: router
        anchors.fill: parent
        animated: window.compact

        routes: ({
            [Navigation.downloadPage]: downloadPage,
            [Navigation.settingsPage]: settingsPage
        })

        Component {
            id: downloadPage
            DownloadPage {}
        }
        Component {
            id: settingsPage
            SettingsPage {}
        }

        Component.onCompleted: navigate(Navigation.downloadPage)
    }
}
