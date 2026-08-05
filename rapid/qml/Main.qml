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

    StackView {
        id: router
        anchors.fill: parent
        clip: true

        // TODO: animate on mobile
        replaceEnter: null
        replaceExit: null
        pushEnter: null
        pushExit: null

        property var routes: ({
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

        Component.onCompleted: {
            Navigation.router = router;
            Navigation.replace(Navigation.downloadPage);
        }
    }
}
