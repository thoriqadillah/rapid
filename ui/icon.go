package ui

import (
	"errors"
	"fmt"

	qt "github.com/mappu/miqt/qt6"
)

const iconDir = "assets/icons"

//go:generate task build:icon

// IconPath resolves an icon file under the canonical assets/icons directory.
// It returns an empty string when lookup fails.
func IconPath(name string) string {
	return fmt.Sprintf(":/icons/%s", name)
}

// TintedPixmap loads a monochrome SVG, tints it to color at size, and keeps
// transparent pixels transparent, mirroring QML's icon.color behavior.
func TintedPixmap(path string, color *qt.QColor, size int) *qt.QPixmap {
	if path == "" || color == nil || size <= 0 {
		return nil
	}

	reader := qt.NewQImageReader3(path)
	reader.SetAutoTransform(true)
	reader.SetScaledSize(qt.NewQSize2(size, size))
	image := reader.Read()
	if image == nil || image.IsNull() {
		return nil
	}
	out := qt.NewQPixmap2(size, size)
	if !out.ConvertFromImage(image) {
		return nil
	}
	p := qt.NewQPainter2(out.QPaintDevice)
	p.SetRenderHint2(qt.QPainter__Antialiasing, true)
	p.SetCompositionMode(qt.QPainter__CompositionMode_SourceIn)
	p.FillRect2(0, 0, size, size, qt.NewQBrush3(color))
	p.End()
	return out
}

// LoadIconChecked returns a tinted icon and a useful error for invalid input.
// LoadIcon remains the nil-safe convenience API used by widgets.
func LoadIconChecked(path string, color *qt.QColor, size int) (*qt.QIcon, error) {
	if path == "" {
		return nil, errors.New("icon path is empty")
	}
	if color == nil {
		return nil, errors.New("icon color is nil")
	}
	if size <= 0 {
		return nil, fmt.Errorf("icon size must be positive: %d", size)
	}

	pm := TintedPixmap(path, color, size)
	if pm == nil {
		return nil, fmt.Errorf("icon %q could not be rasterized", path)
	}

	return qt.NewQIcon2(pm), nil
}

// LoadIcon returns a QIcon of a tinted icon, sized exactly to size.
func LoadIcon(path string, color *qt.QColor, size int) *qt.QIcon {
	icon, err := LoadIconChecked(path, color, size)
	if err != nil {
		return qt.NewQIcon()
	}
	return icon
}
