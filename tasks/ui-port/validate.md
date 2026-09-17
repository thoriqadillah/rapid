# MIQT UI Port Validation

Date: 2026-09-17

Scope: `ui/` reusable-widget port only.

Compared:

- Go implementation in `ui/`
- QML behavior specification in `rapid/qml/ui/`
- Current Go demo wiring in `main.go`
- Current theme contract in `theme/theme.go` and `rapid/qml/Theme.qml`
- MIQT Qt6 generated API available from `github.com/mappu/miqt@v0.14.0`

This report describes the current working tree, including uncommitted changes. It does not assume that the current files match `HEAD`.

## Validation commands run

Passed:

```text
go vet ./ui
git diff --check
```

Failed:

```text
env -u QT_QPA_PLATFORM go test -count=1 ./ui
```

Current failing test:

```text
ui/rbutton_test.go:27: text button does not have QML-equivalent horizontal padding
```

No application build was run. The app was not launched.

## Executive result

The reusable components are not yet feature-complete or visually validated against the QML originals.

The most important unresolved issue is the footer geometry:

1. The current demo adds a second `QHBoxLayout` inside `RDialog.FooterLayout` at `main.go:68-75`.
2. `RDialog.FooterLayout` already contains a leading stretch at `ui/rdialog.go:49`.
3. `RButton` currently has no explicit size policy in the current working tree.
4. The result is a nested-layout sizing path that has not been tested using actual widget geometries.
5. The screenshot-reported footer problem therefore remains unresolved. A wrapper was tried previously and produced unwanted geometry, so the next implementation should measure the direct layout rather than introduce another wrapper by guesswork.

The port has useful behavior coverage, but the tests mostly inspect properties and stylesheets. They do not yet prove rendered geometry, mouse interaction, keyboard interaction, or pixel-level visual parity.

## Status summary

| Area | Status | Main evidence | Priority |
|---|---|---|---|
| Icon lookup | Partial | `ui/icon.go:14-140` | Medium |
| MIQT embedded resources | Implemented | `ui/resources_generated.go`, `ui/resources.rcc`, and the `build:icon` task in `Taskfile.yml` | Done for current resource scope |
| Theme parity | Diverged | `theme/theme.go:57-75`, `theme/theme.go:108`; QML `Theme.qml:6-91` | High |
| `RButton` API | Mostly present | `ui/rbutton.go:37-267` | Medium |
| `RButton` visual parity | Partial | Current padding and disabled/focus rules differ from QML | High |
| `RButton` size behavior | Unresolved | No current size policy; footer screenshot issue | Critical for current UI |
| `RSwitch` API | Partial | `ui/rswitch.go:28-143` | Medium |
| `RSwitch` visual parity | Partial | Unchecked color and interaction differences | High |
| `RTextField` API | Partial | `ui/rtextfield.go:28-268` | High |
| `RTextField` behavior | Partial | `selectByMouse`, loading, wrap behavior are not implemented | High |
| `RDialog` structure | Present | `ui/rdialog.go:24-59` | Medium |
| `RDialog` lifecycle | Partial | `OpenFor`, overlay, owner-close behavior | High |
| `RDialog` sizing | Partial | max height is only a maximum constraint | High |
| Footer integration | Not validated | `main.go:68-75`, `ui/rdialog.go:46-50` | Critical for current screenshot |
| UI tests | Partial | Five test files exist; button padding expectation is stale against intentional compact sizing | High |
| Full UI integration | Missing | No Go equivalents for pages/components yet | Out of current reusable-widget phase |

# Detailed findings

## 1. Icon system

### Implemented

`ui/icon.go` now has:

- canonical `assets/icons` directory constant
- `ResolveIconPath`
- `IconPath`
- checked icon loading through `LoadIconChecked`
- nil-safe `LoadIcon`
- SVG tinting through `TintedPixmap`
- a `go:generate` directive that calls the repository-owned MIQT resource generator
- generated `ui/resources_generated.go` with `go:embed`
- generated `ui/resources.rcc` registered during package initialization

Relevant lines:

```text
ui/icon.go:14    iconDir = "assets/icons"
ui/icon.go:16    miqt-rcc generation directive
ui/icon.go:22    ResolveIconPath
ui/icon.go:80    IconPath
ui/icon.go:90    TintedPixmap
ui/icon.go:114   LoadIconChecked
ui/icon.go:135   LoadIcon
```

### Remaining gaps

#### 1.1 Embedded MIQT resource package is implemented

The resource wrapper now exists:

```text
ui/resources_generated.go
ui/resources.rcc
```

`ui/resources_generated.go`:

- embeds `resources.rcc` with `go:embed`
- registers the binary resource data with `qt.QResource_RegisterResourceWithRccData` during package initialization
- is marked generated and contains no machine-specific generator directive

The source manifest is:

```text
assets/icons/icons.qrc
```

Regeneration is repository-owned:

```text
Taskfile.yml (`build:icon`)
```

The `build:icon` task searches `PATH` for `rcc`, then the project PySide6 venv locations, generates both artifacts, and removes the generated tool directive before formatting the Go wrapper. If `rcc` is missing, it tells the user to run `python3 -m pip install "PySide6>=6.8,<7.0"`.

Validated commands:

```text
go generate ./ui
go test -count=1 ./ui
```

Both pass in the current workspace. The generated `.rcc` file is a Qt Binary Resource file and contains all 30 entries from the manifest.

#### 1.2 `qrc:/` handling is not fully verified

`ResolveIconPath` passes `:/...` and `qrc:/...` strings through at `ui/icon.go:27`, but `TintedPixmap` ultimately loads through `QPixmap` at `ui/icon.go:98`.

The embedded resource path is now tested by `ui/icon_test.go`:

```text
load :/icons/MdiLightPlus.svg
rasterize and tint
assert non-null QPixmap
```

This proves the generated package initializer registered the Qt resource before the UI test runs. A packaged executable smoke test remains outstanding because the user explicitly requested no application build or launch during this validation pass.

#### 1.3 Error observability is only available through the checked API

`LoadIcon` returns a blank `QIcon` on error at `ui/icon.go:135-140`. That is useful for widget safety but can make a production asset failure look like a missing icon.

Recommended follow-up:

- keep `LoadIcon` nil-safe
- make development/demo callers use `LoadIconChecked` or log a controlled warning
- do not silently hide missing required icons in the final application

## 2. Theme contract

### Current divergence

The QML theme defines fixed values, including:

```text
colorBackground = #1e1f29
colorSurface = #282A36
touchTarget = 36
colorDanger = #FF5555
```

Current Go theme behavior differs:

```text
theme/theme.go:57-75    structural colors come from the system palette
theme/theme.go:33        ColorDanger is #CB3A2A instead of QML #FF5555
theme/theme.go:108       TouchTarget is 32 instead of QML 36
```

The palette initialization is called from `main.go:16` and from test setup in `ui/qt_test.go:17`.

This is not automatically wrong, but it is a port decision that needs to be explicit. It changes the visual contract of every widget.

### Required decision

Choose one of these before calling visual parity complete:

1. Match QML exactly with fixed theme values.
2. Intentionally make the MIQT port system-palette-aware and update the parity specification accordingly.

Do not mix the two silently. Current QML-to-Go comparisons will fail or become ambiguous while the theme is split.

### Theme test gap

`theme/theme_test.go` verifies that palette fields initialize and that muted alpha is preserved, but it does not verify parity with `rapid/qml/Theme.qml`.

Add either:

- a fixed-parity test for values that must match QML, or
- an explicit test/documentation that palette mapping is intentional

## 3. `RButton`

Source files:

```text
Go:  ui/rbutton.go
QML: rapid/qml/ui/RButton.qml
Tests: ui/rbutton_test.go
```

### Implemented API

The current Go type provides:

```text
NewRButton
NewRButtonIcon
SetText
Text
SetIconSource
SetIconSize
SetIconOnly
SetLink
SetTooltip
SetCornerRadius
SetEnabled
```

Variants are present:

```text
BaseVariant
PrimaryVariant
InfoVariant
SecondaryVariant
GhostVariant
WarningVariant
DangerVariant
LinkVariant
```

`LinkVariant` is a Go compatibility addition. QML models link state as `property bool link` rather than as an enum member.

### Remaining gaps and contradictions

#### 3.1 Current test and current stylesheet disagree

The current test expects:

```text
ui/rbutton_test.go:26-27
padding: 4px 24px
```

The current implementation emits a stylesheet with:

```text
ui/rbutton.go:237
padding: 0 <paddingH>px
```

The current implementation therefore fails the test because its vertical padding is `0`, while the test expects QML's vertical padding of `4`.

Separately, the current implementation still uses horizontal padding:

```text
ui/rbutton.go:166   paddingH = theme.SpacingXl
ui/rbutton.go:168   icon-only paddingH = theme.SpacingSm
ui/rbutton.go:237   padding: 0 <paddingH>px
```

This conflicts with the current project decision that horizontal padding was intentionally removed because the buttons became too wide.

Required action:

- decide whether the source-of-truth intent is compact zero-horizontal-padding behavior or QML padding parity
- update the test to match that explicit decision
- do not use horizontal padding as a footer-width fix

The existing test failure is a real validation failure, not a cosmetic warning.

#### 3.2 Footer width policy is not implemented in the current tree

The current `NewRButton` constructor sets minimum height but does not set an explicit size policy:

```text
ui/rbutton.go:47-48
```

For the reported footer problem, the button should have a deliberate horizontal policy. The likely compact policy is:

```text
horizontal = Maximum
vertical = Fixed
```

However, this must be tested against the actual caller layout. A policy alone may not fix a nested layout item if the caller adds a child layout rather than the buttons directly.

Required work:

- preserve the compact stylesheet
- set or otherwise enforce a content-width policy deliberately
- add buttons directly to the footer action layout, or define an explicit footer API that owns the action group
- test actual `Geometry()` values after showing and processing events
- test both direct button insertion and the current nested-layout insertion

#### 3.3 The current demo creates a nested footer layout

The current demo does this:

```text
main.go:68   buttons := qt.NewQHBoxLayout2()
main.go:73   buttons.AddWidget(cancelBtn.QWidget)
main.go:74   buttons.AddWidget(okBtn.QWidget)
main.go:75   dialog.FooterLayout.AddLayout(buttons.QLayout)
```

`RDialog.FooterLayout` already contains a leading stretch:

```text
ui/rdialog.go:46-50
```

This creates a layout-inside-layout arrangement. The inner layout is the object that receives the buttons, not the footer layout itself.

This is the strongest current explanation for the screenshot-reported footer geometry issue. The reusable widget cannot fully control the geometry of a layout supplied by a caller.

Recommended implementation direction:

- add `RDialog.AddFooterWidget` and use it for each button, or
- add a narrow `AddFooterLayout` helper whose alignment and size policy are explicit, or
- keep the public `FooterLayout` but document that callers must add buttons directly and not nest an expanding layout

Do not add another wrapper blindly. A prior wrapper attempt changed geometry incorrectly.

#### 3.4 Ghost variant is now implemented

The current switch maps Ghost to transparent background at `ui/rbutton.go:178-179`.

This item from the original task is complete.

#### 3.5 Link behavior is only approximately equivalent

QML link behavior includes:

```text
zero horizontal padding
zero vertical padding
font underline while hovered
transparent background
```

The current Go link stylesheet at `ui/rbutton.go:188-215` has no padding declaration and still sets `min-height` to the touch target.

The resulting vertical geometry is not the same as QML's `verticalPadding: 0` behavior.

Required decision:

- preserve touch-target height for accessibility, while documenting the visual difference, or
- match QML's compact link height

#### 3.6 Disabled opacity is not equivalent

QML applies:

```qml
opacity: enabled ? 1 : 0.5
```

The current QSS substitutes muted background and foreground colors at `ui/rbutton.go:249-252`. This can be acceptable as a platform-specific equivalent, but it is not the same behavior and is not tested visually.

Add a rendering or stylesheet contract test, or explicitly accept the difference.

#### 3.7 Icon/text spacing is not explicitly controlled

QML defines:

```qml
spacing: Theme.spacingXs
```

The Go implementation uses the native `QPushButton` layout and does not set a corresponding icon/text gap. The resulting gap is style-dependent.

If exact visual parity matters, use a custom layout or a platform-supported style setting. If native spacing is acceptable, record that as an intentional deviation.

#### 3.8 Hover and focus transitions lack geometry/render tests

The implementation updates icon tint on enter/leave at `ui/rbutton.go:51-62` and changes foreground colors in QSS. Tests do not synthesize enter/leave events or inspect the resulting hover stylesheet.

Missing tests:

- normal button hover
- outlined button hover
- icon tint after hover
- disabled hover
- focus ring
- actual size hint and geometry
- icon-only size hint

## 4. `RSwitch`

Source files:

```text
Go:  ui/rswitch.go
QML: rapid/qml/ui/RSwitch.qml
Tests: ui/rswitch_test.go
```

### Implemented

The Go implementation has:

- `QCheckBox` base control
- 36 by 20 fixed size
- active and inactive color setters
- custom painting
- 16 ms timer
- elapsed-time animation targeting 120 ms
- checked/toggled state
- focus policy
- custom mouse-release handling
- timer parented to the switch's `QObject`

Relevant lines:

```text
ui/rswitch.go:35       fixed dimensions
ui/rswitch.go:46       parent-owned timer
ui/rswitch.go:50-56    toggled animation start
ui/rswitch.go:60-64    custom mouse release
ui/rswitch.go:72-94    color API
ui/rswitch.go:96-113   animation
ui/rswitch.go:115-133  paint
```

### Remaining gaps

#### 4.1 Unchecked state uses the wrong base color

QML uses:

```qml
color: root.checked ? root.activeColor : root.inactiveColor
```

The Go painter currently starts from:

```text
ui/rswitch.go:120
background := blend(theme.ColorSurface, s.activeColor, s.progress)
```

At progress `0`, the switch uses `theme.ColorSurface`, not `inactiveColor`.

This means `SetInactiveColor` does not control the fully unchecked visual state. It only affects the later interpolation indirectly if the implementation is changed.

Required fix:

```text
blend(s.inactiveColor, s.activeColor, s.progress)
```

with the final unchecked state exactly equal to `inactiveColor`.

#### 4.2 Mouse handling is broader than QML

QML uses a `MouseArea` with `acceptedButtons: Qt.NoButton`, leaving native `Controls.Switch` interaction in charge.

The Go override at `ui/rswitch.go:60-64` toggles on any mouse release when enabled and does not inspect the mouse button. It also does not call the superclass handler.

Potential consequences:

- right-click or other release events can toggle
- native pressed/released behavior is bypassed
- double-toggle behavior is not proven

Required tests:

- left-click once toggles once
- right-click does not toggle
- keyboard activation toggles once
- disabled switch does not toggle
- repeated clicks do not leave animation state inconsistent

#### 4.3 Focus visual state is not implemented

The QML control inherits Qt Quick focus behavior. The Go implementation sets `StrongFocus` but custom-paints no focus indication.

Decide whether the default Qt focus indicator is sufficient. If not, paint one or style it.

#### 4.4 Accessibility contract is not tested

The task calls for an accessibility role/name. Current code does not set an accessible name or description, and tests do not verify either.

Add a public setter or rely on caller-set widget metadata explicitly.

#### 4.5 Paint resource lifetime is not validated

`blend` allocates a new `QColor` every paint at `ui/rswitch.go:137-139`. `paint` also allocates brushes repeatedly.

This may be acceptable, but MIQT ownership and cleanup should be checked under repeated repaint. There is no stress or leak-oriented test.

#### 4.6 Animation tests do not exercise the event loop

The current test manually changes `animationAt` and calls `step` directly. It does not prove:

- QTimer timeout callbacks fire
- animation repaints over time
- toggling rapidly reverses correctly
- the widget reaches the target after real event processing

Add one bounded offscreen event-loop test.

## 5. `RTextField`

Source files:

```text
Go:  ui/rtextfield.go
QML: rapid/qml/ui/RTextField.qml
Tests: ui/rtextfield_test.go
```

### Implemented

The current type supports:

- label
- error text
- text and placeholder
- password/general echo mode
- read-only
- max length
- validator
- input method hints
- stored select-by-mouse state
- stored loading state
- icon size/color
- prefix/suffix icon pixmaps
- field layout access
- focus proxy
- error preservation when icons change
- text changed callback

### Remaining gaps

#### 5.1 `selectByMouse` is state-only, not behavior

The current code explicitly says:

```text
ui/rtextfield.go:173-177
```

It stores the requested value but does not change the `QLineEdit`. MIQT's generated `QLineEdit` API does not expose a direct `selectByMouse` property like QML.

This is a real parity gap. Options:

- document that Qt Widgets always allow selection and remove the setter
- implement a subclass/event policy that blocks mouse selection when false
- keep the state only but mark it unsupported

Do not call this complete until the choice is explicit.

#### 5.2 `wrapMode` is missing

QML exposes `field.wrapMode`, but the Go component uses `QLineEdit`, which is single-line and has no equivalent wrap behavior.

Required decision:

- explicitly document `wrapMode` as unsupported for a single-line port, or
- change the underlying control to a suitable multi-line editor where consumers require it

#### 5.3 `loading` is state-only

The current code stores loading at `ui/rtextfield.go:183-189` but does not render a spinner, disable input, or expose a loading indicator.

The QML reusable component also only declares the property and does not render a loading indicator in `RTextField.qml`, so this can remain state-only if documented. It should not be described as fully implemented UI behavior.

#### 5.4 Wrapper click-to-focus is not fully proven

The wrapper installs `OnMousePressEvent` at `ui/rtextfield.go:55-58`, while the actual child `QLineEdit` receives mouse events itself.

The focus proxy at `ui/rtextfield.go:49` is correct in principle, but there is no real mouse event test proving that clicking the label or wrapper focuses the field.

Add an offscreen event test or explicitly limit the guarantee to keyboard focus and focus proxy behavior.

#### 5.5 Side icon painting is not fully layout-safe

Icons are painted directly inside `QLineEdit` at `ui/rtextfield.go:258-268`.

The padding is adjusted in `applyStyle`, but the painter does not verify:

- field width is large enough for both icons and text
- suffix icon remains inside the text margins at narrow widths
- icons do not overlap selected text or the clear button
- icon paths loaded from Qt resources work

Add geometry tests for:

- prefix only
- suffix only
- both icons
- narrow field
- icon size changes
- icon removal

#### 5.6 Field-row extensibility is not equivalent to QML default data

`AddFieldWidget` adds a widget to the `QHBoxLayout`, but there is no API for inserting content before or after the field deterministically. The underlying field is added first with stretch `1` at `ui/rtextfield.go:62`.

This may be sufficient for current consumers, but a future port of QML `fieldRowData` should define insertion order and stretch behavior.

#### 5.7 Error label sizing needs an actual layout test

Word wrapping is enabled at `ui/rtextfield.go:45`, but there is no test that a long error message increases height or remains visible without clipping.

The QML error label explicitly uses `Layout.fillWidth` and `Text.Wrap`.

Add a shown-widget geometry test with a long error message.

## 6. `RDialog`

Source files:

```text
Go:  ui/rdialog.go
QML: rapid/qml/ui/RDialog.qml
Tests: ui/rdialog_test.go
```

### Implemented

The current Go dialog has:

- minimum width
- modal/window modality settings
- themed stylesheet
- body and footer layouts
- `OpenFor`
- opened callbacks
- maximum-height constraint setter
- overlay creation
- overlay refresh helper
- hide/destroy overlay cleanup

### Remaining gaps

#### 6.1 Footer integration is still unresolved

The dialog has a direct footer layout with a leading stretch at `ui/rdialog.go:46-50`.

The current demo then adds another layout into it at `main.go:68-75`.

This is not the same as QML's single `RowLayout` with right-to-left direction. It has not been validated with real geometries and is the current likely source of the reported footer width problem.

Required work:

- define whether callers add buttons directly or add one action layout
- prevent nested layout expansion from changing compact button geometry
- test `FooterLayout.ItemAt`, child widget geometry, and dialog width after `Show()` plus event processing
- verify visual ordering against QML's `Qt.RightToLeft`

#### 6.2 `SetMaxHeight` does not implement QML height calculation

QML calculates:

```text
height = min(maxHeight, intrinsicHeight)
```

The Go method only calls `SetMaximumHeight` at `ui/rdialog.go:97-106`.

It does not:

- calculate intrinsic height
- resize to the smaller of intrinsic and max height
- add scrolling when content exceeds max height
- distinguish max height zero from an intrinsic-height layout

This is a constraint, not full parity.

#### 6.3 Owner-close behavior was deliberately dropped due to MIQT restrictions

The current comment at `ui/rdialog.go:65-68` explains that owner virtual event overrides are not allowed for arbitrary caller-owned widgets.

That avoids the MIQT panic, but it means QML's owner `onClosing` connection is not implemented. The dialog will not automatically close when the owner closes unless another layer handles it.

Required design options:

- expose explicit `CloseForOwner`/cleanup calls to the window layer
- install a Qt signal connection if MIQT exposes an appropriate non-virtual close signal
- create an owner-specific Go window type and own the event override there

Do not reintroduce `owner.OnResizeEvent` or `owner.OnCloseEvent` on arbitrary widgets. MIQT panics for that use case.

#### 6.4 Overlay resize is manual

`RefreshOverlay` at `ui/rdialog.go:142-146` must be called after owner resizing. No automatic resize tracking exists.

This is an intentional safety compromise after the MIQT virtual override panic, but it is a remaining behavior gap.

#### 6.5 Escape behavior is unverified

QML explicitly declares an Escape shortcut at `RDialog.qml:29-32`.

The Go dialog relies on default `QDialog` behavior. There is no test that pressing Escape closes the dialog and hides the overlay.

Add an event test or implement an explicit shortcut/key path using MIQT APIs.

#### 6.6 Open semantics differ from QML

`OpenFor` calls `Show`, `Raise`, `ActivateWindow`, and `SetFocus` at `ui/rdialog.go:80-83`.

QML calls `showNormal`, `raise`, `requestActivate`, and emits `opened`.

The Go implementation does not call `AdjustSize` or compute intrinsic content size before showing. The resulting dialog size is therefore not proven equivalent.

#### 6.7 Footer direction is not equivalent

QML uses:

```qml
layoutDirection: Qt.RightToLeft
```

The Go implementation uses a leading stretch but keeps default left-to-right layout order. With buttons added as Cancel then OK, the visual order may not match QML.

Test and document the intended visual order.

## 7. Current demo is not a reliable UI parity harness

`main.go` is a useful smoke/demo program, but it currently has several limitations:

- it only exercises one instance of each component
- the dialog uses a nested footer layout
- it does not exercise `OpenFor`; it calls `dialog.Show()` at `main.go:76`
- it does not exercise dialog overlay behavior
- it does not exercise max height
- it does not exercise keyboard interactions
- it does not exercise disabled switch painting
- it does not exercise icon-only buttons from the real QML call patterns except the helper demo

Before using the demo as a visual reference, make it represent the intended Go API and footer contract.

## 8. Test coverage gaps

Existing tests are valuable but incomplete.

### Covered reasonably

- icon path success/failure basics
- invalid icon size
- valid SVG rasterization from filesystem
- button variant construction
- basic button text/icon/toggle callback behavior
- switch state and timer setup
- text field state, error persistence, icons, password, read-only, max length
- dialog construction, OpenFor callback, overlay visibility, maximum-height property

### Not covered

- current button geometry and size hints
- current footer geometry
- nested footer layout behavior
- footer visual ordering
- actual mouse hover enter/leave
- actual keyboard button activation
- outlined hover pixel/style behavior
- disabled opacity rendering
- switch mouse event behavior
- switch keyboard activation
- switch unchecked custom inactive color
- switch real timer progression
- switch focus and accessibility
- text field wrapper mouse focus
- text selection behavior
- `selectByMouse=false`
- long wrapped error geometry
- text field narrow-width icon collision
- icon loading from embedded MIQT resources
- dialog Escape behavior
- dialog owner-close behavior
- dialog overlay refresh after owner resize
- dialog intrinsic-height/max-height behavior
- destruction and repeated open/close cycles

## 9. Scope boundary

The following are intentionally not part of the current reusable-widget phase and remain unported:

```text
rapid/qml/Main.qml
rapid/qml/pages/*
rapid/qml/components/*
rapid/qml/Navigation.qml
rapid/qml/NotificationItem.qml
```

Those files use the reusable components extensively. Their migration should wait until the four reusable widget contracts are stable, especially:

1. button width and footer insertion
2. text field icon/focus contract
3. dialog lifecycle and sizing
4. switch interaction

## Recommended implementation order

### P0. Resolve the current validation contradiction

1. Confirm the intended button sizing rule with the project owner: zero horizontal padding is the current stated preference.
2. Update `ui/rbutton_test.go` so it tests that preference rather than the old QML padding.
3. Decide whether vertical padding remains zero or follows QML's `SpacingXs`.
4. Do not use button padding as the footer fix.

### P1. Fix and prove the footer contract

1. Decide whether `FooterLayout` receives direct buttons or a single action layout.
2. Remove the nested layout from `main.go`, or provide an explicit helper for it.
3. Set an intentional compact size policy only if it works with the chosen direct layout.
4. Add a geometry test after `Show`, `AdjustSize`, and `ProcessEvents`.
5. Assert button widths are content-sized and the group is right-aligned.
6. Assert the visual order matches the intended QML order.

### P1. Fix `RSwitch` unchecked color

Change the paint interpolation base from `theme.ColorSurface` to `inactiveColor`, then test both fully unchecked and fully checked states.

### P1. Decide theme parity

Document whether palette-derived colors and `TouchTarget = 32` are intentional. If exact QML parity is the goal, restore or map the QML values deliberately.

### P2. Complete text-field behavior decisions

1. Decide whether `selectByMouse` is supported, always-on, or unsupported.
2. Document `wrapMode` as unsupported for `QLineEdit`, unless a multiline control is required.
3. Decide whether loading remains state-only.
4. Add long-error and narrow-icon geometry tests.

### P2. Complete dialog lifecycle tests

1. Add Escape test.
2. Define owner-close cleanup without virtual overrides on arbitrary owner widgets.
3. Test overlay refresh after owner resize.
4. Implement or document intrinsic-height behavior.

### P3. Finish resource packaging verification

The wrapper generation and direct qrc test are complete. Remaining verification is packaging-only:

1. Build a standalone binary with the generated wrapper.
2. Run it from a non-repository working directory.
3. Confirm the embedded `:/icons/...` path works with no `assets/icons` filesystem access.

These steps were intentionally not run because the user requested no application build or launch.

### P3. Add visual integration harness

Use a small dedicated UI test/demo window that contains:

- normal, outlined, ghost, link, icon-only, disabled, and icon buttons
- checked and unchecked switches
- text fields with every icon/error combination
- a dialog with direct footer buttons

Use geometry assertions first. Use screenshots only after geometry is deterministic.

## Definition of complete for the reusable `ui/` phase

The reusable port should not be called complete until all of the following are true:

- `go test -count=1 ./ui` passes without stale expectations
- `go vet ./ui` passes
- button sizing intent is explicit and tested
- footer buttons have deterministic geometry in the actual integration path
- no nested-layout ambiguity remains in the demo API
- Ghost, Link, outlined hover, icon-only, tooltip, and disabled behavior are documented as exact or intentional deviations
- switch unchecked and checked colors both use the configured color API
- switch mouse and keyboard activation are tested
- text-field `selectByMouse`, `wrapMode`, and loading decisions are documented
- text-field error/icon/narrow-width geometry is tested
- dialog Escape, overlay cleanup, max-height, and owner lifecycle behavior are tested or explicitly documented as unsupported
- theme divergence from QML is an explicit decision
- MIQT icon resources work from a packaged or non-root execution context
- only after that should `rapid/qml/components` and page-level migration begin

## Bottom line

The port is a solid structural start, but it is not finished. The current footer screenshot issue is a genuine unresolved integration problem, not evidence that another wrapper should be added immediately. The current test failure also needs resolution because it shows the implementation and the intended button-size contract have diverged.

The next implementation pass should be narrow:

1. settle button sizing intent,
2. test the real footer layout path,
3. fix the footer without changing the compact visual style,
4. fix the unchecked switch color,
5. then close the dialog/text-field behavior gaps.
