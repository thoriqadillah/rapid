pragma ComponentBehavior: Bound
import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import "../.."
import "../../ui"

// Editable list of key/value pairs (used for custom HTTP headers and
// cookies in the download dialog). Rows are two RTextField columns with a
// per-row remove button and an "Add" button to append a blank row.
//
// A switch at the far end of the title row swaps the input to a raw JSON
// text area so the whole map can be viewed/edited in one go.
ColumnLayout {
    id: root

    property alias title: titleLabel.text
    property bool useJson: false
    property string jsonError: ""

    function setRows(newRows) {
        listModel.clear();
        for (const r of newRows || []) {
            listModel.append({
                key: r.key ?? "",
                value: r.value ?? ""
            });
        }

        jsonError = "";
        if (root.useJson) root.syncToJson();
    }

    function values() {
        if (root.useJson) {
            if (root.jsonError !== "") return [];
            const parsed = parseJson();
            if (!parsed) return [];
            const result = [];
            for (const key of Object.keys(parsed)) {
                result.push({
                    key: key,
                    value: String(parsed[key])
                });
            }
            return result;
        }
        const result = [];
        for (let i = 0; i < listModel.count; i++) {
            const key = listModel.get(i).key.trim();
            const value = listModel.get(i).value.trim();
            if (key !== "") {
                result.push({
                    key: key,
                    value: value
                });
            }
        }
        return result;
    }

    function ensureRow() {
        if (listModel.count === 0) {
            listModel.append({
                key: "",
                value: ""
            });
        }
    }

    function addRow() {
        listModel.append({
            key: "",
            value: ""
        });
    }

    function clearRow(index) {
        listModel.setProperty(index, "key", "");
        listModel.setProperty(index, "value", "");
    }

    function removeRow(index) {
        listModel.remove(index, 1);
    }

    function handleDelete(index) {
        if (listModel.count <= 1) root.clearRow(index);
        else root.removeRow(index);
    }

    function parseJson() {
        try {
            const value = JSON.parse(jsonArea.text);
            if (value === null || typeof value !== "object" || Array.isArray(value))
                throw new Error("must be a JSON object");
            return value;
        } catch (e) {
            root.jsonError = e.message || "invalid JSON";
            return null;
        }
    }

    function syncToJson() {
        const obj = {};
        for (const r of root.values())
            obj[r.key] = r.value;
        jsonArea.text = JSON.stringify(obj, null, 2);
        root.jsonError = "";
    }

    function syncToRows() {
        const parsed = parseJson();
        if (!parsed) return;
        root.setRows(Object.keys(parsed).map(key => ({ key: key, value: String(parsed[key]) })));
        root.ensureRow();
    }

    ListModel {
        id: listModel
    }

    Component.onCompleted: root.ensureRow()

    RowLayout {
        Layout.fillWidth: true
        spacing: Theme.spacingMd

        Controls.Label {
            id: titleLabel
            Layout.fillWidth: true
            Layout.alignment: Qt.AlignLeft
            Layout.bottomMargin: Theme.spacingSm
            color: Theme.colorTextMuted
            font.pixelSize: Theme.textSize
        }

        RowLayout {
            Layout.alignment: Qt.AlignRight
            spacing: Theme.spacingSm

            Controls.Label {
                text: qsTr("JSON")
                color: Theme.colorTextMuted
                font.pixelSize: Theme.textSizeSm
            }

            RSwitch {
                checked: root.useJson
                onToggled: {
                    if (checked) root.syncToJson();
                    else root.syncToRows();
                    root.useJson = checked;
                }
            }
        }
    }

    ColumnLayout {
        id: kvMode
        Layout.fillWidth: true
        visible: !root.useJson
        spacing: Theme.spacingSm

        Repeater {
            model: listModel

            RowLayout {
                id: rowLayout
                required property int index
                required property string key
                required property string value
                Layout.fillWidth: true
                spacing: Theme.spacingMd

                RTextField {
                    placeholderText: qsTr("Key")
                    text: rowLayout.key
                    selectByMouse: true
                    wrapMode: TextEdit.NoWrap
                    Layout.fillWidth: true
                    Layout.preferredWidth: 1
                    Layout.minimumWidth: 1
                    onTextChanged: listModel.setProperty(rowLayout.index, "key", text)
                }

                RTextField {
                    placeholderText: qsTr("Value")
                    text: rowLayout.value
                    selectByMouse: true
                    wrapMode: TextEdit.NoWrap
                    Layout.fillWidth: true
                    Layout.preferredWidth: 1
                    Layout.minimumWidth: 1
                    onTextChanged: listModel.setProperty(rowLayout.index, "value", text)
                }

                RButton {
                    iconSource: "qrc:/icons/MaterialSymbolsLightCloseRounded.svg"
                    iconOnly: true
                    variant: RButton.GhostVariant
                    Layout.alignment: Qt.AlignTop
                    onClicked: root.handleDelete(rowLayout.index)
                }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingMd

            Rectangle {
                Layout.fillWidth: true
                Layout.preferredHeight: 1
                color: Theme.colorBorder
                opacity: 0.4
            }

            RButton {
                iconSource: "qrc:/icons/MdiLightPlus.svg"
                iconOnly: true
                variant: RButton.GhostVariant
                onClicked: root.addRow()
            }
        }
    }

    Item {
        id: jsonMode
        Layout.fillWidth: true
        Layout.preferredHeight: 180
        visible: root.useJson

        Controls.ScrollView {
            anchors.fill: parent
            clip: true
            Component.onCompleted: contentItem.boundsBehavior = Flickable.StopAtBounds
            Controls.ScrollBar.horizontal: Controls.ScrollBar {
                policy: Controls.ScrollBar.AlwaysOff
            }
            Controls.ScrollBar.vertical: Controls.ScrollBar {
                policy: Controls.ScrollBar.AsNeeded
            }

            Controls.TextArea {
                id: jsonArea
                selectByMouse: true
                wrapMode: TextEdit.WrapAnywhere
                placeholderText: qsTr("{}")
                font.family: "monospace"
                font.pixelSize: Theme.textSizeSm
                color: Theme.colorText
                placeholderTextColor: Theme.colorTextMuted
                padding: Theme.spacingMd
                background: Rectangle {
                    radius: Theme.radiusSm
                    color: Theme.colorInputBackground
                    border.width: 1
                    border.color: root.jsonError.length > 0 ? Theme.colorDanger : (jsonArea.activeFocus ? Theme.colorPrimary : Theme.colorBorder)
                }
                onTextChanged: root.jsonError = ""
            }
        }
    }

    Controls.Label {
        Layout.fillWidth: true
        visible: root.useJson && root.jsonError.length > 0
        color: Theme.colorDanger
        font.pixelSize: Theme.textSizeSm
        wrapMode: Text.Wrap
    }
}
