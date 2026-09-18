The `ui/` port is a **working scaffold, not feature parity**.
`go test ./...` passes, but there are no UI tests, and the
current implementation has several deliberate shortcuts.
MIQT is a reasonable fit here: its repository documents
QtCore, QtGui, QtWidgets, QtSvg, subclassing support, and
`uic`/`rcc` tooling. Use Qt Widgets rather than trying to
reproduce the QML layer in Go.
## Current UI surface
### Already present in Go
```text
ui/icon.go
ui/rbutton.go
ui/rswitch.go
ui/rtextfield.go
ui/rdialog.go
```
### Original reusable UI source
```text
rapid/qml/ui/RButton.qml
rapid/qml/ui/RSwitch.qml
rapid/qml/ui/RTextField.qml
rapid/qml/ui/RDialog.qml
```
### Current validation
```text
go test ./...
```
passes for:
```text
rapid
rapid/widget/theme
rapid/widget/ui
```
That only proves compilation. There are currently no UI
behavior tests.
---
# Port plan
## Phase 1: Lock down the component contract
Before changing implementation, treat the QML files as the
behavioral specification.
Do not port the pages or larger components yet:
```text
rapid/qml/pages/*
rapid/qml/components/*
rapid/qml/Main.qml
```
For this phase, only port the reusable components and the
support they directly require.
The Go API should remain compatible with the current demo in
`main.go`, especially:
```go
ui.NewRButton(...)
ui.NewRSwitch(...)
ui.NewRTextField()
ui.NewRDialog(...)
```
Avoid changing constructor signatures unless there is a strong
 reason. Add setters and helper methods instead.
---
# 1. `ui/icon.go`
## Current state
`ui/icon.go` provides:
```go
IconPath
TintedPixmap
LoadIcon
```
The current implementation assumes icons are available at:
```text
assets/icons/<name>
```
That matches the source tree, but it is not equivalent to the
PySide resource system.
PySide registers the compatibility copy:
```python
from rapid.qml.icons import icons_rc
```
and then uses:
```text
:/icons/rapid.svg
qrc:/icons/MdiLightPlus.svg
```
The Go implementation currently loads files from a relative
filesystem path.
## Work still needed
### 1. Decide the resource strategy
The UI agent should choose one of these and document it:
#### Preferred: Qt resource system
Use the existing:
```text
assets/icons/icons.qrc
```
and generate/register a Go-compatible resource package with
MIQT’s `rcc` support if practical.
Benefits:
- works from any working directory
- works in packaged binaries
- matches the original `qrc:/icons/...` behavior
- avoids shipping a fragile relative path
#### Temporary development fallback
Resolve icons relative to the executable or an explicitly
configured asset directory.
This is acceptable for the first widget demo, but not for a
finished build.
### 2. Make icon lookup testable
Add an API that distinguishes:
- empty icon name
- missing icon
- valid icon
- invalid size
- valid SVG that cannot be rasterized
For example, the implementation can keep `LoadIcon`, but the
agent should ensure failures are observable during development
 rather than silently producing a blank button.
### 3. Verify SVG tinting
`TintedPixmap` currently:
1. loads the SVG through `QPixmap`
2. draws it into a transparent pixmap
3. uses `SourceIn` composition
4. fills it with the target color
Test at least:
- `size <= 0` returns safely
- missing path does not crash
- valid icon returns a non-null pixmap
- transparent pixels stay transparent
- tinted pixels use the requested color
- repeated tinting does not mutate the source
MIQT’s QtSvg support may provide a more reliable
`QSvgRenderer` path if direct `QPixmap` loading is
inconsistent. Verify the generated MIQT API before choosing
it.
### 4. Package icon inventory
The current icon set includes:
```text
rapid.svg
MdiLightBook.svg
MdiLightCheck.svg
MdiLightChevronDown.svg
MdiLightClock.svg
MdiLightConsole.svg
MdiLightContentPaste.svg
MdiLightDownload.svg
MdiLightFile.svg
MdiLightFilmstrip.svg
MdiLightFolder.svg
MdiLightFormatAlignBottom.svg
MdiLightHelpCircle.svg
MdiLightLoading.svg
MdiLightLogout.svg
MdiLightMagnify.svg
MdiLightMenu.svg
MdiLightMusic.svg
MdiLightPause.svg
MdiLightPencil.svg
MdiLightPicture.svg
MdiLightPlay.svg
MdiLightPlus.svg
MdiLightSettings.svg
MdiLightStop.svg
MdiLightUndoVariant.svg
MdiLightViewModule.svg
MdiSquareRounded.svg
MaterialSymbolsLightCloseRounded.svg
MaterialSymbolsLightDeleteForeverOutlineSharp.svg
```
Do not add a new icon library during this port.
---
# 2. `ui/rbutton.go`
Source comparison:
- Go: `ui/rbutton.go`
- QML: `rapid/qml/ui/RButton.qml`
## Current state
The Go implementation has variants for:
```text
Base
Primary
Info
Secondary
Ghost
Warning
Danger
Link
```
It supports:
- text
- variants
- outlined mode
- icon assignment
- clicked signals inherited from `QPushButton`
- basic hover and disabled styles
## Missing or incorrect behavior
### 1. Ghost variant is wrong
In QML:
```qml
GhostVariant -> transparent background
```
The current Go switch does not give `GhostVariant` a
transparent background. It falls through to the base button
color.
Fix the variant mapping.
### 2. Link is modeled differently
The QML component has:
```qml
property bool link: false
```
The QML enum does not include `LinkVariant`.
The Go implementation introduced:
```go
LinkVariant
```
Do not break current Go callers, but add a proper link
property:
```go
func (b *RButton) SetLink(link bool)
```
Recommended compatibility behavior:
- retain `LinkVariant`
- map `LinkVariant` internally to link styling
- allow `SetLink(true)` independently of the variant
### 3. Missing `iconOnly`
QML supports:
```qml
property bool iconOnly: false
```
Implement:
```go
func (b *RButton) SetIconOnly(on bool)
```
Behavior:
- text plus icon: normal horizontal padding
- icon only: compact padding
- link button: zero padding
- icon-only buttons still meet the `Theme.TouchTarget` minimum
 size
This is used extensively by the existing QML UI, including
download actions and header controls.
### 4. Missing configurable icon size
QML supports:
```qml
property int iconSize
```
The Go implementation hardcodes:
```go
theme.IconMd
```
Add:
```go
func (b *RButton) SetIconSize(size int)
```
Store the requested size and regenerate the icon when the size
 or icon source changes.
### 5. Missing tooltip
QML supports:
```qml
property string tooltip
```
and shows it after a delay on hover.
Add:
```go
func (b *RButton) SetTooltip(text string)
```
The simplest implementation is Qt’s native widget tooltip
support. Match the QML behavior as closely as practical:
- hidden when empty
- shown on hover
- approximately 500 ms delay
- no tooltip for disabled controls unless Qt handles that
automatically
### 6. Outlined hover color is incomplete
The QML behavior is:
- normal outlined button: transparent background
- normal outlined button: colored border
- hovered outlined button: filled with variant color
- hovered outlined button: inverted text and icon color
The current Go code has this comment:
```go
// ponytail: hover fg stays base text; QML inverts it, minor
```
That is a known parity gap. Remove the shortcut and implement
the inverted hover foreground.
The icon must change color as well, not only the text.
Possible implementations:
- create normal, active, and disabled `QIcon` pixmaps
- update the icon on hover events
- subclass the button and paint the icon manually
Use the smallest approach that actually gives correct hover
rendering.
### 7. Missing configurable corner radius
QML supports:
```qml
property int cornerRadius
```
Add:
```go
func (b *RButton) SetCornerRadius(radius int)
```
The current implementation hardcodes `theme.RadiusSm`.
### 8. Disabled styling differs
QML applies:
```qml
opacity: enabled ? 1 : 0.5
```
The current QSS uses fixed white alpha values:
```css
background-color: rgba(255,255,255,0.5);
color: rgba(255,255,255,0.5);
```
Use a consistent disabled treatment. Avoid making the disabled
 color depend on white if the theme changes.
### 9. Padding and minimum size
Match the QML rules:
```text
normal button: horizontal padding = Theme.SpacingXl
icon-only: horizontal padding = Theme.SpacingMd
link: zero padding
vertical padding = Theme.SpacingXs
minimum height = Theme.TouchTarget
spacing between icon and text = Theme.SpacingXs
```
The current implementation sets minimum height but does not
explicitly reproduce all QML padding behavior.
## Suggested public API
Keep the existing constructor and add:
```go
func (b *RButton) SetLink(bool)
func (b *RButton) SetIconSource(string)
func (b *RButton) SetIconSize(int)
func (b *RButton) SetIconOnly(bool)
func (b *RButton) SetTooltip(string)
func (b *RButton) SetCornerRadius(int)
```
The variant should remain available through the constructor.
---
# 3. `ui/rswitch.go`
Source comparison:
- Go: `ui/rswitch.go`
- QML: `rapid/qml/ui/RSwitch.qml`
## Current state
The Go implementation:
- embeds `QCheckBox`
- custom-paints the switch
- animates using a 16 ms `QTimer`
- supports checked state and toggled callbacks
- supports mouse release over the full pill
- uses a 36 by 20 widget
This is a reasonable MIQT implementation approach.
## Work still needed
### 1. Add active and inactive colors
QML exposes:
```qml
property color activeColor
property color inactiveColor
```
The Go implementation hardcodes:
```go
theme.ColorPrimary
theme.ColorBorder
```
Add:
```go
func (s *RSwitch) SetActiveColor(*qt.QColor)
func (s *RSwitch) SetInactiveColor(*qt.QColor)
```
Store the values on the widget and repaint when changed.
### 2. Match the border
QML draws:
```qml
border.width: 1
border.color: Qt.darker(activeOrInactiveColor, 1.2)
```
The Go implementation currently paints only the fill and knob.
Add the one-pixel border and match the darker-color behavior.
### 3. Match animation timing
QML uses:
```qml
NumberAnimation {
    duration: 120
}
```
The current timer advances by `0.15` every 16 ms, which is
approximately 112 ms. That is close, but make the duration
explicit rather than relying on a step constant.
The agent should either:
- retain the timer and calculate progress from elapsed time,
or
- use an MIQT-exposed animation class if that is simpler and
reliable
Do not introduce a complicated animation abstraction.
### 4. Preserve native interaction
The underlying `QCheckBox` should continue to provide:
- checked state
- toggled signal
- keyboard activation
- enabled/disabled state
- accessibility role where available
- focus handling
The custom mouse handler must not cause double toggles. Test
both mouse and keyboard activation.
### 5. Handle disabled and focus states
The QML source does not define an elaborate disabled
appearance, but the Go widget should not look interactive when
 disabled.
Test:
- disabled switch does not toggle
- disabled switch has an appropriate visual state
- keyboard focus remains possible and visible
- accessibility name can be assigned
### 6. Parent the timer correctly
The current timer is created without an explicit parent:
```go
s.timer = qt.NewQTimer()
```
Use the appropriate MIQT constructor or ownership mechanism so
 the timer is cleaned up with the switch. Verify MIQT
ownership behavior rather than assuming Go garbage collection
manages Qt objects.
### 7. Avoid color object leaks
`blend` allocates a new `QColor` on every paint. That is
acceptable for a first pass, but the implementation agent
should check whether this creates unmanaged Qt allocations
through MIQT.
If necessary, keep the calculation simple but reuse or scope
temporary color objects appropriately.
---
# 4. `ui/rtextfield.go`
Source comparison:
- Go: `ui/rtextfield.go`
- QML: `rapid/qml/ui/RTextField.qml`
This is the largest remaining component gap.
## Current state
The Go version supports:
- label
- error text
- text
- placeholder
- password mode
- read-only mode
- maximum length
- text-changed callback
- prefix icon
- suffix icon
- custom icon painting
## Concrete bug to fix
In `setSideIcon`:
```go
w.handleError("")
```
Adding or changing an icon clears the error border state even
if an error message is currently visible.
For example:
```go
field.SetError("Invalid URL")
field.SetPrefixIcon(...)
```
will restore the normal border.
Store the error text on `RTextField` and always style from the
 current state:
```text
error text != empty -> danger border
otherwise -> normal or focus border
```
Do not pass an empty string merely because the icon changed.
## Missing QML properties
The QML component exposes:
```qml
label
error
text
placeholderText
echoMode
readOnly
maximumLength
validator
inputMethodHints
wrapMode
selectByMouse
prefixIcon
suffixIcon
iconSize
iconColor
field
loading
```
The Go version is missing or incomplete for:
- validator
- input method hints
- wrap mode
- select-by-mouse configuration
- icon size
- icon color
- loading state
- field-row extensibility
- general echo mode support
Add only the properties that the Go UI will actually need, but
 make the decision explicit.
## Recommended API additions
```go
func (w *RTextField) SetIconSize(int)
func (w *RTextField) SetIconColor(*qt.QColor)
func (w *RTextField) SetSelectByMouse(bool)
func (w *RTextField) SetEchoMode(mode qt.QLineEdit__EchoMode)
func (w *RTextField) SetValidator(*qt.QValidator)
func (w *RTextField) SetInputMethodHints(...)
func (w *RTextField) SetLoading(bool)
func (w *RTextField) FieldLayout() *qt.QHBoxLayout
func (w *RTextField) AddFieldWidget(*qt.QWidget)
```
The exact MIQT enum and validator types must be checked in the
 generated `qt6` package before implementation.
Keep the convenience method:
```go
SetPassword(bool)
```
for current callers, but implement it through the general
echo-mode setting.
## Layout parity
The QML layout is:
```text
ColumnLayout
  Label, visible when non-empty
  RowLayout
    TextField
  Error Label, visible when non-empty
```
The Go implementation is close, but verify:
- label is hidden without leaving unwanted vertical space
- error label is hidden without leaving unwanted vertical
space
- error text wraps
- field expands horizontally
- field has a 36 px minimum height
- spacing matches `Theme.SpacingSm`
- wrapper can delegate focus to the actual line edit
The current Go code adds a stretch after the error label.
Verify that this does not create unwanted height when the
widget is placed in larger layouts.
## Icon behavior
Match QML:
```text
icon size = Theme.IconMd by default
icon color = Theme.ColorTextMuted by default
left/right padding = base padding + icon size + spacing
icons are non-interactive
icons are vertically centered
```
Add support for:
```go
SetPrefixIcon("")
SetSuffixIcon("")
```
to remove icons cleanly.
Changing an icon must:
- update padding
- preserve the current error state
- repaint
- not alter the text cursor position unexpectedly
## Focus behavior
The QML wrapper focuses the field when the wrapper is clicked.
The Go implementation should use a focus proxy or equivalent
behavior so clicking the label/field wrapper gives focus to
the line edit.
Test:
- clicking field focuses it
- tab enters the field
- selecting text with the mouse works
- read-only still permits selection
- password mode does not expose the raw text visually
## Loading behavior
The QML property exists:
```qml
property bool loading
```
Before implementing a spinner, check how the future Go UI
intends to use it. If no current Go consumer needs it, it can
initially be represented as state only or intentionally
deferred. Do not add a new animation or icon system just for
an unused property.
---
# 5. `ui/rdialog.go`
Source comparison:
- Go: `ui/rdialog.go`
- QML: `rapid/qml/ui/RDialog.qml`
## Current state
The Go implementation supports:
- `QDialog`
- minimum width of 500
- themed background
- body layout
- footer layout
That is only the structural part.
## Missing behavior
### 1. No `OpenFor`
QML exposes:
```qml
function openFor(newOwner)
```
and performs:
1. assign owner
2. show overlay
3. show dialog
4. raise dialog
5. request activation
6. emit `opened`
Add a Go equivalent:
```go
func (d *RDialog) OpenFor(owner *qt.QWidget)
```
Keep `Show()` available through the embedded `QDialog`, but
use `OpenFor` for parity.
### 2. No opened callback
QML declares:
```qml
signal opened
```
Add a Go callback mechanism if MIQT custom signal declaration
is inconvenient:
```go
func (d *RDialog) OnOpened(fn func())
```
Call it from `OpenFor`.
Do not invent a new signal framework just for this component.
### 3. Footer alignment is not implemented
The current comment says the footer is intended to
right-align:
```go
// QML footer was RightToLeft
```
but the code only adds the layout:
```go
box.AddLayout(d.FooterLayout.QLayout)
```
There is no stretch or direction change.
Add a stretch before footer buttons, or use the MIQT
equivalent of right-to-left layout direction.
Acceptance test:
```text
Cancel and OK appear on the right side of the dialog.
```
### 4. No `maxHeight`
QML exposes:
```qml
property int maxHeight
```
and calculates:
```text
height = min(maxHeight, intrinsicHeight)
```
Add:
```go
func (d *RDialog) SetMaxHeight(int)
```
At minimum, enforce the maximum height. If content can exceed
it, decide whether the body needs a scroll area.
### 5. No Escape handling
The QML dialog closes on Escape.
Verify Qt’s default `QDialog` behavior. If it is not reliable
through MIQT, add an explicit shortcut or key handler.
### 6. No owner lifecycle handling
QML closes the dialog when its owner closes and removes the
overlay when the dialog is hidden or destroyed.
The Go version should ensure:
- closing the owner closes the dialog
- hiding the dialog removes any overlay
- destroying the dialog cleans up overlay state
- reopening the dialog does not create duplicate overlays
### 7. Overlay behavior
The QML version uses an overlay on the owner window.
For Qt Widgets, the simplest equivalent is a child `QWidget`
covering the owner:
```text
parent = owner
geometry = owner.rect()
background = semi-transparent dark color
visible while dialog is open
raise overlay before showing dialog
```
If keeping this inside `RDialog` becomes too invasive, expose
an optional overlay controller in `ui/` and let the future
page/window layer provide it.
Do not implement page-specific overlay logic in this phase.
### 8. Body and footer extensibility
The current public fields are useful:
```go
BodyLayout
FooterLayout
```
Keep them for now. Add helper methods only if they reduce
repetitive caller code:
```go
func (d *RDialog) AddBodyWidget(*qt.QWidget)
func (d *RDialog) AddFooterWidget(*qt.QWidget)
```
Do not hide the layouts behind a factory or builder.
---
# 6. Theme contract
The theme lives outside `ui/` in:
```text
theme/theme.go
```
It is already ported enough for the current widget scaffold,
but UI parity depends on these values:
```text
colors
text sizes
radii
spacing
touch target
icon sizes
```
For this UI-only task:
- do not redesign the theme
- do not move theme constants into `ui/`
- do not duplicate theme values in each component
- only add a theme value if a QML behavior genuinely requires
it
The current QML and Go theme values should remain visually
consistent.
---
# 7. Tests to add under `ui/`
Add tests in:
```text
ui/icon_test.go
ui/rbutton_test.go
ui/rswitch_test.go
ui/rtextfield_test.go
ui/rdialog_test.go
```
Use one offscreen Qt application setup for widget tests. The
exact MIQT application initialization must follow the
package’s conventions.
## Icon tests
Verify:
- invalid path
- empty path
- invalid size
- valid icon
- tinted pixmap is non-null
- resource lookup works from the test working directory
## Button tests
Verify:
- every variant constructs
- Ghost is transparent
- Link behavior is correct
- outlined state has a border
- outlined hover inverts foreground and icon
- icon-only sizing
- custom icon size
- tooltip assignment
- disabled state
- clicked callback
- corner radius setting
##
…[165 characters hidden from view — full text kept in
history]…
active/inactive colors
- paint event does not crash
- animation reaches the target state
## Text field tests
Verify:
- text round trip
- placeholder
- label visibility
- error visibility
- error border
- error state survives icon changes
- prefix and suffix icons
- clearing icons
- password mode
- read-only
- maximum length
- text-changed callback
- focus proxy
- select-by-mouse behavior
## Dialog tests
Verify:
- minimum width
- body/footer layout availability
- footer right alignment
- `OpenFor`
- opened callback
- Escape closes
- maximum height
- owner close behavior
- overlay creation and cleanup, if implemented
---
# Recommended implementation order
1. **Fix icon/resource loading**
2. **Finish `RButton`**
3. **Finish `RTextField`**
4. **Finish `RSwitch`**
5. **Finish `RDialog`**
6. **Add UI tests**
7. **Run the existing component demo**
8. **Only then start porting `rapid/qml/components`**
That order is deliberate. Buttons and text fields are used
almost everywhere in the QML component layer. Dialog behavior
depends on the button and field contracts.
---
# Definition of done for the `ui/` phase
The UI phase is complete when:
- all four QML reusable components have documented Go
equivalents
- current `main.go` continues to compile without unnecessary
API churn
- icons load from a packaged-safe source
- button variants match QML, including Ghost, Link, outlined
hover, icon-only, and tooltips
- switch matches size, colors, animation, interaction, and
keyboard behavior
- text field supports the properties needed by the future Go
UI
- text-field errors survive icon changes
- dialog supports owner-aware opening, Escape, footer
alignment, max height, and cleanup
- every component has at least one automated test
- the component demo can be run without backend code
- no `rapid/qml/pages` or backend files are modified during
this phase

Port only the reusable UI components from the PySide6/QML
project to MIQT Qt6 Widgets.
Scope:
- ui/icon.go
- ui/rbutton.go
- ui/rswitch.go
- ui/rtextfield.go
- ui/rdialog.go
- add ui/*_test.go tests as needed
Do not modify:
- rapid/backend
- rapid/qml/pages
- rapid/qml/components
- main.go unless required only to preserve compilation
- theme/theme.go unless a missing UI token is unavoidable
Use these QML files as the behavior specification:
- rapid/qml/ui/RButton.qml
- rapid/qml/ui/RSwitch.qml
- rapid/qml/ui/RTextField.qml
- rapid/qml/ui/RDialog.qml
Preserve the existing Go constructors where practical:
- NewRButton(text, variant, outlined)
- NewRSwitch(checked)
- NewRTextField()
- NewRDialog(owner)
Required parity:
- RButton: all variants, Ghost transparency, Link mode,
outlined hover inversion, icon-only mode, icon size, tooltip,
corner radius, disabled styling, touch-target sizing
- RSwitch: 36x20 dimensions, active/inactive colors, one-pixel
 border, animated knob, mouse and keyboard interaction,
disabled behavior
- RTextField: label, error, text, placeholder, password/echo
mode, read-only, max length, text changed, select-by-mouse,
prefix/suffix icons, configurable icon size/color, focus
proxy, and error state preservation when icons change
- RDialog: body/footer layouts, right-aligned footer,
OpenFor(owner), opened callback, Escape close, max height,
owner lifecycle cleanup, and overlay cleanup
Important known bug:
RTextField.setSideIcon currently calls handleError("") and
clears the error border. Preserve the current error state
instead.
Resource requirement:
The current IconPath uses canonical assets/icons while PySide
uses qrc resources from its compatibility copy at
rapid/qml/icons. Make MIQT icon loading reliable for tests and
packaged binaries. Do not add a new icon dependency.
Validation:
- gofmt all changed Go files
- go test ./...
- run the component demo with offscreen Qt if available
- add tests for every behavior listed above
- report any MIQT API that cannot be verified rather than
inventing a wrapper
- keep the diff limited to the UI phase
