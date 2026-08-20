import QtQuick
import QtQuick.Layouts
import "../.."

// DownloadService is a Python-registered context property, invisible to
// qmllint's static analysis.
// qmllint disable unqualified
Rectangle {
    id: root

    required property string gid
    property string category: "unknown"
    property var samples: []

    color: Theme.colorSurface
    radius: Theme.radiusSm
    implicitHeight: 120
    Layout.fillWidth: true

    function refresh() {
        root.samples = DownloadService.speedHistory(root.gid)
    }

    onSamplesChanged: chart.requestPaint()

    Connections {
        target: DownloadService
        Component.onCompleted: root.refresh()
        function onDownloadChanged(gid) {
            if (gid === root.gid) root.refresh()
        }
    }

    Canvas {
        id: chart
        anchors.fill: parent
        onPaint: {
            const ctx = chart.getContext("2d")
            ctx.reset()
            const w = chart.width
            const h = chart.height
            const s = root.samples
            if (!s || s.length === 0) {
                ctx.strokeStyle = Theme.colorTextMuted
                ctx.lineWidth = 2 * Screen.devicePixelRatio
                ctx.beginPath()
                ctx.moveTo(0, h)
                ctx.lineTo(w, h)
                ctx.stroke()
                return
            }
            let max = 1
            for (let i = 0; i < s.length; i++) max = Math.max(max, s[i].speed)
            const stepX = s.length > 1 ? w / (s.length - 1) : 0
            ctx.beginPath()
            ctx.moveTo(0, h)
            for (let i = 0; i < s.length; i++) {
                const x = i * stepX
                const y = h - (h * s[i].speed / max)
                ctx.lineTo(x, y)
            }
            ctx.lineTo(w, h)
            ctx.closePath()
            const base = Theme.categoryColor(root.category)
            ctx.fillStyle = Qt.rgba(base.r, base.g, base.b, 0.15)
            ctx.fill()
            ctx.beginPath()
            for (let i = 0; i < s.length; i++) {
                const x = i * stepX
                const y = h - (h * s[i].speed / max)
                if (i === 0) ctx.moveTo(x, y)
                else ctx.lineTo(x, y)
            }
            ctx.strokeStyle = base
            ctx.lineWidth = 2 * Screen.devicePixelRatio
            ctx.lineJoin = "round"
            ctx.stroke()
        }
    }
}
