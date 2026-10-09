package downloads

import (
	"fmt"
	"rapid/lib/helpers"
	"rapid/lib/helpers/bools"
	"rapid/services/download/api"
	"rapid/widget/theme"
	"rapid/widget/ui"
	"strings"

	qt "github.com/mappu/miqt/qt6"
)

// resolvedUriPanel is the readonly ResolverUri card: category icon, title, url and
// size/category meta.
type resolvedUriPanel struct {
	*roundedPanel
	icon  *qt.QLabel
	title *elidedLabel
	url   *elidedLabel
	meta  *qt.QLabel
}

func newResolverPanel(parent *qt.QWidget) *resolvedUriPanel {
	p := &resolvedUriPanel{roundedPanel: newRoundedPanel(parent, theme.ColorSurface, nil, 1, theme.RadiusSm)}

	p.icon = qt.NewQLabel3("")
	p.icon.SetFixedSize2(theme.IconXl, theme.IconXl)
	p.icon.SetAlignment(qt.AlignCenter)

	p.title = newElidedLabel("", theme.ColorText, qt.ElideMiddle)
	p.url = newElidedLabel("", theme.ColorTextMuted, qt.ElideMiddle)
	p.url.SetStyleSheet(fmt.Sprintf(`
		QLabel {
			background: transparent;
			border: none;
			color: %s;
			font-size: %dpx;
		}`,
		theme.CssColor(theme.ColorTextMuted),
		theme.TextSizeSm,
	))
	p.meta = smallLabel("", theme.ColorTextMuted)

	text := qt.NewQVBoxLayout2()
	text.SetContentsMargins(0, 0, 0, 0)
	text.SetSpacing(theme.SpacingXs)
	text.AddWidget(p.title.QWidget)
	text.AddWidget(p.url.QWidget)
	text.AddWidget(p.meta.QWidget)

	row := qt.NewQHBoxLayout2()
	row.SetContentsMargins(theme.SpacingMd, theme.SpacingMd, theme.SpacingMd, theme.SpacingMd)
	row.SetSpacing(theme.SpacingMd)
	row.AddWidget(p.icon.QWidget)
	row.AddLayout2(text.QLayout, 1)
	p.SetLayout(row.QLayout)
	return p
}

func (p *resolvedUriPanel) SetItem(item api.Download) {
	category := item.Category
	color := theme.CategoryColor(category)
	p.SetBorderColor(color)
	p.icon.SetPixmap(ui.TintedPixmap(
		ui.IconPath(categoryIcon(category)), color, theme.IconLg))
	p.icon.SetStyleSheet(fmt.Sprintf(
		"background-color: %s; border-radius: %dpx;",
		theme.CssColor(theme.WithAlpha(color, 38)), theme.RadiusSm))

	title := ""
	size := int64(0)
	url := ""
	if item.Resolved != nil {
		title = item.Resolved.Title
		if title == "" {
			title = item.Resolved.Filename
		}
		size = item.Resolved.Size
		url = item.Resolved.URL
	}

	title = bools.Ternary(title != "", title, item.Name())
	p.title.SetFullText(title)
	p.url.SetFullText(url)

	parts := make([]string, 0, 2)
	if formatted := helpers.FormatSize(size); formatted != "—" {
		parts = append(parts, formatted)
	}
	if label := category.Label(); label != "" {
		parts = append(parts, label)
	}
	p.meta.SetText(strings.Join(parts, " · "))
}
