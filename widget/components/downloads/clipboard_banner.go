package downloads

import (
	"fmt"

	"rapid/widget/theme"
	"rapid/widget/ui"

	qt "github.com/mappu/miqt/qt6"
)

// ClipboardBanner is shown above the table when a clipboard URL is present.
// Live clipboard monitoring is wired later; Phase 1 drives it via SetURL.
type ClipboardBanner struct {
	*qt.QWidget
	url        string
	downloadCB []func(string)
	dismissCB  []func()
}

func NewClipboardBanner(parent *qt.QWidget) *ClipboardBanner {
	b := &ClipboardBanner{QWidget: qt.NewQWidget3(parent, 0)}
	b.SetFixedHeight(theme.TouchTarget)
	b.SetAttribute(qt.WA_StyledBackground)
	b.SetStyleSheet(fmt.Sprintf(`
		QWidget {
			background-color: %s;
			border-radius: %dpx;
		}`,
		theme.CssColor(theme.WithAlpha(theme.ColorInfo, 38)),
		theme.RadiusSm,
	))

	icon := qt.NewQLabel3("")
	icon.SetFixedSize2(theme.IconSm, theme.IconSm)
	icon.SetPixmap(ui.TintedPixmap(ui.IconPath("MdiLightContentPaste.svg"), theme.ColorInfo, theme.IconSm))

	message := styledLabel("There is one url in the clipboard", theme.ColorInfo)

	downloadBtn := ui.NewRButton("Download", ui.LinkVariant, false)
	downloadBtn.SetIconSource(ui.IconPath("MdiLightDownload.svg"))
	downloadBtn.SetIconSize(theme.IconSm)
	downloadBtn.OnClicked(func() {
		for _, fn := range b.downloadCB {
			if fn != nil {
				fn(b.url)
			}
		}
	})

	dismissBtn := ui.NewRButtonIcon("MaterialSymbolsLightCloseRounded.svg", ui.LinkVariant, false)
	dismissBtn.SetIconSize(theme.IconSm)
	dismissBtn.OnClicked(func() {
		for _, fn := range b.dismissCB {
			if fn != nil {
				fn()
			}
		}
	})

	row := qt.NewQHBoxLayout2()
	row.SetContentsMargins(theme.SpacingMd, 0, theme.SpacingMd, 0)
	row.SetSpacing(theme.SpacingSm)
	row.AddWidget(icon.QWidget)
	row.AddWidget2(message.QWidget, 1)
	row.AddWidget(downloadBtn.QWidget)
	row.AddWidget(dismissBtn.QWidget)
	b.SetLayout(row.QLayout)

	b.SetVisible(false)
	return b
}

// SetURL shows the banner for a non-empty URL and hides it otherwise.
func (b *ClipboardBanner) SetURL(url string) {
	if b == nil {
		return
	}
	b.url = url
	b.SetVisible(url != "")
}

func (b *ClipboardBanner) URL() string {
	return b.url
}

// OnDownload fires with the clipboard URL when the Download action is clicked.
func (b *ClipboardBanner) OnDownload(fn func(string)) {
	if fn != nil {
		b.downloadCB = append(b.downloadCB, fn)
	}
}

// OnDismiss fires when the dismiss icon is clicked.
func (b *ClipboardBanner) OnDismiss(fn func()) {
	if fn != nil {
		b.dismissCB = append(b.dismissCB, fn)
	}
}
