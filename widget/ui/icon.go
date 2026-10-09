package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	qt "github.com/mappu/miqt/qt6"
)

const iconDir = "assets/icons"

//go:generate task build:icon

// IconPath returns the embedded Qt resource form for a bare icon filename.
func IconPath(name string) string {
	name = strings.TrimSpace(name)
	if strings.HasPrefix(name, ":/") || strings.HasPrefix(name, "qrc:/") {
		return name
	}
	return ":/icons/" + filepath.Base(name)
}

// iconKey identifies one rendered icon variant. Color is folded in as its
// packed RGBA value so equal colors share a cache entry.
type iconKey struct {
	path string
	rgba uint
	size int
	dpr  uint16 // device pixel ratio * 100
}

// iconCache memoizes rasterized+tinted pixmaps. Icon rendering is the single
// most expensive operation in the UI (image decode + SVG rasterize + paint),
// and the download list re-runs it for every row on every poll tick. Caching
// both removes the per-tick cost and makes the native allocations bounded
// instead of growing without limit.
//
// MIQT does NOT attach a Go finalizer to value types returned by NewX
// constructors (only to pass-by-value results like QColor.DarkerWithInt), so
// every temporary below must be released explicitly with Delete()/GoGC().
var (
	iconCacheMu sync.RWMutex
	iconCache   = map[iconKey]*qt.QPixmap{}
)

// TintedPixmap loads a monochrome SVG, tints it to color at size, and keeps
// transparent pixels transparent, mirroring QML's icon.color behavior.
//
// The returned pixmap is shared: it is owned by the internal cache, not by the
// caller. Callers must treat it as immutable and must not Delete() it (that
// would corrupt every other user of the same icon). Qt copies the pixmap on
// QLabel.SetPixmap / QIcon construction, so sharing is safe.
func TintedPixmap(path string, color *qt.QColor, size int) *qt.QPixmap {
	if path == "" || color == nil || size <= 0 {
		return nil
	}
	path = normalizeResourcePath(path)

	dpr := pixmapDevicePixelRatio()
	key := iconKey{
		path: path,
		rgba: color.Rgba(),
		size: size,
		dpr:  uint16(dpr * 100),
	}

	iconCacheMu.RLock()
	if pm, ok := iconCache[key]; ok {
		iconCacheMu.RUnlock()
		return pm
	}
	iconCacheMu.RUnlock()

	pm := renderTintedPixmap(path, color, size, dpr)
	if pm == nil {
		return nil
	}

	iconCacheMu.Lock()
	if existing, ok := iconCache[key]; ok {
		iconCacheMu.Unlock()
		pm.Delete()
		return existing
	}
	iconCache[key] = pm
	iconCacheMu.Unlock()
	return pm
}

// renderTintedPixmap rasterizes and tints one icon. Every Qt temporary is
// released before returning; the returned pixmap is the sole allocation the
// caller (the cache) owns.
func renderTintedPixmap(path string, color *qt.QColor, size int, dpr float64) *qt.QPixmap {
	// rasterize at device pixels so HiDPI icons stay `size` logical
	// pixels instead of rendering at half size vs the QML vector icons.
	scaled := max(int(float64(size)*dpr+0.5), size)

	reader := qt.NewQImageReader3(path)
	defer reader.Delete()
	reader.SetAutoTransform(true)
	scaledSize := qt.NewQSize2(scaled, scaled)
	defer scaledSize.Delete()
	reader.SetScaledSize(scaledSize)

	// NOTE: reader.Read() returns a QImage that MIQT already attaches a
	// finalizer to (pass-by-value semantics), so it must NOT be Delete()d here —
	// doing so double-frees and segfaults. Only constructor results (NewX) lack a
	// finalizer and need an explicit Delete().
	image := reader.Read()
	if image == nil || image.IsNull() {
		return nil
	}

	out := qt.NewQPixmap2(scaled, scaled)
	if !out.ConvertFromImage(image) {
		out.Delete()
		return nil
	}
	if dpr > 1 {
		out.SetDevicePixelRatio(dpr)
	}

	painter := qt.NewQPainter2(out.QPaintDevice)
	painter.SetRenderHint2(qt.QPainter__Antialiasing, true)
	painter.SetCompositionMode(qt.QPainter__CompositionMode_SourceIn)
	brush := qt.NewQBrush3(color)
	painter.FillRect2(0, 0, size, size, brush)
	// QPainter's destructor ends the paint device session, so Delete alone is
	// enough (an explicit End() before Delete is also fine but redundant).
	painter.Delete()
	brush.Delete()

	return out
}

// pixmapDevicePixelRatio returns the primary screen DPR, or 1 when there is
// no screen (tests) or the platform reports none.
func pixmapDevicePixelRatio() float64 {
	screen := qt.QGuiApplication_PrimaryScreen()
	if screen == nil {
		return 1
	}
	if dpr := screen.DevicePixelRatio(); dpr > 1 {
		return dpr
	}
	return 1
}

func normalizeResourcePath(path string) string {
	if strings.HasPrefix(path, "qrc:/") {
		path = ":" + strings.TrimPrefix(path, "qrc:")
	}
	if strings.HasPrefix(path, ":/") && !strings.HasPrefix(path, ":/icons/") {
		path = ":/icons/" + strings.TrimPrefix(path, ":/")
	}
	return path
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

	icon := qt.NewQIcon2(pm)
	// The caller only ever forwards this to SetIcon (which copies), so let the
	// GC release the wrapper instead of leaking one QIcon per icon refresh.
	icon.GoGC()
	return icon, nil
}

// LoadIcon returns a QIcon of a tinted icon, sized exactly to size.
func LoadIcon(path string, color *qt.QColor, size int) *qt.QIcon {
	icon, err := LoadIconChecked(path, color, size)
	if err != nil {
		empty := qt.NewQIcon()
		empty.GoGC()
		return empty
	}
	return icon
}
