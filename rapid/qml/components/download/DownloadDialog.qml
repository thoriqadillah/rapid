pragma ComponentBehavior: Bound
import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import QtQuick.Window
import "../.."
import "../../ui"

// DownloadService and Clipboard are Python-registered context properties,
// invisible to qmllint's static analysis.
// qmllint disable unqualified
RDialog {
    id: root

    property string defaultDir: ""
    property var resolvedUris: []
    property var options: ({})
    property var requestContext: ({})
    property var errors: ({})
    property var previousResolvedUri: []
    property bool isFetching: false
    property bool isEditing: false

    minimumWidth: 480
    maxHeight: {
        const avail = Screen.availableHeight > 0 ? Screen.availableHeight : 700;
        return Math.max(Theme.touchTarget * 6, Math.round(avail * 0.9));
    }
    title: qsTr("New download")

    onOpened: root.resolve()
    onClosing: root.reset()

    Shortcut {
        sequence: "Return"
        onActivated: if (downloadButton.enabled) root.submit()
    }
    Shortcut {
        sequence: "Enter"
        onActivated: if (downloadButton.enabled) root.submit()
    }

    Controls.ScrollView {
        id: scroll
        Layout.fillWidth: true
        Layout.fillHeight: true
        clip: true
        Controls.ScrollBar.horizontal: Controls.ScrollBar { policy: Controls.ScrollBar.AlwaysOff }
        Controls.ScrollBar.vertical: Controls.ScrollBar {
            id: vbar
            policy: Controls.ScrollBar.AsNeeded
        }
        Component.onCompleted: contentItem.boundsBehavior = Flickable.StopAtBounds

        ColumnLayout {
            id: form
            width: scroll.availableWidth
            spacing: Theme.spacingLg

            MouseArea {
                Layout.fillWidth: true
                Layout.fillHeight: true
                z: -1
                onClicked: links.field.forceActiveFocus(false)
            }

            RTextField {
                id: links
                label: qsTr("URL")
                placeholderText: qsTr("https://example.com")
                selectByMouse: true
                wrapMode: TextEdit.NoWrap
                Layout.fillWidth: true

                onTextChanged: {
                    if (root.requestContext.url && root.requestContext.url !== links.text)
                        root.requestContext = ({});
                    resolveTimer.restart();
                    root.errors = ({});
                    links.error = "";

                    if (links.text === "") root.startTransition([]);
                    else root.startTransition([{ loading: true }]);
                }

                RButton {
                    iconSource: "qrc:/icons/MdiLightContentPaste.svg"
                    iconOnly: true
                    enabled: Clipboard.text.length > 0
                    onClicked: links.text = Clipboard.text
                    tooltip: qsTr("Paste from clipboard")
                }
            }

            ColumnLayout {
                id: uriBlock
                visible: root.resolvedUris.length > 0 || root.isFetching
                spacing: Theme.spacingSm
                Layout.minimumHeight: 75
                property bool exiting: false

                function leave() {
                    uriBlock.exiting = true;
                    exitTimer.restart();
                }

                Timer {
                    id: exitTimer
                    interval: 310
                    onTriggered: {
                        root.resolvedUris = root.previousResolvedUri;
                        root.previousResolvedUri = [];
                        uriBlock.exiting = false;
                    }
                }

                Repeater {
                    id: uriRepeater
                    model: root.resolvedUris

                    ResolverUri {
                        required property int index
                        closable: root.resolvedUris.length > 1
                        onDeleteClicked: root.removeUri(index)
                        onSaved: newUri => {
                            root.resolvedUris[index] = newUri;
                        }
                        onEditingChanged: () => {
                            root.isEditing = root.resolvedUris.some((_, i) => uriRepeater.itemAt(i)?.editing);
                        }

                        Connections {
                            target: uriBlock
                            function onExitingChanged() {
                                if (uriBlock.exiting) uriRepeater.itemAt(index)?.exit();
                                uriBlock.exiting = false;
                            }
                        }

                    }
                }
            }

            ColumnLayout {
                id: saveDirLayout
                implicitWidth: scroll.availableWidth
                spacing: Theme.spacingMd
                RTextField {
                    id: saveDir
                    label: qsTr("Destination")
                    text: root.defaultDir
                    Layout.fillWidth: true

                    RButton {
                        iconSource: "qrc:/icons/MdiLightFolder.svg"
                        iconOnly: true
                        tooltip: qsTr("Pick folder")
                        onClicked: root.pickFolder()
                    }
                }
            }

            AdvancedOptions {
                id: advanced
                implicitWidth: scroll.availableWidth
                Layout.fillWidth: true
            }
        }
    }

    Timer {
        id: resolveTimer
        interval: 300
        onTriggered: root.resolve()
    }

    function pickFolder() {
        const dir = DownloadService.pickFolder(saveDir.text);
        if (dir)
            saveDir.text = dir;
    }

    function headersDict() {
        const headers = {};
        for (const row of advanced.headers.values())
            headers[row.key] = row.value;
        return headers;
    }

    function cookiesDict() {
        const cookies = {};
        for (const row of advanced.cookies.values())
            cookies[row.key] = row.value;
        return cookies;
    }

    function submit() {
        const dir = saveDir.text;
        const headers = root.headersDict();
        const cookies = root.cookiesDict();
        root.resolvedUris = root.resolvedUris.map(uri => {
            const patch = {
                headers: Object.assign({}, uri.headers || {}, headers),
                cookies: Object.assign({}, uri.cookies || {}, cookies)
            };
            if (dir) patch.dir = dir;
            return Object.assign({}, uri, patch);
        });
        DownloadService.download(root.resolvedUris);
        root.close();
    }

    function dictToRows(dict) {
        const rows = [];
        for (const key of Object.keys(dict || {}))
            rows.push({ key: key, value: dict[key] });
        return rows;
    }

    function openFromBrowser(request, newOwner) {
        root.requestContext = request;
        links.text = request.url ?? "";
        root.openFor(newOwner);
        root.setAdvancedFromContext(request);
        resolveTimer.stop();
    }

    function setUrl(url) {
        links.text = url;
    }

    function setAdvancedFromContext(context) {
        const headers = context.headers;
        const cookies = context.cookies;
        advanced.headers.setRows(
            headers && typeof headers === "object" && Object.keys(headers).length > 0
                ? root.dictToRows(headers)
                : []
        );
        advanced.headers.ensureRow();
        advanced.cookies.setRows(
            cookies && typeof cookies === "object" && Object.keys(cookies).length > 0
                ? root.dictToRows(cookies)
                : []
        );
        advanced.cookies.ensureRow();
    }

    function reset() {
        links.text = "";
        saveDir.text = root.defaultDir;
        root.resolvedUris = [];
        root.errors = ({});
        root.requestContext = ({});
        root.previousResolvedUri = [];
        root.isFetching = false;
        advanced.headers.setRows([]);
        advanced.headers.ensureRow();
        advanced.cookies.setRows([]);
        advanced.cookies.ensureRow();
        resolveTimer.stop();
        exitTimer.stop();
    }

    function removeUri(id) {
        root.resolvedUris = root.resolvedUris.filter((_, idx) => idx !== id);
    }

    function resolve() {
        if (links.text === "")
            return;
        root.isFetching = true;
        const options = Object.assign({}, root.requestContext, {
            url: links.text,
            headers: root.headersDict(),
            cookies: root.cookiesDict()
        });
        DownloadService.resolveRequest(options);
    }

    function startTransition(newUris) {
        if (root.resolvedUris.length > 0) {
            root.previousResolvedUri = newUris;
            uriBlock.leave();
            return;
        }

        root.resolvedUris = newUris;
    }

    Connections {
        target: DownloadService
        function onResolved(uris, errors) {
            root.errors = errors;
            root.isFetching = false;
            links.error = errors.url ?? "";
            root.startTransition(uris);
            for (const uri of uris) {
                if (uri && uri.dir) {
                    saveDir.text = uri.dir;
                    break;
                }
            }
        }
    }

    footer: RowLayout {
        spacing: Theme.spacingMd

        RButton {
            text: qsTr("Cancel")
            onClicked: root.close()
            enabled: !root.isFetching
        }
        RButton {
            id: downloadButton
            text: qsTr("Download")
            variant: RButton.PrimaryVariant
            enabled: links.text !== "" && !Object.keys(root.errors).length && root.resolvedUris.length > 0 && !root.isFetching && !root.isEditing
            onClicked: root.submit()
        }
    }
}
