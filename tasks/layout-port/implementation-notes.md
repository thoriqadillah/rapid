# Layout/components port implementation notes

## Scope and baseline

Ported only the six eligible top-level QML components:

```text
rapid/qml/components/Header.qml
rapid/qml/components/Layout.qml
rapid/qml/components/Sidebar.qml
rapid/qml/components/SidebarItem.qml
rapid/qml/components/SidebarLabel.qml
rapid/qml/components/SidebarSection.qml
```

The following remain excluded and untouched:

```text
rapid/qml/components/download/
rapid/qml/components/settings/
```

Pages, `Main.qml`, routing, notification popups, and backend services were not
ported. The existing button, text-field, dialog, theme, and icon systems were
reused.

Baseline/final focused commands:

```text
env -u QT_QPA_PLATFORM go test -count=1 ./...
go vet ./...
git diff --check
```

No application build or launch was performed.

## Files added or changed

Added:

```text
ui/header.go
ui/layout.go
ui/sidebar.go
ui/sidebar_item.go
ui/sidebar_label.go
ui/sidebar_section.go
ui/layout_components_test.go
```

Changed for composite support:

```text
main.go
ui/icon.go
ui/rtextfield.go
```

`main.go` now places the current component demo inside `ui.Layout`, with a
real Header and Sidebar around it. The sidebar is grouped into a `Download`
section containing four distinct Download rows, an expanding spacer, and a
`Setting` section containing the Settings row. The page trees themselves remain
placeholders because they are outside this task.

## Component mapping

### Header

`NewHeader` composes existing `RButton` and `RTextField` widgets in a
`QHBoxLayout`:

```text
[menu] [stretch] [search <= 350px] [New]
```

It exposes ordered, nil-safe `OnMenuClicked` and `OnAddClicked` callbacks,
search text accessors, and a preferred search width setter. The menu, search,
and New controls use embedded `:/icons/...` resource paths.

### SidebarLabel

`NewSidebarLabel` is a styled QLabel with muted text, small font, one-pixel
absolute letter spacing, transparent background, and no interactive behavior.

### SidebarItem

`NewSidebarItem` is a Go-owned QWidget with a horizontal layout containing an
icon QLabel, a non-wrapping label, and an optional count label. Child labels
are transparent for mouse events so the parent row receives activation. Hover
and selected state update the row stylesheet and icon tint. Normal icons are
18px, category icons 10px, and the row minimum height is 36px.

### SidebarSection

`SidebarItemData` is a typed Go model replacing QML JavaScript object maps.
`SetItems` replaces section-owned item widgets without duplicating layout
entries. Selection is exact by destination and activation forwards the
selected destination.

### Sidebar

`Sidebar` owns a vertical content layout, fixed 200px open width, zero target
width when closed, surface background, and right border. `AddSection` wires
section activation into `Activate`, which updates current destination before
callbacks run. `AddStretch` inserts flexible vertical space between section
groups, matching the QML `ColumnLayout` stretch behavior used to keep Settings
at the bottom of the navigation rail.

### Layout

`Layout` uses a QGridLayout:

```text
row 0, column 0: sidebar, spans rows 0..1
row 0, column 1: header
row 1, column 1: content
```

Column 1 and row 1 stretch. `SetHeader` and `SetSidebar` are explicit Go
replacement APIs; `AddContentWidget` is the QML default-property equivalent.
Header menu toggles the sidebar; sidebar destinations and header add actions are
forwarded through Layout callbacks.

## Deviations

### Deviation: immediate sidebar transition

QML behavior: Sidebar width animates from 200 to 0 over 200ms with InOutQuad.

Attempted Qt equivalent: property animation support was not needed to establish
and test the widget contract.

Chosen implementation: `SetOpen` changes minimum and maximum widths
immediately. This is deterministic and avoids adding an unverified animation
layer.

Validation: `TestSidebarAndLayoutComposition` checks open geometry and menu
state transitions.

Follow-up: add a QPropertyAnimation only if screenshot review requires the
transition and the generated MIQT API is verified.

### Deviation: Loader/Repeater/default properties

QML uses Loader, Component, Repeater, and default property aliases. Qt Widgets
uses explicit Go setters, typed models, layouts, and callbacks instead. This
keeps ownership and replacement behavior visible and avoids introducing a QML
runtime into the Go port.

### Deviation: theme palette

The Go theme is palette-derived while the QML theme uses fixed color values.
Components reuse `rapid/widget/theme` rather than duplicating QML constants. Existing
project-wide `theme.TouchTarget` and compact button styling are preserved.

## Ownership and MIQT safety

Layout insertion transfers Qt child ownership to the containing layout/widget.
Go callers retain references but must not manually delete installed children.
Section replacement detaches and hides old layout items; it does not manually
free them. No virtual event override is installed on caller-owned windows or
parents. The only event hooks are on directly constructed row widgets.

## Resource and icon notes

`ui/icon.go` uses the canonical `assets/icons` root and embedded resource URLs.
`TintedPixmap` uses `QImageReader.SetScaledSize` before decoding SVGs so the
vector is rasterized at its final requested size rather than loading a default
bitmap and scaling it down. Accepted `qrc:/...` forms are normalized to the
registered `:/icons/...` path.

Generated resource artifacts are expected to be maintained with:

```text
task build:icon
go generate ./ui
```

## Unresolved work

- Download and settings page implementations remain intentionally deferred.
- Sidebar open/close is immediate instead of animated.
- No full application build or screenshot recapture was run.
- Visual parity should be reviewed after a user-run screenshot; compilation and
  offscreen geometry do not prove pixel parity.
