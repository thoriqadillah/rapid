# Reusable Widgets

MIQT Qt6 Widgets equivalents of the reusable QML components in
`rapid/qml/ui/` (`RButton`, `RSwitch`, `RTextField`, `RDialog`) plus the icon
helper in `ui/icon.go`.

## Icons

Icons are embedded through the Qt resource system. The manifest is
`assets/icons/icons.qrc`; the generated wrapper is checked in as:

```text
widget/ui/resources_generated.go
widget/ui/resources.rcc
```

Regenerate it with:

```bash
go generate ./widget/ui
```

The `//go:generate` directive on `widget/ui/icon.go` calls `task build:icon`,
which locates Qt's `rcc`, embeds the resource data, and removes the
machine-specific generator directive from the checked-in Go source. The task
can also be run directly:

```bash
task build:icon
```

`IconPath` returns the `:/icons/<name>` resource form. `LoadIconChecked`
returns a useful error for an empty path, a nil color, a non-positive size, or
an SVG that cannot be rasterized; `LoadIcon` is the nil-safe convenience
wrapper used by widgets.

Rendered icons are memoized by `(path, color, size, device pixel ratio)` in
`ui/icon.go`. `TintedPixmap` returns a **shared, read-only** pixmap owned by
that cache: callers must not delete or mutate it (Qt copies it on
`SetPixmap`/`NewQIcon2`).

## Memory model

MIQT does **not** attach a Go finalizer to `NewX(...)` constructor results, so
Qt temporaries created in paint, hover, or poll paths (`QPainter`, `QPen`,
`QBrush`, `QColor`, `QFontMetrics`, `QSize`, `QPoint`, `QVariant`,
`QEasingCurve`, `QPixmap`, `QIcon`) must be released explicitly:

- local and not escaping → `defer x.Delete()` (or `x.Delete()` after use);
- escaping (returned, cached, handed to a Qt slot that copies) → `x.GoGC()`
  once, or keep it alive in a cache/field.

Values returned **from methods** (for example `QColor.DarkerWithInt`,
`QImageReader.Read`, `QPixmap.Scaled`) are already finalized by MIQT and must
**not** be deleted — doing so double-frees and crashes.

See `tasks/review/task.md` for the memory-growth investigation and the full
list of affected call sites.
