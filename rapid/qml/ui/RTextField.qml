import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import ".."

Item {
    id: root

    property alias label: labelLabel.text
    property alias error: errorLabel.text
    property alias text: field.text
    property alias placeholderText: field.placeholderText
    property alias echoMode: field.echoMode
    property alias readOnly: field.readOnly
    property alias maximumLength: field.maximumLength
    property alias validator: field.validator
    property alias inputMethodHints: field.inputMethodHints
    property alias wrapMode: field.wrapMode
    property alias selectByMouse: field.selectByMouse
    property alias prefixIcon: field.prefixIcon
    property alias suffixIcon: field.suffixIcon
    property alias iconSize: field.iconSize
    property alias iconColor: field.iconColor
    property alias field: field
    property alias loading: field.loading

    default property alias fieldRowData: fieldRow.data

    implicitWidth: column.implicitWidth
    implicitHeight: column.implicitHeight

    ColumnLayout {
        id: column
        anchors.fill: parent
        spacing: Theme.spacingSm

        Controls.Label {
            id: labelLabel
            visible: text.length > 0
            color: Theme.colorText
            font.pixelSize: Theme.textSize
            Layout.alignment: Qt.AlignLeft
        }

        RowLayout {
            id: fieldRow
            Layout.fillWidth: true
            spacing: Theme.spacingMd

            Controls.TextField {
                id: field
                property bool loading: false
                property url prefixIcon: ""
                property url suffixIcon: loading ? "qrc:/icons/MdiLightLoading.svg" : ""
                property int iconSize: Theme.iconMd
                property color iconColor: Theme.colorTextMuted
                readonly property int cornerRadius: Qt.platform.os === "linux" ? Theme.radiusSm : Theme.radiusMd

                Layout.fillWidth: true
                selectByMouse: true
                color: Theme.colorText
                placeholderTextColor: Theme.colorTextMuted
                padding: Theme.spacingMd
                leftPadding: Theme.spacingMd + (prefixIconButton.visible ? field.iconSize + Theme.spacingXs : 0)
                rightPadding: Theme.spacingMd + (suffixIconButton.visible ? field.iconSize + Theme.spacingXs : 0)
                implicitHeight: Math.max(
                    Theme.touchTarget,
                    contentHeight + topPadding + bottomPadding
                )

                background: Rectangle {
                    radius: field.cornerRadius
                    color: Theme.colorInputBackground
                    border.width: 1
                    border.color: root.error.length > 0
                                 ? Theme.colorDanger
                                 : (field.activeFocus ? Theme.colorPrimary : Theme.colorBorder)
                }

                Controls.Button {
                    id: prefixIconButton
                    anchors.left: field.left
                    anchors.leftMargin: Theme.spacingMd
                    anchors.verticalCenter: field.verticalCenter
                    width: field.iconSize
                    height: field.iconSize
                    visible: field.prefixIcon.toString() !== ""
                    enabled: false
                    opacity: 1
                    padding: 0
                    display: Controls.AbstractButton.IconOnly
                    icon.source: field.prefixIcon
                    icon.width: field.iconSize
                    icon.height: field.iconSize
                    icon.color: field.iconColor
                    background: null
                }

                Controls.Button {
                    id: suffixIconButton
                    anchors.right: field.right
                    anchors.rightMargin: Theme.spacingMd
                    anchors.verticalCenter: field.verticalCenter
                    width: field.iconSize
                    height: field.iconSize
                    visible: field.suffixIcon.toString() !== ""
                    enabled: false
                    opacity: 1
                    padding: 0
                    display: Controls.AbstractButton.IconOnly
                    icon.source: field.suffixIcon
                    icon.width: field.iconSize
                    icon.height: field.iconSize
                    icon.color: field.iconColor
                    background: null

                    RotationAnimator {
                        target: suffixIconButton
                        running: field.loading
                        from: 0
                        to: 360
                        duration: 1000
                        loops: Animation.Infinite
                    }
                }
            }
        }

        Controls.Label {
            id: errorLabel
            visible: text.length > 0
            color: Theme.colorDanger
            font.pixelSize: Theme.textSizeSm
            Layout.fillWidth: true
            wrapMode: Text.Wrap
        }
    }
}
