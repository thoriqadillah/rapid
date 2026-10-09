package downloads

import (
	"fmt"

	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

// Column widths from DownloadPage.qml. The ETA column uses its own width (the
// QML copy-paste bug used sizeColumnWidth for it).
const (
	NameColumnMinWidth  = 100
	ProgressColumnWidth = 300
	SpeedColumnWidth    = 120
	SizeColumnWidth     = 120
	EtaColumnWidth      = 120
)

// NewTablePanel is the rounded, surface-bordered container that holds the
// table header and list (DownloadPage.qml's bordered Rectangle). It paints its
// border directly because plain-QWidget stylesheet borders don't render
// reliably on this MIQT backend.
func NewTablePanel(parent *qt.QWidget) *qt.QWidget {
	return newRoundedPanel(parent, nil, theme.ColorSurface, 2, theme.RadiusSm).QWidget
}

// TableHeader is the muted "Name / Progress / Speed / Size / Estimation" row
// that sits above the download list.
type TableHeader struct {
	*qt.QWidget
}

func NewTableHeader(parent *qt.QWidget) *TableHeader {
	h := &TableHeader{QWidget: qt.NewQWidget3(parent, 0)}
	h.SetAttribute(qt.WA_StyledBackground)
	h.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Fixed)
	h.SetStyleSheet(fmt.Sprintf("background-color: %s;", theme.CssColor(theme.ColorSurface)))

	row := qt.NewQHBoxLayout2()
	row.SetContentsMargins(theme.SpacingLg, theme.SpacingSm, theme.SpacingLg, theme.SpacingSm)
	row.SetSpacing(theme.SpacingMd)

	spacer := qt.NewQWidget3(h.QWidget, 0)
	spacer.SetFixedWidth(theme.IconXs)
	row.AddWidget(spacer)

	name := styledLabel("Name", theme.ColorTextMuted)
	name.SetAlignment(qt.AlignLeft | qt.AlignVCenter)
	name.SetMinimumWidth(NameColumnMinWidth)
	name.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Preferred)
	row.AddWidget2(name.QWidget, 1)

	row.AddWidget(fixedLabel("Progress", theme.ColorTextMuted, ProgressColumnWidth).QWidget)
	row.AddWidget(fixedLabel("Speed", theme.ColorTextMuted, SpeedColumnWidth).QWidget)
	row.AddWidget(fixedLabel("Size", theme.ColorTextMuted, SizeColumnWidth).QWidget)
	row.AddWidget(fixedLabel("Estimation", theme.ColorTextMuted, EtaColumnWidth).QWidget)

	h.SetLayout(row.QLayout)
	return h
}
