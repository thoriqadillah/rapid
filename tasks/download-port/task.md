# Download UI port: Python/QML → Go + miqt

## Goal

Port all download-related UI (page + components + everything the page touches)
from `rapid/qml/` + `rapid/backend/download/service.py` (QML-facing surface only)
to `widget/` + `services/download/` in idiomatic Go + miqt.

Two phases:

- **Phase 1 — UI port, no service wiring.** Build the widgets with fake/seed data.
  Design the service boundary now so Phase 2 and full wiring later are trivial.
- **Phase 2 — minimal read-only wiring.** Create `services/download/service.go`
  (new, read-only) on top of the existing `store.go` / `transform.go` / `api` /
  `downloader` / `db` implementation. Wire only **sorting, filtering, and
  displaying download items**. Full action wiring (pause/resume/stop/delete/
  new-download/resolve/etc.) comes later.

Idiom rule: **prefer idiomatic Go + miqt over 1:1 QML translation**. QML
declarative tricks (Loaders, attached properties, Canvas, ListModel roles,
proxy models) get replaced by the cheapest QWidget equivalent that already
exists in this repo.

## Source inventory (what "everything related" means)

### QML page (1)

| File | Role |
|---|---|
| `rapid/qml/pages/DownloadPage.qml` (306 lines) | Page root. Owns `type` (category filter), search wiring, single-expand state (`expandedGid`), context-menu state (`contextMenuOpen/Gid`), empty state, table shell (header row + list + rounded-clip container), `DeleteConfirmationDialog` + `DownloadItemContextMenu` instances, `DownloadService.downloadName/delete` calls, `DownloadFilter.setCategory/setSearch` calls, `DownloadDialog.openFor/setUrl` on add, `NotificationService` on completed/failed |

### QML components (12, all under `rapid/qml/components/download/`)

| File | Port? | Notes |
|---|---|---|
| `DownloadLayout.qml` (118) | **mostly done** — `widget/components/downloads/layout.go` has sidebar sections but **no counts** | needs count badges (`countFor`) |
| `DownloadItem.qml` (298) | **port** — the row widget | core of Phase 1 |
| `DownloadItemDetail.qml` (248) | **port** — expandable detail under the row | Phase 1 visual only; buttons stubbed (see §5) |
| `DownloadItemContextMenu.qml` (51) | **port** — `QMenu` | Phase 1 with enablement matrix; actions stubbed |
| `DownloadDialog.qml` (323) | **stub only** — exists as placeholder `download_dialog.go` (29 lines, "Dialog body" + Cancel/OK) | full port is explicitly out of scope; keep stub, fix constructor shape for later |
| `AdvancedOptions.qml` (71) | **defer** (belongs to full DownloadDialog port) | — |
| `ResolverUri.qml` (292) | **defer** (belongs to full DownloadDialog port) | — |
| `KVEditor.qml` (273) | **defer** (belongs to full DownloadDialog port) | — |
| `ClipboardBanner.qml` (62) | **port (small)** — banner above table, `visible = Clipboard.url != ""` | Phase 1 static w/ callback; live clipboard wiring later |
| `DeleteConfirmationDialog.qml` (72) | **port (small)** — `RDialog` + From-disk switch (default checked) | Phase 1 functional UI, confirm callback stubbed |
| `QuitConfirmationDialog.qml` (86) | **defer** — owned by `main.py` quit flow, not the page | note in plan, do not build |
| `DownloadSpeedSample.qml` (80) | **port (small)** — sparkline in detail | Phase 1 static paint from passed samples; live `speedHistory` in Phase 2 |
| `DownloadLayout.qml` sidebar counts | **port** — `DownloadService.counts` badges | part of layout extension |

### Supporting QML (do not port, already ported in Go)

- `components/Layout.qml` → `widget/components/layout.go` (done)
- `components/Header.qml` → `widget/components/header.go` (done: menu/search/New)
- `components/Sidebar*.qml` → `widget/components/sidebar*.go` (done)
- `ui/R*.qml` → `widget/ui/rbutton.go, rtextfield.go, rdialog.go, rswitch.go, icon.go` (done)
- `Theme.qml` → `widget/theme/theme.go` (done)

### Python backend surface the UI touches (for boundary design, not to port)

- `rapid/backend/download/service.py`: `DownloadService` (`QAbstractListModel`, roles = all `Download` fields + synthetic `progress`) + `DownloadFilterProxy` (`setCategory/setSearch`, newest-first order preserved, no sorting override). In Go there is **no model/role layer and no `asDict`**: `asDict` existed only to serialize Python dataclasses for QML. The Go port passes typed `api.Download` structs directly — do not port `asDict`/`fromPayload` shapes.
- `rapid/backend/download/models.py`: `Download / DownloadFile / FileUri / ResolvedUrl / SpeedSample` (+ `progress` property). Already ported as `services/download/api/api.go` (`Download.Progress()`); no dict conversion needed.
- `rapid/main.py::downloader_service()`: registers `DownloadService` + `DownloadFilter` context props; quit flow reads `activeCount`/`activeDownloads()`.
- `rapid/backend/download/{store,aria2_downloader,downloader}.py` map to existing Go `services/download/{store,downloader}/...` — reuse, do not reimplement.

## Details & nuances checklist (do not lose these in the port)

### Data model / roles

1. Row fields (from model roles + `DownloadItem` required props): `gid, status, category, resolved{title,filename,url,mimeType,category,size,dir,headers,cookies,checked}, files[{path}], totalLength, completedLength, downloadSpeed, errorMessage`. Plus computed `progress`.
2. `category` is a free string; unknown/empty renders as `unknown` (`lib.Category.String()` already does this; `theme.CategoryColor` defaults to `ColorCategoryUnknown`).
3. Statuses observed in code/seed: `active, waiting, paused, complete, error, removed`. Only `active` drives speed samples + active counts; `complete → downloadCompleted`, `error → downloadFailed`. Everything else passes through opaquely.
4. Newest-first order. Both Python (`order_by created_at desc`, `insert(0)`) and Go store (`Order("created_at DESC")`) agree — the list widget must **not** re-sort by name/progress; filtering preserves insertion order.
5. `resolved` may be nil (Go: `toResolvedModel` returns nil on empty URL). Every display helper must nil-guard.

### Derived display values (exact QML semantics — keep them)

6. `displayName = resolved.title || resolved.filename || basename(files[0].path) || gid`. (`DownloadItem.qml` + `service.downloadName` agree; Python service falls back to `files[0].path` basename, QML additionally tries `resolved.title || filename` — unify in one Go helper.)
7. `fileDir = dirname(files[0].path) || ""`, empty renders as `"—"`.
8. `percent = total>0 ? min(1, completed/total) : 0`; label `"%1%".arg(round(percent*100))` with **fixed width** for `"100%"` (use `QFontMetrics` or fixed-width label so the column doesn't jitter).
9. `speedText`: `"—"` when `status==error`, else `formatSize(speed)+"/s"` when `speed>0`, else `"—"`.
10. `sizeText`: `formatSize(done)+" / "+formatSize(total)`, `"—"` when total falsy. `formatSize`: falsy → `"—"`; units B/KB/MB/GB/TB ÷1024; 0 decimals when `value>=100` or unit is B, else 1 decimal. (Python `formatSize` slot; no Go equivalent yet — new `lib.FormatSize` or `services/download.FormatSize`, shared by row + detail.)
11. `estText`: `remaining<=0 || speed<=0 → "—"`; `<60s → "%1s"`, `<60m → "%1m"`, else `"%1h %2m"`. **QML bug to fix in port:** ETA column uses `sizeColumnWidth` for all three constraints instead of `etaColumnWidth` — use the ETA width in Go.
12. `statusText`: `removed → "Stopped"`, else capitalized status. `canPause = active||waiting`, `canResume = paused`.

### Table shell (`DownloadPage.qml`)

13. Column widths: name min 100 (stretch), progress 300, speed/size/ETA 120 each. Header labels: Name / Progress / Speed / Size / Estimation, muted 14px. Icon spacer column (`iconXs` = 10px) aligns rows with header.
14. Container: rounded (`radiusSm`) surface panel with 2px `colorSurface` border, clipped content (QML uses mask hack; in Go a styled `QWidget` + `WA_StyledBackground` is enough — no mask).
15. Header: surface background, only **top** corners rounded (QML overlays a square patch; in Go just style the header widget + panel and accept square inner corners, or round the whole panel — do not replicate the patch).
16. Page margins: `spacingPage*` → current Go views use `SpacingXl` all around; keep consistent with `views/download.go` + `settings.go` (do not invent new page tokens).
17. Row zebra: `highlighted || expanded || (hovered && !menuOpen) → colorSurface`, else odd rows `rgba(surface,0.25)`, even transparent. Selected/hover beats zebra; open context menu suppresses hover.
18. Hairline divider under each row: 1px `colorBorder` @ 0.4 opacity.
19. Empty state: centered disabled icon (`MdiLightFormatAlignBottom`, `iconXl`=48, muted) + `"No downloads yet"` muted text, visible iff `count==0`.
20. List behavior: clip, `StopAtBounds` (no overscroll bounce). QML list add/remove transitions (150ms fade+slide: add `opacity 0→1 + x -spacingXl→0 OutCubic`, remove `opacity→0 + x→spacingXl InCubic` in parallel) — **keep in Go** via `QPropertyAnimation` on `pos` + `QGraphicsOpacityEffect` opacity (same pattern as `Sidebar.animate` with `QPropertyAnimation2`). Animate only the affected row widget on insert/remove, not the full rebuild.
21. Single-expand: `expandedGid`, toggling one row collapses the previous; **changing category or search collapses** (`expandedGid=""`).

### Row interactions (`DownloadItem.qml`)

22. Left-click toggles expand; right-click opens context menu with `{gid,status,fileDir,resolved,displayName}` payload. In Go: `OnMousePressEvent` checking `event.Button()`, or `SetContextMenuPolicy(Qt__CustomContextMenu)` + `OnCustomContextMenuRequested` — prefer the latter (idiomatic, handles keyboard menu key too).
23. Progress bar: h=10, pill radius (h/2), 1px border `colorBorder` on `colorSurface` track; fill width `trackWidth*percent`, color `colorDanger` when error else `theme.CategoryColor(category)`.
24. Category dot: `MdiSquareRounded.svg` at `iconXs` tinted by `theme.CategoryColor(category)` (already the sidebar pattern — reuse `ui.TintedPixmap`).
25. Text elide: name `ElideRight`, file location in detail `ElideMiddle`, URL/meta `ElideMiddle`. Set `Qt__ElideRight/Middle` + `SetWordWrap(false)` on the `QLabel`s.
26. Detail expand/collapse animation (QML: `detailHeight` 150ms OutCubic + detail `opacity`/`y -implicitHeight→0` 250ms OutCubic) — **keep in Go**: animate the detail container `maximumHeight 0↔contentHeight` (150–250ms OutCubic via `QPropertyAnimation2`, same pattern as `Sidebar.animate`) + `QGraphicsOpacityEffect` opacity fade. Instant show/hide is not acceptable; reuse one small `animateHeight(widget, from, to)` helper for this and the AdvancedOptions-style wrappers later.

### Detail (`DownloadItemDetail.qml`)

27. Rows (label min-width 100, muted + value): `ResolverUri` (readonly) / speed sparkline / Progress (label+bar+%) / File location / MIME type (`resolved.mimeType || "—"`) / Status / Error (`visible = errorMessage && status==error`, `colorDanger`, wrap) / Actions.
28. Actions: Pause↔Resume toggle (`enabled = canPause||canResume`, icon Play/Pause), Stop (`enabled = canPause`, `MdiLightStop`), spacer, Delete icon-only danger (`MaterialSymbolsLightDeleteForeverOutlineSharp`, tooltip `"Remove the file forever"` → opens `DeleteConfirmationDialog`).
29. `DownloadSpeedSample`: 120px tall surface panel, samples from `speedHistory(gid)` (≤60, chronological), area fill `rgba(categoryColor,0.15)` + 2px stroke, flat baseline when empty, HiDPI-aware (`2*dpr` → use `devicePixelRatio`). Refresh on `downloadChanged(gid)`. Phase 1: pure widget taking `samples []int64 + category`; Phase 2 feeds it from the store.

### Context menu (`DownloadItemContextMenu.qml`) — exact enablement matrix

30. Pause `enabled = active||waiting`; Resume `enabled = paused`; Stop `enabled = active||waiting`; **Delete `enabled = complete||error`** (note: detail-view delete button has no such guard — keep both behaviors as-is); separator; Copy URL `enabled = resolved && resolved.url` → `Clipboard.copy(url)`; Go to file `enabled = fileDir != ""` → `Qt.openUrlExternally("file://"+fileDir)` (Go: `qt.QDesktopServices_OpenUrl`).
31. After delete-from-menu, QML opens the delete confirm dialog (menu closes first). After menu close, highlight resets (`contextMenuOpen=false, contextMenuGid=""`).

### Sidebar (`DownloadLayout.qml` → `layout.go` extension)

32. Sections: LIBRARY = [All downloads (`MdiLightDownload.svg`, `colorText`)]; CATEGORIES (topMargin `spacingMd`) = 7 items with `MdiSquareRounded.svg` tinted per category + `categoryItem:true` (renders icon at `iconXs` not `iconSm` — `SetCategoryItem` already handles this); stretch; Settings (`MdiLightSettings.svg` → `navigation.Push(RouteSettings)`).
33. Counts: `countFor(key) = counts[key] ?? 0`, **hide when 0** (`SetCount("")` hides — `SidebarItem` already supports this). Keys: `all` + each category string. Counts update on `countsChanged` (Phase 2: service callback; Phase 1: static/seed).
34. `settings` destination routes via Navigation instead of setting `type` (all other destinations set the category filter; `settings` pushes the route — see next point).

### Filtering wiring (`DownloadPage.qml` logic to preserve)

35. `destinationSelected`: `settings → Navigation.push`, else `type=destination → DownloadFilter.setCategory(type)` + collapse detail. Search: `DownloadFilter.setSearch(text)` + collapse detail. In Go both funnel into one `service.SetCategory/SetSearch → list.Refresh()` path.
36. Search matching (Python `filterAcceptsRow` — replicate in Go service): category = exact match (empty = all); search = lowercase substring over `resolved.title || resolved.filename`, else `basename(files[0].path)`; empty search matches all.

### Dialogs / banner (small ports)

37. `DeleteConfirmationDialog`: title `"Delete download"`, body `"Remove %1 forever?"` (displayName or `"this download"`), From-disk row (`RSwitch` **checked by default** + label), footer Cancel / Delete-danger. Return/Enter confirms. Signal `deleteConfirmed(deleteFromDisk)`. Port as `NewDeleteConfirmationDialog(parent, displayName, onConfirm func(bool))`.
38. `ClipboardBanner`: visible iff `Clipboard.url != ""`; h=`touchTarget`; `rgba(colorInfo,0.15)` bg + `radiusSm`; paste icon (info, sm) + `"There is one url in the clipboard"` (info, 14px) + Download link-button (info) + dismiss icon-button (clears clipboard url). Signal `download`. Port as small widget with `SetVisible` + `OnDownload` callback; live clipboard later.
39. `DownloadDialog` stub: keep `NewDownloadDialog(parent)` signature but change to the **deps-ready shape** below; full 323-line port (debounced 300ms resolve, dry-run skeleton rows, `pickFolder` zenity/kdialog→`QFileDialog` fallback, headers/cookies `KVEditor`, per-URI edit, footer enablement `url!="" && noErrors && uris>0 && !fetching && !editing`, Esc/Return shortcuts) is **explicitly deferred**.
40. `QuitConfirmationDialog` (active-download quit guard, `activeNames[]`, HTML list) belongs to the app quit flow in `main.go`, not the page — **defer** with the tray/quit wiring.

### Known QML bugs — fix, don't port

41. ETA column width copy-paste (`sizeColumnWidth` ×3) → use ETA width.
42. `KVEditor` JSON error label has no `text:` binding (error never shows) → bind when that dialog is eventually ported.
43. Python `_getCategory` can hit unbound `guessMimeType` when filename is falsy; Go `getCategory` already guards (`mimeType=="" || guess=="" → unknown`). Keep Go behavior.

## Idiomatic Go + miqt design (not 1:1)

### Widget pattern (follow existing repo idioms)

- Each component = `struct` embedding `*qt.QWidget` (or `*qt.QDialog`/`*qt.QMenu` for dialogs/menus), constructor `NewXxx(parent *qt.QWidget, ...) *Xxx`, callbacks as `[]func(...)` + `OnXxx(fn)` methods. Never subclass Qt models/delegates.
- Layouts: `qt.NewQVBoxLayout2/QHBoxLayout2/QGridLayout2`, `SetContentsMargins/SetSpacing` with `theme.Spacing*`, `AddStretch` spacers. Sizes from `theme.TouchTarget/IconXs..Xl/RadiusSm/TextSize`.
- Colors: `theme.CssColor(...)` stylesheets; `theme.CategoryColor(lib.Category)` for the dot + progress fill; `WA_StyledBackground` where stylesheets must paint.
- Icons: `ui.IconPath("MdiLightPlus.svg")` + `ui.TintedPixmap`/`ui.LoadIcon`; `RButton.SetIconSource/SetIconOnly/SetTooltip`; sizes via `SetIconSize`.
- Primitives to reuse (do not rebuild): `ui.NewRButton` (+variants `Primary/Danger/Ghost/Info/Secondary/Link`), `ui.NewRTextField` (`SetLabel/SetError/SetPrefixIcon/OnTextChanged`), `ui.NewRSwitch`, `ui.NewRDialog` (`AddBodyWidget/AddFooterWidget/Open/OnOpened`), `components.Layout/Header/Sidebar*`.

### Key non-1:1 decisions

| QML | Go idiom | Why |
|---|---|---|
| `QAbstractListModel` + roles + `DownloadFilterProxy` | Plain `[]api.Download` in service + `Filtered()` returning a slice; list widget rebuilds row widgets | No Qt model/view plumbing in miqt; list is small (tens of rows); matches repo's QWidget-composition style; sorting/filtering becomes testable pure Go |
| `ListView` delegate | `QScrollArea` + `QWidget` + `QVBoxLayout`, one `DownloadItem` widget per visible row, `DeleteLater` on rebuild | `QListView` + custom delegate in miqt is heavy and unidiomatic here; scroll-area-of-widgets supports expandable details naturally |
| QML transitions (list add/remove 150ms fade+slide; detail expand 150–250ms height/opacity/slide) | `QPropertyAnimation2` on `pos` / `minimumHeight`+`maximumHeight` + `QGraphicsOpacityEffect` opacity, `OutCubic`/`InCubic` easing, `QParallelAnimationGroup` where QML uses `Parallel` | Same mechanism as existing `Sidebar.animate` (`QPropertyAnimation2` + `QEasingCurve`); QWidget has no declarative `Behavior`/`Transition`, this is the Qt idiom |
| ResolverUri enter/exit (300ms) + AdvancedOptions wrapper (100ms) | Same `QPropertyAnimation` helper, applied when the full `DownloadDialog` is ported later | Deferred with the dialog, not re-decided then |
| `Canvas` sparkline | Custom `SpeedChart` widget (`QWidget` + `OnPaintEvent` + `QPainter`: fill path + stroke path, `devicePixelRatio` scaling) | Direct QWidget painting equivalent; no QML scene graph |
| `Repeater`/`Loader`/`ComponentBehavior` | `for` loops constructing widgets; `SetVisible` for detail/banner/empty-state | Go has no declarative repeaters; explicit loops are clearer |
| `Connections { target: DownloadService }` | `service.OnChanged(func(gids []string))` subscription + `QTimer` poll in the view | miqt signal bridging across threads is fragile; poll + callback keeps threading in one place (see §5) |
| QML context properties (`DownloadService`, `DownloadFilter`, `Clipboard`, `Navigation`) | Explicit `DownloadViewDeps` struct passed to `NewDownloadView` | Constructor injection replaces global QML context; makes "easy wiring later" concrete |
| `Qt.openUrlExternally("file://"+dir)` | `qt.QDesktopServices_OpenUrl(...)` | Platform equivalent |
| `QFileDialog/zenity/kdialog` pickFolder | Defer entirely (dialog stub); when needed use `qt.QFileDialog_GetExistingDirectory` | One call, no subprocess fallback chain in v1 |

### Proposed files

```text
widget/components/downloads/
  layout.go            # EXTEND: count badges via service callback (keep NewLayout shape)
  table_header.go      # NEW: header row (icon spacer + 5 labels, fixed widths)
  download_list.go     # NEW: scroll area + rows + empty state + expandedGid/menu state
  download_item.go     # NEW: row widget (dot, name, progress bar+%, speed/size/eta)
  download_item_detail.go # NEW: detail widget (meta rows, sparkline, actions-stubbed)
  context_menu.go      # NEW: QMenu builder with enablement matrix
  clipboard_banner.go  # NEW: banner widget
  delete_dialog.go     # NEW: RDialog-based confirm + From-disk switch
  speed_chart.go       # NEW: sparkline widget
  format.go            # NEW: FormatSize/DisplayName/FileDir/Progress/StatusText/EstText helpers
  download_dialog.go   # RESHAPE: keep stub, adopt deps-ready constructor
widget/views/
  download.go          # REWRITE: real page (layout + banner + panel + list), drop demo panel
services/download/
  service.go           # NEW (Phase 2): read-only service over bun DB (see §5)
lib/ (optional)
  format.go            # only if FormatSize belongs shared; else keep in downloads package
```

No new dependencies. No changes to `db/`, `services/download/{store,transform,api,downloader,rpc}/`, migrations, or theme tokens.

### Service boundary (design in Phase 1, implement read-only in Phase 2)

`db.DB()` is a process singleton — the service calls it directly instead of
taking a DB handle. No `bun.IDB` parameter, no stored DB field; when a
transaction is needed later, the service opens it via `db.DB().RunInTx(...)`
at the call site.

```go
// Phase 1: define + stub. Phase 2: implement read-only parts.
// Full actions (Pause/Resume/Stop/Delete/Download/Resolve/...) are method
// stubs returning zero values so UI compiles; each marked // TODO(full-wire).
type Service struct { /* mu, items []api.Download, category, search string, changed []func() */ }

func NewService() *Service
func (s *Service) All() []api.Download          // newest-first
func (s *Service) Filtered() []api.Download      // category + search (pure, testable)
func (s *Service) SetCategory(c string)          // "" = all; clears expanded in view
func (s *Service) SetSearch(q string)            // lower+trim; clears expanded in view
func (s *Service) Counts() map[string]int        // "all" + per-category
func (s *Service) Refresh(ctx context.Context) error // reload All() from store via db.DB()
func (s *Service) OnChanged(fn func())           // view subscribes; service calls on GUI thread (see below)
func (s *Service) SpeedHistory(ctx context.Context, gid string) ([]api.SpeedSample, error) // via db.DB()
```
// TODO(full-wire): Pause/Resume/Stop/Delete/Download/Resolve/PickFolder/ActiveCount/...
```

Pure helpers (no Qt, unit-testable): `DisplayName(d api.Download) string`,
`FileDir(d api.Download) string`, `FormatSize(n int64) string`,
`Progress(d) float64`, `SpeedText/SizeText/EstText/StatusText`, `Matches(d, category, search) bool`.

Threading: aria2 callbacks and store I/O must never touch widgets. Phase 2
rule: service does DB reads on a worker goroutine/`QTimer` tick
(`PollIntervalMs` from settings, default 1000ms) and emits `OnChanged` on the
GUI thread (via `qt.QTimer` single-shot / `QApplication.PostEvent`-style
marshal — pick whatever the codebase settles on; keep it behind `OnChanged`
so callers never care). View refresh = `list.SetItems(service.Filtered())` +
sidebar `SetCount(...)`.

View constructor becomes:

```go
type DownloadViewDeps struct {
    Navigation *app.Navigation
    Notifier   *notification.Service
    Service    *downloads.Service // nil in Phase 1 → seed/fake items
}
func NewDownloadView(parent *qt.QWidget, deps DownloadViewDeps) *qt.QWidget
```

`main.go` passes `Service: nil` in Phase 1, a real `NewService()` (it reads
`db.DB()` itself) in Phase 2. `NewDownloadDialog(parent, deps)` takes the same deps later so the
full dialog port needs no signature churn.

## Phase 1 plan — UI port without service wiring

1. `format.go`: pure helpers + unit test (`formatSize` table: 0→`"—"`, B/KB/MB/GB/TB boundaries, 0-vs-1 decimal rule; displayName/fileDir fallbacks; estText branches). No Qt needed — plain `go test`.
2. `table_header.go` + `speed_chart.go` (static paint from `[]int64`): smallest widgets first, verified visually via existing `qt_test.go` harness pattern.
3. `download_item.go` (row): dot + name + progress bar + % + speed/size/ETA labels; hover/zebra/highlight stylesheet; right-click → `OnContextMenu(api.Download)`; left-click → `OnToggle`. Progress: `QProgressBar` styled per-category if clean, else custom paint widget — pick whichever is less code after a 10-minute spike; per-row color changes must not leak stylesheets to siblings.
4. `download_item_detail.go`: meta rows + readonly resolved line + sparkline + action buttons that fire `OnPause/OnResume/OnStop/OnRemove` callbacks (view logs/TODOs them in Phase 1 — **no service calls**).
5. `context_menu.go` + `delete_dialog.go` + `clipboard_banner.go`: exact enablement matrix / default-checked switch / banner copy; actions route to the same stub callbacks as detail buttons.
6. `download_list.go`: owns items, `expandedGid`, menu highlight, empty state; `SetItems([]api.Download)` rebuilds rows (preserve `expandedGid` if still present) with the 150ms fade+slide add/remove animations intact; `OnToggle/OnContextMenu/OnRemove` forwarded to view.
7. `layout.go` extension: `SetCounts(map[string]int)` updating each `SidebarItem.SetCount` (hide zeros); keep `NewLayout(navigation)` signature.
8. `views/download.go` rewrite: real page — `downloads.NewLayout` + content margins `SpacingXl`, banner + rounded panel + `table_header` + `download_list`; `OnDestinationSelected` (settings→push else `SetCategory`), header search → `SetSearch`, header New → dialog stub `Open()`; drive everything from in-memory seed items (`services/download` seeder fixtures or 3 hand-written `api.Download`s covering active/complete/error/paused). **Delete the demo panel** (buttons/switch/textfield showcase) — it was scaffolding.
9. `download_dialog.go` reshape to `NewDownloadDialog(parent *qt.QWidget, deps ...)`-compatible stub (title "New download", min width 500) so Phase 1 call sites match Phase 2.

Acceptance (Phase 1): `go build ./...`, `go test ./widget/... ./lib/...` green;
app shows download page with seeded rows, working category sidebar (with
counts), working search filter, expand/collapse, context menu with correct
enablement, delete-confirm dialog, clipboard banner toggle — all without a DB.

## Phase 2 plan — minimal read-only wiring (sort/filter/display only)

Scope: **only** displaying, sorting (newest-first = store order), filtering
(category + search). Everything else stays stubbed.

1. Create `services/download/service.go` per the boundary above, implemented
   read-only: `NewService()` with no DB arg (calls `db.DB()` directly);
   `Refresh(ctx)` re-reads via existing `getAllDownloads`;
   `Filtered()` applies `Matches` (exact replicate of
   Python `filterAcceptsRow`); `Counts()` aggregates; `SpeedHistory` delegates
   to existing `getSpeedHistory(ctx, db, gid, 60)`.
2. Wire `views/download.go`: construct service in `main.go` (`download.NewService()`, `Refresh` once + `QTimer` every `PollIntervalMs`), pass into `NewDownloadView` deps; view subscribes `OnChanged` → `list.SetItems(Filtered())` + `layout.SetCounts(Counts())`; sidebar/search callbacks call `SetCategory/SetSearch` (view clears `expandedGid` via `list.Collapse()`); detail sparkline fed by `SpeedHistory`.
3. Seed data: reuse `DownloadSeeder` via `cmd/seed` for manual verification (already exists — no new fixtures).
4. Tests: service filter/count/match unit tests over `OpenMemory` + seeder data (pattern exists in `store_test.go`); widget test for list filtering if the `qt_test.go` harness allows, else manual.

Explicit non-goals (later full wiring): pause/resume/stop/delete-from-disk
actions, `DownloadDialog` resolve/download flow, `pickFolder`, clipboard
monitoring, `speedHistory` live refresh + `downloadChanged` push, polling +
aria2 `Listen/Refresh`, notifications (`downloadCompleted/Failed`), tray badge
(`activeCount`), quit guard, `removed`-status handling, plugin/browser resolve.
Each has a `// TODO(full-wire)` stub so the compiler guides the next task.

Acceptance (Phase 2): app against a real DB shows seeded downloads, sidebar
counts hide zeros, category + search filter correctly (including
`title→filename→basename` fallback order), expand/detail/sparkline render real
data, newest-first order matches old app; no action button performs I/O yet.

## Verification

- `go build ./...` + `go vet ./...`
- `go test ./services/download/... ./widget/... ./lib/...`
- Manual: `go run ./cmd/seed` (if applicable) then `go run .`, check page:
  all-downloads + each category, search hit/miss, empty state, expand one row,
  right-click menu enablement per status, delete dialog text + switch default,
  banner show/hide.
- Regression eyeball vs old QML app for column widths, dot colors, `"—"`
  placeholders, `%` alignment, error red states.
