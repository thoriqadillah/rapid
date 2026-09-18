package components

import (
	"fmt"

	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

type SidebarLabel struct {
	*qt.QLabel
}

func NewSidebarLabel(text string) *SidebarLabel {
	l := &SidebarLabel{QLabel: qt.NewQLabel3(text)}
	l.SetAlignment(qt.AlignLeft | qt.AlignVCenter)
	font := l.Font()
	font.SetPointSize(theme.TextSizeSm)
	font.SetLetterSpacing(qt.QFont__AbsoluteSpacing, 1)
	l.SetFont(font)
	l.SetStyleSheet(fmt.Sprintf(
		"color: %s; background: transparent; border: none; font-size: %dpx;",
		theme.CssColor(theme.ColorTextMuted), theme.TextSizeSm,
	))
	l.SetMinimumHeight(theme.TextSizeSm + theme.SpacingXs)
	l.SetSizePolicy2(qt.QSizePolicy__Preferred, qt.QSizePolicy__Fixed)
	return l
}

func (l *SidebarLabel) SetText(text string) {
	l.QLabel.SetText(text)
}

func (l *SidebarLabel) Text() string {
	return l.QLabel.Text()
}
