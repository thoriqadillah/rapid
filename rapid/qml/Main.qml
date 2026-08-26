import QtQuick
import QtQuick.Controls
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

    property alias dialogOverlay: overlay

    StackView {
        id: router
        anchors.fill: parent
        clip: true

        // TODO: animate on mobile
        replaceEnter: null
        replaceExit: null
        pushEnter: null
        pushExit: null

        Component {
            id: downloadPage
            DownloadPage {}
        }
        Component {
            id: settingsPage
            SettingsPage {}
        }

        Component.onCompleted: {
            const routes = ({
                [Navigation.downloadPage]: downloadPage,
                [Navigation.settingsPage]: settingsPage
            })

            Navigation.create(router, routes)
            Navigation.replace(Navigation.downloadPage)
        }
    }

    Rectangle {
        id: overlay
        anchors.fill: parent
        z: 999
        visible: false
        color: Qt.rgba(0, 0, 0, 0.4)
        focus: true

        // Block every interaction while visible: clicks, hover, wheel, drags.
        MouseArea {
            id: overlayBlocker
            anchors.fill: parent
            hoverEnabled: true
            acceptedButtons: Qt.AllButtons
            preventStealing: true
            onPressed: function (event) { event.accepted = true }
            onReleased: function (event) { event.accepted = true }
            onClicked: function (event) { event.accepted = true }
            onDoubleClicked: function (event) { event.accepted = true }
            onWheel: function (event) { event.accepted = true }
        }

        // Swallow all keys so focus can't reach the dimmed controls.
        Keys.onPressed: function (event) { event.accepted = true }
        Keys.onReleased: function (event) { event.accepted = true }
    }

    Connections {
        target: Notifier
        property int margin: 70
        property int gap: 70
        property var activePopups: []

        function reposition() {
            const base = window.height - margin
            for (let i = 0; i < activePopups.length; i++)
                activePopups[i].y = base - i * gap
        }

        function onNotificationRequested(type, title, message) {
            const colors = ({
                error: Theme.colorDanger,
                success: Theme.colorSuccess,
                info: Theme.colorInfo
            })

            const popup = notificationComponent.createObject(window, {
                title: title,
                message: message,
                typeColor: colors[type] || Theme.colorInfo,
                y: window.height - margin - activePopups.length * gap
            })

            activePopups.push(popup)
            popup.dismissed.connect(function () {
                activePopups = activePopups.filter(function (p) { return p !== popup })
                popup.destroy()
                reposition()
            })

            popup.open()
        }
    }

    Component {
        id: notificationComponent
        NotificationItem {}
    }
}
