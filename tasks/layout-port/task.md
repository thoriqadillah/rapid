# Layout/components port to MIQT Qt Widgets

## Objective

Port every reusable component currently under:

```text
rapid/qml/components/
```

except these explicitly excluded subtrees:

```text
rapid/qml/components/download/
rapid/qml/components/settings/
```

The eligible current scope is:

```text
rapid/qml/components/Header.qml
rapid/qml/components/Layout.qml
rapid/qml/components/Sidebar.qml
rapid/qml/components/SidebarItem.qml
rapid/qml/components/SidebarLabel.qml
rapid/qml/components/SidebarSection.qml
```

Port the eligible components into Go under the current `ui/` package using
MIQT and Qt Widgets. Do not port the excluded `download` or `settings`
components in this task. Do not port pages, `Main.qml`, notification popups,
or backend services unless a very small adapter is required to prove a
component contract.

The target is behavioral and visual parity where Qt Widgets can provide it.
When an exact QML behavior is not a good fit for Qt Widgets, use the closest
native Qt Widget idiom and document the deviation. Do not reproduce QML
`Loader`, `Repeater`, anchors, property bindings, or JavaScript object patterns
literally in Go. Prefer ordinary Go structs, explicit constructors, layouts,
models, callbacks, and Qt size policies.

Reuse the existing reusable widgets in `ui/`. Do not create a second button,
switch, text-field, dialog, or icon system.

## Required operating rules

1. Read this task completely before editing code.
2. Inspect the current working tree before assuming that `HEAD` or older task
   documents describe the current implementation. Uncommitted changes exist.
3. Preserve unrelated user changes. Do not reset, checkout, clean, or rewrite
   files outside the port scope.
4. Use Qt Widgets and MIQT Go idioms. Do not add a QML runtime or copy QML into
   strings embedded in Go.
5. Keep existing public APIs in `ui/` compatible unless a new constructor is
   genuinely required. Prefer setters, callbacks, typed models, and helpers.
6. Never override virtual methods on arbitrary caller-owned Qt objects. MIQT
   only permits virtual overrides for directly constructed Go-owned types. Do
   not call `OnResizeEvent`, `OnCloseEvent`, or similar methods on an owner
   supplied to a component.
7. Do not use wrapper widgets merely to guess at layout geometry. Measure size
   hints, minimum/maximum sizes, policies, margins, and layout contents first.
8. Preserve the intentional compact button styling. Do not restore large QML
   horizontal button padding to solve a layout problem. Fix layout and policy
   separately.
9. Use the canonical MIQT icon/resource path:

   ```text
   assets/icons/
   ```

   and Qt resource URLs such as `:/icons/MdiLightPlus.svg` where appropriate.
   Do not introduce another filesystem icon root. `rapid/qml/icons/` remains
   only as a synchronized PySide6 compatibility copy.
10. Do not build or launch the application unless explicitly requested.
    Use focused offscreen tests, `go vet`, and `git diff --check`.
11. Do not claim visual parity from compilation alone. Add geometry and
    interaction assertions where behavior matters.
12. Record important nuances, deviations, generated artifacts, screenshots,
    MIQT API discoveries, and unresolved issues as described below.

## Current starting point

### Existing Go widgets

```text
ui/icon.go
ui/rbutton.go
ui/rswitch.go
ui/rtextfield.go
ui/rdialog.go
```

Tests and generated resource artifacts may also be present:

```text
ui/*_test.go
ui/resources_generated.go
ui/resources.rcc
```

Inspect the current constructors and setters before designing composites.

### Resource workflow

The embedded MIQT icon resource is generated through:

```bash
task build:icon
```

`ui/icon.go` contains:

```go
//go:generate task build:icon
```

The task generates:

```text
ui/resources_generated.go
ui/resources.rcc
```

The manifest is `assets/icons/icons.qrc`. Use `IconPath` when possible:

```text
IconPath("MdiLightMenu.svg")
IconPath("MdiLightMagnify.svg")
IconPath("MdiLightPlus.svg")
```

Do not add an icon dependency or move the canonical assets.

### Theme workflow

Use `rapid/widget/theme`. Do not duplicate QML theme constants in each component.
The QML reference is `rapid/qml/Theme.qml`; the Go theme may be palette-derived
rather than byte-for-byte equal. That is acceptable when documented and
visually coherent. Preserve existing `theme.Spacing*`, `theme.TextSize*`,
`theme.Icon*`, `theme.TouchTarget`, radius, and color APIs.

### Focused validation

Use:

```bash
env -u QT_QPA_PLATFORM go test -count=1 ./ui
go vet ./ui
git diff --check
```

Include new component packages in the test command. Do not run `go build`,
`task build`, or `task dev` in the normal implementation loop.

# QML contracts to port

## 1. Header.qml

Source: `rapid/qml/components/Header.qml`.

QML public contract:

```qml
signal addClicked()
signal menuClicked()
property alias searchText: searchField.text
property int preferredSearchWidth: 350
```

Recommended Go shape:

```go
type Header struct {
    *qt.QWidget
    SearchField *ui.RTextField
    MenuButton  *ui.RButton
    AddButton   *ui.RButton
}

func NewHeader() *Header
func (h *Header) OnAddClicked(fn func())
func (h *Header) OnMenuClicked(fn func())
func (h *Header) SetSearchText(text string)
func (h *Header) SearchText() string
func (h *Header) SetPreferredSearchWidth(width int)
func (h *Header) PreferredSearchWidth() int
```

The exact exported fields are optional, but callers must connect both actions
and access search text without reaching into private implementation details.

The header is a full-width widget with background `Theme.colorBackground`,
implicit row height plus two `spacingSm` margins, a one-pixel bottom boundary,
and a horizontal row with page margins and `spacingSm` spacing. Its order is:

```text
[menu button] [stretch] [search field, preferred/max width 350] [New button]
```

Use a `QHBoxLayout` and a real stretch. The search field has minimum width 0,
preferred/max width 350, and minimum height at least `Theme.TouchTarget`.
Menu and New buttons must not horizontally expand.

Child mapping:

```text
menu:   RButton GhostVariant, :/icons/MdiLightMenu.svg, icon-only
search: RTextField, prefix :/icons/MdiLightMagnify.svg, placeholder Search...
add:    RButton PrimaryVariant, text New, :/icons/MdiLightPlus.svg
```

All three controls meet the touch target. Menu and New callbacks fire once per
click. Search text round-trips through setters/getters and remains editable.

Required tests:

- expected child controls exist
- placeholder is `Search...`
- search policy is minimum 0 / maximum 350 by default
- buttons meet the touch target and are not horizontally expanding
- menu/add callbacks fire exactly once
- search text and preferred width round-trip
- focusing the search field is not stolen by the header

## 2. SidebarLabel.qml

Source: `rapid/qml/components/SidebarLabel.qml`.

This is a small text-only heading:

```qml
Text {
    color: Theme.colorTextMuted
    font.pixelSize: Theme.textSizeSm
    font.letterSpacing: 1
    text: ""
}
```

Implement it as a styled `QLabel`, not a custom hierarchy. Recommended API:

```go
func NewSidebarLabel(text string) *SidebarLabel
func (l *SidebarLabel) SetText(text string)
func (l *SidebarLabel) Text() string
```

Apply muted color, small text size, no frame/background, and approximately one
pixel letter spacing. Check generated MIQT APIs before using a font workaround.
If exact letter spacing is unavailable, preserve the small muted heading and
record the deviation in `tasks/layout-port/implementation-notes.md`.

Required tests: text round-trip, muted/small style, and no unexpected focus or
button behavior.

## 3. SidebarItem.qml

Source: `rapid/qml/components/SidebarItem.qml`.

QML contract:

```qml
signal activated()
required property string destination
required property string label
property string count: ""
property url iconSource: ""
property color iconColor: Theme.colorTextMuted
property bool categoryItem: false
property bool selected: false
implicitHeight: 36
```

Recommended API:

```go
type SidebarItem struct { *qt.QWidget }
func NewSidebarItem(destination, label string) *SidebarItem
func (i *SidebarItem) SetDestination(string)
func (i *SidebarItem) Destination() string
func (i *SidebarItem) SetLabel(string)
func (i *SidebarItem) Label() string
func (i *SidebarItem) SetCount(string)
func (i *SidebarItem) Count() string
func (i *SidebarItem) SetIconSource(string)
func (i *SidebarItem) SetIconColor(*qt.QColor)
func (i *SidebarItem) SetCategoryItem(bool)
func (i *SidebarItem) SetSelected(bool)
func (i *SidebarItem) Selected() bool
func (i *SidebarItem) OnActivated(func())
```

Keep destination as data. The item emits activation; the parent owns routing.

Geometry/style:

- minimum/preferred height 36 px
- parent-filling horizontal policy
- rounded background with `Theme.radiusSm`
- transparent idle background
- lighter surface background when selected or hovered
- full row clickable, pointing cursor, hover over the whole row
- row left/right margins `spacingSm`, internal spacing `spacingSm`
- icon slot width `iconSm`
- category icon size `iconXs`; normal icon size `iconSm`
- selected non-category icon uses normal text color; otherwise `iconColor`
- label fills remaining width and elides right, never wraps
- optional count visible only when non-empty, uses `textSizeSm`
- idle label/count muted, selected/hovered label/count normal text color
- icon is non-interactive and vertically centered

Do not copy the QML trick of using a disabled button as an icon. Use a QLabel,
icon child, or small paint path. Prefer a native button-like interaction for
keyboard accessibility; if a plain QWidget is used, document keyboard behavior.

Required tests: property round-trips, count visibility, selected/hover style,
category icon sizing, 36 px height, icon updates, one activation per click, and
bounded/elided long labels.

## 4. SidebarSection.qml

Source: `rapid/qml/components/SidebarSection.qml`.

QML accepts an array of JavaScript objects. Translate it to a typed Go model,
not `interface{}` maps:

```go
type SidebarItemData struct {
    Destination string
    Label string
    Count string
    IconSource string
    IconColor *qt.QColor
    CategoryItem bool
}
```

Recommended API:

```go
type SidebarSection struct { *qt.QWidget; ItemsLayout *qt.QVBoxLayout }
func NewSidebarSection() *SidebarSection
func (s *SidebarSection) SetHeading(string)
func (s *SidebarSection) Heading() string
func (s *SidebarSection) SetItems([]SidebarItemData)
func (s *SidebarSection) Items() []SidebarItemData
func (s *SidebarSection) SetCurrentDestination(string)
func (s *SidebarSection) CurrentDestination() string
func (s *SidebarSection) SetTopMargin(int)
func (s *SidebarSection) TopMargin() int
func (s *SidebarSection) OnActivated(func(string))
func (s *SidebarSection) Refresh()
```

The section is a vertical layout: optional heading followed by SidebarItems.
Heading is visible only when non-empty and has bottom margin `spacingXs`.
Items have `spacingXs`, fill section width, and selected state when their
destination equals current destination. Activation forwards the destination.

`SetItems` must replace only section-owned item widgets, remove stale items,
and not duplicate widgets or callbacks on repeated calls. Stable item reuse is
fine but not required for the first implementation.

Required tests: heading visibility, typed model creation/replacement, exact
selection, top margin, and destination forwarding.

## 5. Sidebar.qml

Source: `rapid/qml/components/Sidebar.qml`.

QML contract:

```qml
signal destinationSelected(string destination)
property string currentDestination: ""
property bool open: true
function activate(destination)
default property alias content: contentColumn.data
```

Recommended API:

```go
type Sidebar struct { *qt.QWidget; ContentLayout *qt.QVBoxLayout }
func NewSidebar() *Sidebar
func (s *Sidebar) SetCurrentDestination(string)
func (s *Sidebar) CurrentDestination() string
func (s *Sidebar) SetOpen(bool)
func (s *Sidebar) Open() bool
func (s *Sidebar) Activate(string)
func (s *Sidebar) AddContentWidget(*qt.QWidget)
func (s *Sidebar) OnDestinationSelected(func(string))
```

Use a QWidget/QFrame with a vertical content layout. Surface background,
implicit/open width 200, closed width target 0, clipping, and a one-pixel right
border are required. Content margins are top `spacingSm`, bottom
`spacingPageBottom`, left/right page margins, with `spacingXs` between items.

QML animates width for 200 ms with InOutQuad easing. Prefer
`QPropertyAnimation` on the Go-owned sidebar if the generated MIQT API is
available. A timer with elapsed-time interpolation is acceptable. Immediate
width changes are acceptable only as a documented native-Qt deviation.
Never animate a caller-owned parent through an event override.

`Activate` must set current destination before firing callbacks. `AddContent`
is the explicit Go equivalent of the QML default property slot; do not accept a
QML-like dynamic `Component`.

Required tests: default/open/closed widths, margins/spacing, destination
forwarding, state-before-callback, right border, clipping, and verified or
documented animation behavior.

## 6. Layout.qml

Source: `rapid/qml/components/Layout.qml`.

QML contract:

```qml
signal destinationSelected(string destination)
signal addClicked()
property Component headerContent: null
property Component sidebarContent: null
property Component defaultHeaderComponent: Component { Header {} }
property bool sidebarOpen: true
default property alias content: contentArea.data
readonly property alias header: headerLoader.item
readonly property alias sidebar: sidebarLoader.item
readonly property string searchText: header?.searchText ?? ''
```

Do not expose `Component` or `Loader` in Go. Use explicit composition:

```go
type Layout struct {
    *qt.QWidget
    HeaderWidget *Header
    SidebarWidget *Sidebar
    ContentWidget *qt.QWidget
    ContentLayout *qt.QVBoxLayout
}
func NewLayout() *Layout
func (l *Layout) SetHeader(*Header)
func (l *Layout) SetSidebar(*Sidebar)
func (l *Layout) SetSidebarOpen(bool)
func (l *Layout) SidebarOpen() bool
func (l *Layout) AddContentWidget(*qt.QWidget)
func (l *Layout) OnDestinationSelected(func(string))
func (l *Layout) OnAddClicked(func())
func (l *Layout) SearchText() string
```

A practical native arrangement is a `QGridLayout`:

```text
row 0, col 0: sidebar, spanning rows 0..1
row 0, col 1: header
row 1, col 1: content
```

Make row 1 and column 1 stretch. The sidebar stays in the left column and the
header/content occupy the right. Do not let regions overlap accidentally.

Forwarding rules:

- sidebar destination -> Layout destination callback
- header menu -> toggle sidebar open
- header add -> Layout add callback
- SearchText -> active header search text, or empty with no header

`SetHeader` and `SetSidebar` replace the region explicitly. Nil may mean use a
default component. Define ownership when replacing; do not double-parent or
delete caller-owned widgets unexpectedly. `AddContentWidget` is the explicit
content slot.

For the QML background focus MouseArea, expose `FocusContent()` or use a safe
content-container policy. Do not install a global event filter that consumes
child events without tests proving propagation.

Required tests: default regions, replacement without duplication, menu toggle,
destination/add forwarding, search forwarding, stretch factors, content
placement, and nil safety.

# Cross-component architecture

## Package organization

A reasonable implementation is:

```text
ui/header.go                 ui/header_test.go
ui/sidebar_label.go          ui/sidebar_label_test.go
ui/sidebar_item.go           ui/sidebar_item_test.go
ui/sidebar_section.go        ui/sidebar_section_test.go
ui/sidebar.go                ui/sidebar_test.go
ui/layout.go                 ui/layout_test.go
```

Use a shared model file only if needed. Do not create a package per tiny QML
component.

## Callback rules

- nil callback registration is a no-op
- callbacks run on the owning Qt thread
- callbacks fire once per user action
- state updates precede callbacks where QML does so
- construction does not invoke callbacks
- registration order is preserved
- no global event bus

## Ownership rules

Document which widget owns every child. Layout insertion gives Qt ownership;
callers retain Go references but must not separately delete installed widgets.
Replacement must specify whether old widgets are detached, hidden, or deleted.
Avoid double-parenting and do not manually free parent-owned Qt objects without
verified MIQT guidance.

## MIQT virtual-method hard constraint

MIQT reports:

```text
miqt: can only override virtual methods for directly constructed types
```

Safe patterns include native signals, direct property setters, layouts,
stylesheets, timers, and overrides only on directly constructed Go-owned
widgets. Unsafe patterns include owner resize/close overrides and event hooks
on a QMainWindow supplied by another layer. If parent tracking cannot be done
safely, expose an explicit refresh method and document it.

## Size policy and geometry rule

The project has already had footer/button width regressions caused by nested
layouts and guessed wrapper widgets. For this task:

- preserve compact button styling
- set deliberate policies on composite children
- inspect SizeHint, MinimumSize, MaximumSize, and SizePolicy
- use layout stretches for alignment, not button expansion
- do not solve layout geometry by restoring large horizontal padding
- add geometry tests for Header, SidebarItem, Sidebar, and Layout

# Theme and parity policy

Preserve these QML reference values when Go theme equivalents exist:

```text
spacingXs=4  spacingSm=8  spacingMd=12  spacingLg=16  spacingXl=24
radiusSm=4  textSize=14  textSizeSm=10
aiconXs=10  iconSm=18  iconMd=24  iconLg=32
touchTarget=36  sidebarWidth=200  sidebarItemHeight=36
headerSearchWidth=350  sidebarAnimation=200ms
```

(The `aiconXs` spelling above is only a documentation label; use the actual
`theme.IconXs` symbol.)

Accepted deviations include palette-derived Go colors versus fixed QML colors,
font metric differences, unavailable Qt Quick-only properties, and different
Loader/Repeater ownership semantics. Every deviation must record the QML
behavior, attempted Qt equivalent, limitation, chosen alternative, and test.
Do not silently lower fidelity.

# Testing plan

Use the existing offscreen setup in `ui/qt_test.go`; do not create a second
QApplication. Recommended checks:

```bash
env -u QT_QPA_PLATFORM go test -count=1 ./ui
go vet ./ui
git diff --check
```

If relevant, also run `env -u QT_QPA_PLATFORM go test -count=1 ./...`.
Do not run `go build`, `task build`, `task dev`, or `go run .` in the normal
loop for this task.

Required test categories:

- construction with QApplication and nil-safe optional inputs
- child/layout ownership and no duplicate insertion
- geometry and policies after `Show`, `AdjustSize`, `Resize`, and event
  processing in an offscreen setup
- Header menu/add callbacks and bounded search width
- SidebarItem hover/selection/click/category/count behavior
- SidebarSection model replacement and selection forwarding
- Sidebar open/closed target widths, clipping, and animation decision
- Layout replacement, region placement, stretch, forwarding, and focus policy
- embedded icon loading with `:/icons/...`

Assert relationships rather than fragile screen coordinates. Existing `ui/`
tests must continue to pass unless an expectation is demonstrably stale relative
to an explicit user decision; record any such change.

# Implementation phases

## Phase 0: baseline

1. Inspect the working tree and current `ui` APIs.
2. Run focused tests without editing.
3. Record pre-existing failures separately.
4. Confirm only the six eligible QML files are in scope.
5. Verify MIQT APIs for layouts, size policies, labels, animation, icons,
   callbacks, and keyboard interaction.

## Phase 1: leaf widgets/models

Implement and test `SidebarLabel`, `SidebarItem`, `SidebarItemData`, and
`SidebarSection` first.

## Phase 2: Sidebar

Implement content layout, destination state/callbacks, border, clipping, open
state, width targets, animation choice, and tests. Do not connect page routing.

## Phase 3: Header

Compose current RButton/RTextField widgets. Add search constraints, callbacks,
and tests. Preserve compact button sizing.

## Phase 4: Layout

Implement explicit sidebar/header/content regions, replacement APIs, signal
forwarding, sidebar toggle, search forwarding, and content insertion. Do not
port Main.qml or page routes.

## Phase 5: integration harness/review

Construct an offscreen fixture:

```text
Layout
  Header
  Sidebar
    SidebarSection
      SidebarItem(s)
  content label/widget
```

Assert geometry and callbacks. Review the diff for excluded-directory changes,
duplicate widgets, unsafe virtual overrides, page assumptions, hard-coded theme
values, and stale QML terminology.

# Required notes and artifacts

Create and maintain:

```text
tasks/layout-port/implementation-notes.md
```

Create it before implementation and record the baseline commands/results.
For each component record changed files, public API, direct QML mappings,
intentional deviations, ownership, size policies, and verified MIQT APIs.

For every deviation use:

```markdown
### Deviation: <short title>

QML behavior:

Attempted Qt Widget equivalent:

Why exact parity was not used:

Chosen implementation:

Validation:

Follow-up:
```

Also record generated files, resource regeneration commands, fixture names,
offscreen screenshots and paths (if any), temporary artifacts not committed,
and exact commands used. Keep an unresolved-work section listing parity gaps,
unverified MIQT behavior, deferred work, and user decisions. Do not commit
random screenshots, binaries, or logs without explaining their purpose.

# Acceptance criteria

- only the six eligible component files and required Go/UI support are changed
- `download/` and `settings/` remain untouched
- pages, Main.qml, and backend services are not silently ported
- every eligible QML component has a clear Go/MIQT equivalent
- public APIs use typed Go data and callbacks; no QML Components/Loaders leak
- current button, text-field, icon, and theme infrastructure is reused
- no unsafe virtual override is installed on caller-owned Qt objects
- Header order, search width, callbacks, and touch targets are correct
- Sidebar/Item/Section geometry, selection, clipping, icons, count, and routing
  callbacks are correct
- SidebarLabel is styled and documented if letter spacing differs
- Layout regions, replacement, stretch, forwarding, search, and focus policy
  are correct
- compact button sizing remains compact; no padding regression is introduced
- focused tests, `go vet`, and `git diff --check` pass
- geometry assertions cover the known width/size-policy failure class
- embedded resource paths are used where relevant
- `implementation-notes.md` is complete enough for another agent to continue

## Final report format

Finish with:

1. files added/changed
2. component-by-component summary
3. explicit QML-to-Qt deviations
4. generated artifacts
5. commands and pass/fail results
6. unresolved items and next steps

Do not report visual success based only on compilation. If geometry or
interaction was not measured, state that explicitly.
