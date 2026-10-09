# Rapid Go + MIQT port — code review & memory-growth report (merged)

Date: 2026-10-08 (merged 2026-10-09 with the independent `rapid-miqt` review)
Scope: the whole Go port (`widget/`, `services/`, `db/`, `lib/`, `main.go`,
`cmd/seed`) against the PySide6/QML originals in `rapid/`.

This document is both a review and a patch log. It merges two review passes:

- **Pass 1** (this repo's original review) — root-caused the growing memory
  usage; its fixes are already applied and tested in this working tree.
- **Pass 2** (the independent `rapid-miqt-review(1).md`) — a much broader
  correctness/security/perf pass whose items are merged below. **Every item has
  been re-verified against the current working tree** — several of its P0s are
  already fixed; the rest are listed with status + concrete fix. Snippets from
  pass 2 were written without a toolchain: treat them as designs, not
  guaranteed-to-compile code (anything marked *(verify)* needs a miqt name
  check). `stderrors` in snippets means stdlib `errors` (because
  `rapid/lib/errors` shadows it).

Status legend: **Fixed** · **Proposed** (verified still broken, fix below).

---

## 0. TL;DR — memory/leak work (pass 1, all Fixed)

| # | Finding | Severity | Status |
|---|---------|----------|--------|
| 1 | `ui.TintedPixmap` leaked ~45 KB native per call; download list re-renders every row on a 1 s poll | **Critical (root cause of rising RAM)** | **Fixed** |
| 2 | `theme.WithAlpha` allocated a new native `QColor` on every call | High | **Fixed** |
| 3 | Every `paint()` allocated `QPainter`/`QPen`/`QBrush`/`QPainterPath`/`QColor` with no release | High | **Fixed** |
| 4 | `Service.OnChanged` had no unsubscribe; view rebuilt on navigation | High | **Fixed** |
| 5 | `Service.Refresh` notified on every poll even when nothing changed | High | **Fixed** |
| 6 | `elidedLabel.apply` allocated a `QFontMetrics` on every resize/text change | Medium | **Fixed** |
| 7 | Row/detail/notification animation temporaries unfreed | Medium | **Fixed** |
| 8 | `Aria2Downloader.Refresh` read the listener map without the mutex (data race) | Medium | **Fixed** |
| 9 | `JSONRpc` used `http.DefaultClient` (no timeout) | Medium | **Fixed** |
| 10 | `VerifiedLength` always non-nil (breaks absent-vs-zero convention) | Medium | **Fixed** |

## 0bis. TL;DR — pass-2 items (indexed; see sections)

| ID | Sev | Title (working tree verified) | Status |
|----|-----|-------------------------------|--------|
| B01 | P0 | `Refresh` never polls listeners w/o ids; data race | **Proposed** (race half fixed) |
| B02 | P0 | nil-map panics in `Resolve`/`aria2Options` | **Fixed** (via `CloneMap`) |
| B03 | P0 | `Download`/`doRequest` success with empty gid | **Fixed** |
| B04 | P0 | HEAD probe failure fails resolve — probe now best-effort; **client still has no timeout** | **Proposed** (timeout part) |
| B05 | P0 | Superseded `Resolve` leaks its dry-run gid; semaphore ordering | **Fixed** |
| B06 | P0 | aria2 daemon lifecycle (zombie, SIGKILL, no detach, double Wait) | **Proposed** |
| B07 | P0 | nil `rpc` panics in `NewAria2Downloader` | **Fixed** |
| B08 | P0 | Detail pane Pause/Resume emits the opposite action | **Proposed** |
| B09 | P0 | `RSwitch`: toggles on any release, never calls `super`, wrong inactive color | **Proposed** (color fixed; click logic not) |
| B10 | P0 | Navigation sync-`Delete`, settings Back dup pages, unsubscribe | **Fixed** (with a caveat, see item) |
| B11 | P0 | "Go to file" broken `file://` URL; wrong directory | **Proposed** |
| B12 | P0 | Delete dialog deletes on Enter with Cancel focused; dialogs not freed on Esc | **Proposed** |
| B13 | P0 | `main.go`: tray-less hang, panic-on-error, no quiescing of ctx | **Proposed** (tray fixed) |
| B14 | P1 | `RTextField` icons mispositioned on HiDPI | **Proposed** |
| B15 | P1 | `removeAny` swallows errors; `Purge` always "succeeds" | **Proposed** |
| B16 | P2 | `ShouldResolve` compiles regexes per call — actually plain substring already | **Fixed** (different impl) |
| B17 | P1 | `Progress()` not clamped (row clamps, api doesn't); duplicated size format | **Proposed** |
| B18 | P1 | SQL ordering not deterministic; `updated_at` trigger/format issues | **Proposed** |
| B19 | P1 | `.gitignore` hides Python `rapid/` dir and `.github/` | **Proposed** |
| B20 | P2 | `cmd/seed` bugs (os.Kill, log.Fatal skips defers, full QApplication) | **Proposed** |
| B21 | P1 | `Sidebar.AddSection` loses activation wiring for later items | **Proposed** |
| B22 | P2 | `getCategory` returns Unknown when only one side is known | **Fixed** (semantics differ from pass 2 — read note) |
| T01 | P1 | GUI-thread dispatcher + `LockOSThread` | **Partial**: `mainthread` already used for signals; no general dispatcher yet | 
| T02 | P1 | Generic `Signal[T]` w/ disconnect | **Proposed** (manual unsubscribes exist for Service/Navigation only) |
| T03 | P1 | Context propagation + shutdown ordering | **Proposed** |
| T04 | P1 | miqt ownership policy, animation leaks, paint allocs | **Fixed** (pass 1); animation-reuse idea still **Proposed** |
| P01 | P1 | Full DB reload + full re-render every second | **Proposed** (change detection at service level only) |
| P02 | P1 | Cache tinted pixmaps/icons | **Fixed** |
| P03 | P1 | Stylesheet churn (hover/tick QSS re-parse) | **Proposed** |
| P04 | P1 | `DownloadList` rebuilds whole layout on order change | **Proposed** |
| P05 | P2 | Debounce search | **Proposed** |
| P06 | P1 | One `tellActive` instead of N `tellStatus` | **Proposed** |
| P07 | P1 | Store write amplification (SELECT+rebuild+re-read every tick) | **Proposed** |
| P08 | P1 | `speed_samples` grows forever | **Proposed** |
| P09 | P2 | Typed JSON-RPC client (double marshal), error wrapping, secrets in WS URL | **Proposed** |
| P10 | P2 | Virtualised list for thousands of rows | **Proposed (long term)** |
| P11 | P2 | Paint-time allocations — per-paint QBrush/QPen/QColor | **Fixed** (via pass-1 deletes); pre-hoisted style caches still optional |
| D01 | P1 | SQLite DSN: WAL, busy_timeout | **Partially fixed** (pragmas added; driver pin + DSN-escaping + goose provider remain) |
| D02 | P2 | Schema hygiene migration (trigger, NOT NULL, `CREATE TABLE IF NOT EXISTS`) | **Proposed** |
| D03 | P2 | `ErrNotFound`, `errors.Is(sql.ErrNoRows)` | **Proposed** |
| D04 | P2 | Remove the global `*bun.DB` singleton (unsynchronised) | **Proposed** |
| S01 | P0 | aria2 RPC without a secret (drive-by scripting risk) | **Proposed** |
| S02 | P1 | Browser bridge: Echo, Host check, strict origin list, server timeouts, typed maps | **Proposed** (typed maps partially fixed) |
| S03 | P1 | Sanitise filenames/save paths before aria2 or delete | **Proposed** |
| I01 | P2 | Replace `lib/errors` (shadows stdlib, no `Unwrap`) | **Proposed** |
| I02 | P2 | Replace `iter`/`bools` with stdlib | **Proposed** |
| I03 | P2 | `api` package: status type, presentation split, tag hygiene | **Proposed** |
| I04 | P2 | Decouple `settings` from Qt | **Proposed** |
| I05 | P2 | Functional options / structured DownloadView | **Proposed** |
| I06 | P2 | Design the not-yet-written Service write path | **Open** (service is read-only) |
| I07 | P2 | slog, doc comments, naming, dead code | **Proposed** |
| U01 | P1 | Keyboard accessibility | **Partial**: RSwitch has focus policy; rows don't; press-vs-release unchanged |
| U02 | P1 | Event filter for host resize / owner close | **Proposed** (rdialog has partial handling) |
| U03 | P2 | Selector-less stylesheets, theme colors mutable, runtime theme switch | **Partial** (alpha-aware `CssColor` landed) |
| U04 | P2 | Sidebar animation reuse + closed-state hide | **Proposed** |
| X01 | P1 | Taskfile: no PySide6 dependency for rcc; lint/fmt/race tasks | **Fixed** (Qt rcc preferred w/ PySide6 fallback); fmt/lint tasks remain **Proposed** |
| X02 | P2 | CI + golangci-lint | **Proposed** |
| X03 | P2 | Regression tests to add | **Proposed** (some exist; huge table pre pass 2) |
| X04 | P2 | Suggested order | **Merged** below |

---

# Part I — memory root cause (pass 1, already applied & tested)

## 1. MIQT ownership model

MIQT does **not** blanket-finalize Qt objects. `runtime.SetFinalizer` is attached
only by `(*T).GoGC()` and by methods returning Qt types **by value**
(`QColor.DarkerWithInt`, `QImageReader.Read`, `QPixmap.Scaled`). Objects from
`NewX(...)` constructors are raw C++ heap allocations with **no** finalizer.

Rule for this codebase:

- `NewX(...)` result that does not escape → `defer x.Delete()` (deterministic).
- `NewX(...)` result that escapes (returned/stored/cached) → hand ownership to a
  container (`SetParent`, cache map) or `x.GoGC()` once.
- Method-returned values (`reader.Read()`, `color.DarkerWithInt(...)`) are
  **already finalized** — never `Delete()` them, or you double-free (that
  SIGSEGV actually happened; see §9.1).

## 1.1 The leak, measured

Throwaway probe looping the hot path, reading `/proc/self/statm`:

```
PROBE TintedPixmap x50000:                start=31484KB  end=2243300KB  delta=2211816KB (45298 bytes/iter)
PROBE QPen+QBrush x50000:                 start=2243308KB end=2262076KB  delta=18768KB (384 bytes/iter)
PROBE QPen+QBrush with Delete x50000:     start=2262088KB end=2262648KB  delta=560KB    (11.5 bytes/iter)
```

After the fix:

```
PROBE TintedPixmap(cached) x50000:        delta=3200KB (65 bytes/iter)   // cache hits, no native alloc
PROBE renderTintedPixmap x5000:           delta=29264KB (5993 bytes/iter) // only GC-scheduled QImage finalizers
```

## 1.2 Fix — `widget/ui/icon.go`

Explicit release of every constructor temporary + memoized rendered icons so a
poll that re-sets the same icon does no native work. Key parts (final code is in
`widget/ui/icon.go`):

```go
type iconKey struct{ path string; rgba uint; size int; dpr uint16 }
var (iconCacheMu sync.RWMutex; iconCache = map[iconKey]*qt.QPixmap{})

func TintedPixmap(path string, color *qt.QColor, size int) *qt.QPixmap {
    // cache by (normalized path, packed RGBA, size, device-pixel-ratio)
    // reader/scaledSize/brush deleted; painter.Delete() ends the session
    // reader.Read() NOT deleted (miqt already finalized it)
    // out pixmap is shared, owned by the cache
}
```

`LoadIconChecked`/`LoadIcon` `GoGC()` the `QIcon` (caller only forwards it to
`SetIcon`, which copies). Public contract: **the returned pixmap is shared and
owned by the cache — treat as read-only, never `Delete()`**. Regression test:
`TestTintedPixmapIsCached`.

> If dynamic per-item icon colors appear later, bound the cache (small LRU
> keyed by `iconKey`); today's key space is a few MB max.

## 2. `theme.WithAlpha` cached & shared (Fixed)

Result memoized per `(rgba, alpha)`; shared/immutable like the icon cache;
`WithAlpha(nil, …)` returns nil. Follow-up landed: `CssColor` is always
`rgba(r,g,b,a)` (spec-independent) and nil-safe (`"transparent"`).
Remaining nit (U03): `theme.ColorXxx` are still exported mutable `*QColor`
globals — see §U03.

## 3. Paint-event leaks (Fixed)

Every custom `paint()` now frees constructor temporaries; pattern:

```go
painter := qt.NewQPainter2(dev)
defer painter.Delete()   // dtor calls end(); an End() first is redundant
pen := qt.NewQPen3(color); brush := qt.NewQBrush3(color)
painter.SetPenWithPen(pen); painter.SetBrush(brush)
painter.DrawPath(path)
pen.Delete(); brush.Delete()
```

Applied to: `progress.go`, `speed_chart.go`, `common.go` (`roundedPanel.paint`),
`download_item.go`, `sidebar_item.go`, `rswitch.go`. Misc temporaries freed
everywhere else (cursors, key sequences, QVariants, QEasingCurves, QAnyStringViews,
QPoints, QFontMetrics, QPixmaps, QIcons) — see pass-1 appendix A list.

`elidedLabel` builds one `QFontMetrics` per label, freed in `OnDestroyed`.

### 3.1 Detail expander animations

`stopDetailAnims` now `Stop()` + `DeleteLater()`s the previous
`QPropertyAnimation` pair (they were parented to the row and accumulated).
Remaining idea (T04.3): reuse one animation per property instead of Stop+new;
or use `Start2(DeleteWhenStopped)` for one-shot anims *(verify name)*.

## 4. Poll loop and subscription lifecycle (Fixed)

- `Service.OnChanged` returns an unsubscribe func (nil-slots, other indexes
  stay valid); `widget/views/download.go` wires it via `layout.OnDestroyed`.
- `Refresh` compares element-wise (`reflect.DeepEqual` per item, NOT on slices
  — nil vs empty non-nil would otherwise notify every poll on an empty DB) and
  skips `notify` when nothing changed.
- Per-row skip landed: `downloadItem.SetItem` early-returns on
  `reflect.DeepEqual(it.item, d)` (pass-1 §4.3: was proposed, now **Fixed**).

## 5. Concurrency/correctness — pass-2 items still open

### 5.1 (B01, Proposed) `Refresh()` never polls structurally

Two half-fixes landed: listener map is snapshot under `d.mu` (race fixed), and
`onResolved` uses a detached context so the resolve cleanup fires after
supersession (B05). **Still broken:** with **no** ids, the id loop iterates an
empty slice, so only `getGlobalStat` runs — listener polling relies entirely on
WS notifications; when WS reconnects mid-transfer, paused (`waiting`) or
never-notified listeners lag until the next WS event. The Python original
iterates all listeners when `id` is None.

```go
func (d *Aria2Downloader) Refresh(ctx context.Context, ids ...string) {
    d.mu.Lock()
    targets := make(map[string]listenerEntry, len(d.listening))
    if len(ids) == 0 { maps.Copy(targets, d.listening) } else {
        for _, id := range ids { if e, ok := d.listening[id]; ok { targets[id] = e } }
    }
    d.mu.Unlock()
    for gid, e := range targets {
        d.notify(ctx, gid, e)
    }
    if len(ids) > 0 { return }
    // existing getGlobalStat block unchanged
}
```

And guard each callback for nil before calling (`Listen` accepts nil callbacks
today; `notify` currently panics on them):

```go
func (d *Aria2Downloader) notify(ctx context.Context, gid string, e listenerEntry) {
    status, err := d.GetStatus(ctx, gid)
    if err != nil {
        if e.onError != nil { e.onError(err) }
        if errors.IsAria2NotFound(err) { d.Unlisten(gid) } // needs I01 helper; only forget when aria2 says gone
        return
    }
    if e.onNotify != nil { e.onNotify(status) }
    switch status.Status {
    case "error", "removed", "complete": d.Unlisten(gid)
    }
}
```

Also note: `notify` currently unlistens on **any** error (transient transport
failure drops the listener silently). Test sketch (pass 2 B01):

```go
func TestRefreshWithoutIDsNotifiesAllListeners(t *testing.T) { /* fake rpc; d.Refresh(ctx); assert got */ }
```

### 5.2 (B04, Proposed) probeClient timeout

The probe is now best-effort inside `doRequest` (errors never fail addUri —
B04's main fix), but `probeHeader` still uses `http.DefaultClient`:

```go
var probeClient = &http.Client{Timeout: 5 * time.Second}
// probeHeader: resp, err := probeClient.Do(req)
```

### 5.3 (B06, Proposed) aria2 daemon lifecycle

`spawnDaemon` still uses `exec.CommandContext` (SIGKILL on ctx cancel → no
session file) and `isAlive` (zombie look-up; a crashed aria2 is never
respawned); no `Setsid`; no `spawnMu` so `Start` + WS-goroutine `ensureDaemon`
can spawn concurrently; no `--save-session-interval` wiring despite the
setting; `Aria2SaveSessionInterval` ignored; `Stop` double-Waits.

Fix sketch (see pass 2 §B06 for full code): new `daemon.go` with
`startDaemon` (plain `exec.Command` + `Setsid`/`CREATE_NEW_PROCESS_GROUP`,
`Wait` goroutine closing an `exited` chan), `alive()` via that channel,
`stop(grace)` SIGTERM→grace→Kill; replace `process *exec.Cmd`/`ownsDaemon` with
`dm *daemon`/`adopted bool`; add `spawnMu`; pass
`--save-session-interval=max(Aria2SaveSessionInterval, 1)`; in `Stop()` take
`dm := d.dm`, call `dm.stop(2s)`. Windows build tags for signal/creation flags.
`syscall.SIGTERM`/`Signal(0)` are already unix-only helpers — note portability.

### 5.4 (B08, Proposed) Pause/Resume inverted

`DownloadItemDetail` click handler:

```go
d.pauseBtn.OnClicked(func() {
    if d.item.CanPause() {   // wrong: should be CanResume()
        d.emit(d.resumeCB, d.item.GID); return
    }
    d.emit(d.pauseCB, d.item.GID)
})
```

The handler currently checks `CanPause()` to pick which callback to emit; the
button label uses `CanResume()` (`"Resume"` when `CanResume()`), so a paused item
labeled Resume actually emits a pause request. Fix — one line:

```go
if d.item.CanResume() { d.emit(d.resumeCB, d.item.GID); return }
d.emit(d.pauseCB, d.item.GID)
```

### 5.5 (B09, Proposed) `RSwitch`

- Crash-risk fix NOT applied: `OnMouseReleaseEvent` sets `SetChecked(!Checked)`
  on **any** release (right-click, press started elsewhere) and never calls
  `super(event)`, so `QAbstractButton`'s pressed/down state is never cleared.
- Fixed: `paint()` blends `theme.ColorSurface → s.activeColor` (it previously
  blended nothing).

```go
// reuse QCheckBox click logic instead; make the 36x20 widget the hit area:
s.SetFixedSize2(36, 20)
s.SetStyleSheet("QCheckBox::indicator { width: 36px; height: 20px; }")
// then DELETE the OnMouseReleaseEvent override entirely.
```

(Alternative *(verify)*: `OnHitButton` on a directly constructed QObject.)

### 5.6 (B11, Proposed) "Go to file"

```go
q := qt.NewQUrl3("file://" + m.item.Dir)
```

Spaces/`#`/`%` break this URL, and `item.Dir` is the download base dir while
the QML opener used `dirname(files[0].path)`. Fix:

```go
// api
func (d Download) FileDir() string {
    if len(d.Files) > 0 && d.Files[0].Path != "" { return filepath.Dir(d.Files[0].Path) }
    return d.Dir
}
// context_menu.go
qt.QDesktopServices_OpenUrl(qt.QUrl_FromLocalFile(m.item.FileDir()))
```

Show `FileDir()` in the detail pane's File location too (also fixes B12 "wrong
directory" reporting there).

### 5.7 (B12, Proposed) Delete-dialog Enter + dialog lifetime

Current state: `Return`/`Enter` shortcuts run the destructive `confirm`
**regardless of widget focus**; "From disk" defaults checked; the dialog +
overlay are only freed on the explicit Cancel/Delete buttons — Esc and the
window X leave hidden dialogs and overlays behind.

Fix (sketch; also drop the two QkeySequence shortcuts entirely — `QDialog`
already handles Enter (default button) and Esc):

```go
dialog.SetAttribute(qt.WA_DeleteOnClose) // closes frees dialog + overlay via OnFinished
cancel.SetAutoDefault(false); del.SetDefault(true); cancel.SetFocus()
fromDiskOnConfirm := func() { fromDiskState := fromDisk.IsChecked(); /* read before Accept */ } // reworked
```

In `RDialog.Open` we already clean up the overlay on `OnHideEvent`, but a hidden
dialog + its overlay survive until the page is deleted; one acceptance is to
always `DeleteLater()` after `Hide()` on every close path. Dialog callers
(`openDeleteDialog`, `NewDeleteConfirmationDialog`, download dialog) should also
remove their `Hide()+DeleteLater()` now-totally manual paths.

### 5.8 (B13, Proposed) `main.go` lifetime bits

Already applied: `QSystemTrayIcon_IsSystemTrayAvailable()` (tray-less close +
`QuitOnLastWindowClosed`), `super(event)` used properly, host/`mainthread`-
routed signal quit, ≥ one deferred DB close. Still open:

- `panic(err)` for startup failures (skips `db.Close`, shows nothing); use
  `run() error` + `slog.Error` + `os.Exit(1)`.
- `super(event)` then `event.Ignore()` correctness — re-verify both branches
  of trayAvailable (currently handled but confirm Ignore+Hide ordering).
- Signal goroutine traps `SIGHUP` into quit — a terminal hangup kills the app;
  drop it or treat as a full quit-with-cleanup.
- `settings.Default(".")` (see §B13 / I04 — plugin dir rooted at CWD).
- No cancellable app ctx; view timers build `context.Background()` each page
  (see T03).

Suggested `main` skeleton (pass-2 B13): `run() error` + `signal.NotifyContext`,
cfg from `os.Executable()`, `QMessageBox` on startup error.

### 5.9 (B10, Fixed) Navigation lifetime

`Back()`/`Replace()` now both `Hide()+DeleteLater()`; settings "Back" pushes
the settings page as a new route instead of spamming duplicate download copies
(destination "back" calls `navigation.Back()` via `SidebarItemData.OnActivated`
in `widget/components/settings/layout.go`); `Navigation.OnRouteChanged` +
`Service.OnChanged` both return unsubscribes and the download view unregisters
on widget destroy. Pass 2's suggestion to use `Back()` from the settings page
was adopted (push was intentional per QML; keep as is).
Caveat: route identifiers are still bare strings — B10's "Route type" part is
now covered by `app.RouteDownload`/`app.RouteSettings` constants in
`widget/app/routes.go`; lowercase destinations like `"general"`/
`"personalization"`/destination routing are still strings (I07/B10 tail).

### 5.10 (B15, Proposed) `removeAny` swallows errors

`Remove`/`Purge` discard delete errors; `Purge` always returns nil.

```go
func (d *Aria2Downloader) removeAny(ctx context.Context, id string, methods ...string) error {
    var errs []error
    for _, m := range methods {
        params, _ := json.Marshal([]any{id})
        if _, err := d.rpc.Call(ctx, m, params); err != nil { errs = append(errs, err); continue }
        return nil
    }
    return stderrors.Join(errs...)
}
func (d *Aria2Downloader) Purge(ctx context.Context, id string) error {
    if err := d.removeAny(ctx, id, "aria2.removeDownloadResult", "aria2.remove"); err != nil && !errors.IsAria2NotFound(err) { return err }
    return nil
}
```

(Join of typed not-found errors still satisfies `errors.As`, so the Is check
works; but it needs the I01 `Unwrap` + `IsAria2NotFound`.)

### 5.11 (B17, Proposed) `Progress()` clamp + duplicated FormatSize

`api.Progress()` can exceed 1 when aria2 briefly reports completed > total
(row widget clamps when painting; `FormatProgress` doesn't; `FormatEstimation`
uses the raw difference). Also `ResolvedURL.FormatSize` duplicates
`helpers.FormatSize` verbatim.

```go
func (d Download) Progress() float64 {
    if d.TotalLength <= 0 { return 0 }
    return min(1, max(0, float64(d.CompletedLength)/float64(d.TotalLength)))
}
func (r ResolvedURL) FormatSize() string { return helpers.FormatSize(r.Size) }
```

### 5.12 (B18, Proposed) SQL determinism + `updated_at`

- `Order("created_at DESC")` only; ties non-deterministic → add `"gid ASC"`.
- `getSpeedHistory`'s `Relation("URIs")` (via `loadFileRows`) — order it
  (`q.Order("id ASC")`).
- `trg_downloads_updated_at` writes `CURRENT_TIMESTAMP` (text
  `2006-01-02 15:04:05`) while bun writes Go `time.Time` text — mixed formats
  plus a second UPDATE per write. Fix via D02 (drop trigger, set in Go).
  NOTE: pass 1's §6.1 upsert proposal already sets `updated_at = now` in Go —
  the trigger must go for that to stick.

### 5.13 (B19, Proposed) `.gitignore`

```
.*/            # ignores .github/
/rapid         # also ignores the rapid/ python package dir (source!)
```

```gitignore
.venv/
__pycache__/
*.pyc
dist/
.task/
.*/!
!.github/
!/rapid/
rapid/deployment/
```

(or always build to `dist/` and drop the `/rapid` rule — build already emits
`dist/rapid`).

### 5.14 (B20, Proposed) `cmd/seed`

- `signal.NotifyContext(..., syscall.SIGABRT, os.Interrupt, os.Kill)` —
  SIGKILL is uncatchable; SIGABRT shouldn't be trapped.
- `log.Fatal` skips deferred `db.Close()`/`cancel()`.
- Full `QApplication` just to resolve paths; `errgroup` for one goroutine.
- Amusing extra: `defer cancel()` after the Fatal branch is dead code.

```go
func main() { if err := run(); err != nil { log.Fatal(err) } }  // run() carries defers
// ctx: NotifyContext(Background(), os.Interrupt, syscall.SIGTERM)
```

I04 lets `cmd/seed` drop the QApplication entirely.

### 5.15 (B21, Proposed) Sidebar section wiring

`AddSection` ties `OnActivated` to items existing at wiring time; items later
replaced by `SetItems` never reconnect → sidebar click does nothing. Move
ownership into `SidebarSection` (`s.activated.Emit(data.Destination)` from
`SetItems`, observe from `Sidebar.AddSection`). Code in pass 2 §B21.

---

# Part II — performance (state after pass 1 + what remains)

## P01 — full reload on the GUI thread (Proposed)

Service-layer change detection + per-row skip landed (§4 above). The cost is
still O(entire DB + all rows) per tick on the GUI thread:

- SQL still loads **all downloads + all files + all URIs** each tick. Pass 2's
  layered fix stands:
  - (a) comparable `ViewKey` per download; `slices.EqualFunc` on keys
    (replaces `reflect.DeepEqual`, also cheaper than pointer/data compare).
  - (b) async `RefreshAsync` (worker goroutine + `gui.Post`), expand-only
    sparkline loads for the active expanded row.
  - (c) end-game: event-driven service (downloader callbacks → in-memory list +
    `changed(gids)`; DB read only at startup) — pairs with I06.

## P02 (Fixed) — icon/pixmap cache

`TintedPixmap` memoizes (path, RGBA, size, dpr) → shared `*QPixmap`; `LoadIcon`
is cache-fed. Widgets also skip no-op re-sets. See §1.

## P03 (Proposed) — stylesheet churn

`RButton` still re-runs full `setStyleSheet()` on every Enter/Leave; `styledLabel`
still allocates one QSS string per label (construct-time only — fine);
`SidebarItem.refreshStyle` still runs for all badge updates every tick if
counts differ. Pass-2 fixes: drop redundant hover setStyleSheet (only `outlined`
needs a swap; prebuild normal+hover `QIcon`s and swap on hover), memoize the
emitted QSS by `(variant, outlined, link, iconOnly, hovered, enabled, radius)`,
guard `SidebarItem.SetCount(n == i.count)`, replace label stylesheets with a
palette (`QPalette.WindowText`) — the QSS form is more work; palette-color use
here is a nice nicety, not required.

## P04 (Proposed) — list reflow

`SetItems` still `detachAll()`s + re-adds every row when order changes (one new
arrival at the top = full relayout + lost hover). On populate, `AddStretch` is
added per rebuild (temporary leak fixed by explicit delete; a permanent stretch
is cleaner). Pass-2 reconcile sketch keeps a single permanent stretch and edits
the layout minimally (Remove/Insert only moved rows). Also fixes `TakeAt`
spacer-item drop risk.

## P05 (Proposed) — debounce search

Each keystroke → `SetSearch` → full re-render. `NewDebouncer(parent, 150ms)`
with a `QTimer` on text change; `list.Collapse()` kept on actual trigger only.

## P06 (Proposed) — one `tellActive`

N listeners = N `tellStatus` RPCs per tick. One `tellActive(StatusKeys)` covers
actives; only non-active listeners need per-gid `tellStatus`. Sketch in pass 2.
Do this **after** B01 (the all-listeners loop) — otherwise the same broken
"only notified when WS pings us" behavior gets replicated at scale.

## P07 (Proposed) — store write amplification

Per active download per tick: SELECT existing → INSERT/UPDATE (full row) →
SELECT file rows → per-file UPDATE + URIs DELETE+INSERT → prune DELETE →
`getDownload` 3-SELECT re-read of what the caller just had. Fully replaces
`upsertDownload`'s SELECT-then-write. Fixes:

```go
// one tx, atomic, no re-read:
q := tx.NewInsert().Model(&row).On("CONFLICT (gid) DO UPDATE")
for _, c := range downloadCols { q = q.Set("? = EXCLUDED.?", bun.Ident(c), bun.Ident(c)) }
```

- extend `syncFiles` with same-file / progress-only fast paths
- hot path: single-column `updateProgress` UPDATE when only completed/speed
  moved (the 1 Hz case), full upsert only on shape change
- returns error only; caller already holds the merged value.

## P08 (Proposed) — `speed_samples` growth

One row/sec/active, no pruning, only the last 60 read.

```go
// prune everything older than the 60th-newest sample in the same statement flow:
Where("ts < (SELECT ts FROM speed_samples WHERE gid = ? ORDER BY ts DESC LIMIT 1 OFFSET ?)", gid, keep-1)
```

Plus: `DELETE ... WHERE gid = ?` on terminal status; seeder bulk insert.
(Note: speed_samples retention lives in the same write path as P07 — do them
together.)

## P09 (Proposed) — typed JSON-RPC + error wrapping

`rpc.Call(ctx, method, params []byte)` round-trips params through
marshal/unmarshal/marshal. Pass-2 sketch: `Call(ctx, method, params []any,
result any)`; token inserted in exactly one place; `atomic.Int64` id; body read
capped; error `&Aria2Error{Code, Msg, Err}` with `Unwrap` so
`context.Canceled` is visible; drop `?token=` from the WS URL (that query form
is not an aria2 feature — aria2 authenticates in params; the secret currently
lands in logs via `Dial` failures). Also: single definition of
`StatusKeys`/`JSONRPCVersion` (currently defined in both `aria2.go` and
`rpc/jsonrpc.go`).

## P10 (Proposed, long-term) — virtualised list

Composite QWidget per download is fine for ~100 rows. Options: cap/paginate
"all" view; or `QListView` + model + delegate. Only after P01–P04. Seed 5k rows
with `cmd/seed` to measure.

## P11 (Fixed with an optional follow-up) — paint allocations

Per-paint `QPen/QBrush/QColor` now deleted (pass 1). Hoisting them into
cached `rowStyle`/`switchBrushes` (pass 2 §P11) only buys perf, not
correctness — do if profiling shows it matters (hi-DPI repaint loops).

---

# Part III — database

## D01 (Partial) — SQLite pragmas/driver/migrations

Landed: WAL+`busy_timeout(5000)`+`synchronous(NORMAL)` appends in
`withPragmas` (file DBs only; memory DBs skip WAL correctly), goose ctx-ordered
`UpContext`. Still open:

- `sqliteshim` picks mattn (cgo) vs modernc silently. miqt already needs cgo,
  so pin **explicitly** to one driver (modernc is in go.mod as indirect) and
  drop the shim — `_pragma=…(pause)` syntax is modernc-only.
- `withPragmas` returns the DSN untouched if *it already contains* `_pragma=`
  (so a user DSN silently loses foreign keys); use `url.Values` on a
  `file:` URL (spaces `?`/`#` still break a raw-path DSN).
- `goose.SetBaseFS`/`SetDialect` are process-globals; `goose.NewProvider`
  removes them (`NewProvider(DialectSQLite3, sqldb, fsys)` + `Up(ctx)`).
  (Current `migrate(sqldb, ctx)` muddles with stat order; rewrite with provider, plus a
  guard test from pass 2 §D01.)

```go
func TestPragmasApplied(t *testing.T) {  // guards the shim/driver choice
    d, _ := OpenMemory(t.Context())      // or a file DSN for WAL
    var fk int
    require.NoError(t, d.QueryRowContext(t.Context(), `PRAGMA foreign_keys`).Scan(&fk))
    require.Equal(t, 1, fk)
}
```

## D02 (Proposed) — schema hygiene migration `00003_*.sql`

- Drop `trg_downloads_updated_at` (write `updated_at` in Go — pairs with P07).
- `CREATE TABLE IF NOT EXISTS` in versioned migrations hides drift: plain
  `CREATE TABLE` going forward.
- Make the store's assumptions match the schema (`NOT NULL DEFAULT ''/0`) or
  flip the structs to pointers; pre-release, edit `00001` and delete dev DBs.
- `resolved_url` should store NULL when absent, not an empty serialized
  `ResolvedURL`.

## D03 (Proposed) — `ErrNotFound`, `errors.Is`

`getDownload` returns `(zero, nil)` when missing. Either document it or:

```go
var ErrNotFound = stderrors.New("download not found")
// errors.Is(err, sql.ErrNoRows) — never == compare
```

Related bug: `upsertDownload` checks `err != sql.ErrNoRows` with `==` (same fix),
and on a lookup error other than NoRows it proceeds to *upsert with a zero
row*.

## D04 (Proposed) — global `*bun.DB`

`db` package global is unsynchronised (`Open` twice leaks, `Close` races with
readers, tests can't be parallel). Pass 1 proposed an RWMutex; pass 2 proposes
removing the global entirely (`Open(ctx) (*bun.DB, error)` + inject into
`Service`/`Seed`). Do the injection (better testability, matches the
interfaces-first direction of I06); keep a mutex only if you keep the global.

---

# Part IV — security

## S01 (Proposed, P0) — aria2 without a secret

`Aria2Token` defaults empty. Any local web page can
`fetch("http://127.0.0.1:6800/jsonrpc", {mode:"no-cors", method:"POST",
body: JSON.stringify({..., "aria2.addUri", options:{dir,out}})})` (JSON top
level works with any content type, no preflight). Persisted secret:

```go
func loadOrCreateSecret(path string) (string, error) {
    if b, err := os.ReadFile(path); err == nil && len(bytes.TrimSpace(b)) >= 16 {
        return string(bytes.TrimSpace(b)), nil   // reused so restart can adopt the same daemon
    }
    raw := make([]byte, 16); rand.Read(raw)
    return hex.EncodeToString(raw), os.WriteFile(path, raw, 0o600)
}
```

called from `settings.Default` when the user hasn't set one (file
`aria2.secret` in `DataDir`).

## S02 (Proposed) — browser bridge

Current state (after typed-headers fix): `Headers`/`Cookies` are still
`map[string]any` on the wire, but the request-side decode path funnels through
`lib.StringMap`/`CloneMap` so non-string values no longer fatal the bind and no
writes happen to nil maps (B02). Still open:

1. **Echo** (+ gommon/validator/fasttemplate deps) for 3 routes; Go 1.22+
   `net/http` pattern mux is enough and matches "stdlib only".
2. `http.Server` with no `ReadHeaderTimeout`/`ReadTimeout`/`WriteTimeout`/`IdleTimeout`.
3. `originAllowed` accepts ANY `chrome-extension://…` — every installed
   extension can queue downloads. Pin the Rapid extension id for Chrome
   (Firefox ids are random: allow scheme + require `X-Rapid-Extension`).
4. No `Host` validation → DNS-rebinding exposure. The trustedMiddleware
   Origin check does not protect `GET /health` from a rebinding page that
   lacks an origin — guard `net.SplitHostPort(r.Host)` ∈ {127.0.0.1,
   localhost, ::1} before anything else.
5. Start() races: `b.server != nil` check releases the lock before the
   listen+assign — two `Start`s can double-listen. Hold the lock for the whole
   path (or a start-once gate).
6. Handlers run on HTTP goroutines; anything touching widgets must go through
   a `gui.Post` (T01). Good target: assert/document `OnDownloadRequested`.
7. Error type assertions (`err.(*echo.HTTPError)`) go away with the stdlib
   switch; `BridgeResponse map[string]any` → typed struct.
8. `Validate` accepts `"http://"` (`u.Host == ""`) and mutates the receiver —
   rename `Normalize()` and check the host.
9. Exported mutable globals (`BlockedHeaders`, `AllowedOriginSchemes`) →
   unexported.

Struct/transport changes are typed in pass 2 §S02 (stringMap unmarshal, okResponse,
hostGuard, srv timeouts); keep the JSON shapes the extension expects.

## S03 (Proposed) — sanitise remote-controlled paths

When the write path (I06) lands, sanitize anything that reaches `dir`/`out` or
"delete from disk":

```go
func safeFileName(name string) string { /* base + drop control chars + rejects "."/".." */ }
func within(base, p string) bool { /* filepath.Rel-based containment + no symlink escape via os.Lstat */ }
```

Only `os.Remove` paths `within(download.Dir, p)`.

---

# Part V — architecture/idioms

## I01 (Proposed) — replace `lib/errors`

Package name shadows stdlib `errors`; no `Unwrap`, no `Is/As` interop; seven
near-identical types; `JsonRpcError.Error()` drops the code. Move to
`lib/apperr` with `Code/Msg/Err` + `Unwrap` + `IsAria2NotFound`, and prefer
plain `fmt.Errorf` + sentinels elsewhere. Needed by B01/B15/S02/P09.

## I02 (Proposed) — replace `lib/helpers/iter` and `bools`

`iter` re-implements Map/Filter/ForEach/Reduce (shadowing Go 1.23's stdlib
`iter`), with `Reduce`'s arg order flipped. `bools.Tertiary` (misspelled) is a
one-liner `cmp.Or`. Show the base idiomatic forms. `lib.StringToInt` silently
truncates for narrow types — either document or return `(T, error)`.

## I03 (Proposed) — `api` package

- Move `Download.Format*`/`FormatStatus`/`FormatEstimation` presentation out of
  `api` (`widget/components/downloads/format.go`) so `api` stays a pure domain
  package.
- `Status` as a typed enum with `IsActive/IsTerminal` (raw string literals
  appear in ~30 sites incl. `notify`, `CanPause`, `store`, menus).
- One presence rule for optional fields: today `VerifiedLength *int64` +
  `omitempty` on others + `Resolved *ResolvedURL` mix — since
  `StringToIntPtr` made `VerifiedLength` truly optional, consider plain values
  + `HasVerified`, or keep pointers for all optional numerics.
- Dead JSON tags: only `ResolvedURL` is serialised; remove tags from the rest
  (or document why).
- `Download` is copied by value with slices + pointer inside; either document
  immutability or add `Clone()` (list code shallow-copies today).

## I04 (Proposed) — decouple `settings` from miqt

`settings` imports Qt for `QStandardPaths`, forcing every consumer of
`services/download` (and `cmd/seed`, and any non-GUI test) into cgo+offscreen
Qt. Inject `settings.Paths{AppData, Downloads}`:

```go
func Default(baseDir string, p Paths) (Settings, error)   // pure Go + error
// qtpaths subpackage: the only Qt-aware piece, imported by main only
```

Also fixes B13's `Default(".")` from the executable dir, and lets `cmd/seed`
drop its QApplication. Note the freopen of session file
(`os.OpenFile(..., os.O_CREATE|os.O_APPEND)` without `os.O_WRONLY` today) is
harmless but sloppy; give it an explicit mode.

## I05 (Proposed) — ergonomics

- `NewRButton(text, variant, outlined)` etc. → functional options
  (`WithVariant`, `Outlined`, `WithIcon`).
- `NewDownloadView` is a 110-line closure with no handle →
  `type DownloadView struct{...}` (also gives B10.3/T02 its unsubscribe owner).
- `downloads.NewLayout(navigation)` ignores its parameter; settings layout has
  a variable named `copy` (shadows builtin); `downloadDialog := ...Open()` in
  the tray menu also `Open()`s without a parent handle to hide later.

## I06 (Open) — the write path

`Service` is read-only (no Start/Pause/Resume/Remove/Download/Resolve wiring;
action buttons print to stdout). Design from pass 2 §I06: one owner goroutine
holding `map[gid]Download`; downloader callbacks update memory+persist delta
(P07) → emit `changed(gid)`; dispatch through `gui.Post`; inject `Store/
Downloader/Resolver/Clock` interfaces; recover() around `Listen` callbacks; typed
`ResolveOptions` instead of `map[string]any` (dynamic-type mismatches are
silently dropped today).

## I07 (Proposed) — housekeeping

- `log.Print("websocket connect failed, retrying")` — drops the error; slog
  with fields.
- Doc comments starting with the wrong name (`getAllDownloads` documented as
  "All returns", etc.).
- Naming: `fileURIrow` → `fileURIRow`; `Ari2VersionResponse` typo in rpc/api if
  still present; sidebar label "Setting" vs "Settings"; `bools.Tertiary`.
- Dead: `clipboard.Clipboard` empty struct (make `ExtractURL` a function;
  also strip `>'"` from the match end), unused `MenuItems`,
  `JSONRPCVersion`/`StatusKeys` duplicated in `aria2.go` + `rpc/jsonrpc.go`,
  exported mutable globals → unexported/frozen.
- `Service` methods: inconsistent nil-receiver guards (`All` vs `Filtered`) —
  pick one policy.

---

# Part VI — UI/Qt

## U01 (Partial) — keyboard accessibility

`RSwitch` sets StrongFocus. `SidebarItem`/`downloadItem` don't: they activate on
mouse **press**, ignore Return/Space, and have no focus ring. Apply pass-2
§U01: StrongFocus, `OnKeyPressEvent` Enter/Return/Space → activate, activate on
*release* inside bounds, `SetAccessibleName` on icon-only buttons + switch,
focus ring in `paint()` when `HasFocus()`.

## U02 (Proposed) — host resize / owner close tracking

Notification `Manager` doesn't re-layout on window resize; `RDialog`'s overlay
is sized at open only (it self-cleans on hide/destroy). Pass-2 event-filter
watcher (`NewQObject2(parented)` + `OnEventFilter` catching
Resize/Move) *(verify the exact miqt override signature)* re-sizes popups and
overlays and closes with the owner.

## U03 (Partial) — theme/QSS hygiene

Landed: alpha-aware `CssColor`, `WithAlpha` cache. Still open:

- selectors-less stylesheets (`Header`, clipboard banner, overlay,
  DownloadList `QWidget#qt_scrollarea_viewport` is correctly scoped; others
  are not) leak styles to descendants — scope via `#objectName`.
- exported mutable `theme.ColorXxx` globals (`SetAlpha` on one corrupts every
  user) → accessors returning cached copies, or hex strings built at use.
- `RButton` QSS on outline/focus/hover already covers most;
  nothing listens for runtime palette change (see U02 filter +
  `QEvent::ApplicationPaletteChange`); `TouchTarget = 32` vs QML 36 mismatch
  persists for row min-height (`theme.TouchTarget + 2*SpacingSm` vs the QML's
  36px row) — reconcile against the QML originals.
- no `opacity: 0.5` QSS on RButton in this tree (pass-2's item already absent).

## U04 (Proposed) — sidebar animation

`Sidebar.animate` creates two `QPropertyAnimation`s per toggle (parented, so
they accumulate with the widget) and `SetOpen(false)` leaves the layout wider
than 0 whenever `minimumSizeHint` disagrees. Reuse one pair (T04.3), and
`Hide()` at the end of the closing animation + `Show()` before opening.

---

# Part VII — tooling, tests

## X01 (Partial) — Taskfile

Landed: `rcc` prefers Qt's own resource compiler (qt6-base-dev-tools /
distro qt paths) with PySide6 fallback; `test` and `test:race` tasks with
`QT_QPA_PLATFORM=offscreen`; builds go to `dist/rapid`. Still open: `fmt`
(`gofmt -s -w .`), `vet`, `lint` (golangci-lint), `tidy`. Optional (pass-2):
drop `rcc` entirely — `//go:embed` the SVGs and render via `QSvgRenderer` from
bytes; no generated files, no external tool at all.

## X02 (Proposed) — CI + golangci-lint

`.github/workflows/ci.yml`: apt/fedora qt6 dev packages; `go vet`; golangci
(errcheck, govet, staticcheck, revive, gocritic, bodyclose, noctx, errorlint,
gosec, unused, ineffassign, copyloopvar, perfsprint); `QT_QPA_PLATFORM=offscreen
go test -race ./...`. (B19 must land first — `.*/` currently excludes
`.github/` from the repo's own git tracking.)

## X03 (Proposed; extend pass-1's two tests)

Existing: `TestTintedPixmapIsCached`,
`TestServiceUnsubscribeAndNoOpRefresh`, store/service/downloader suites.
Add the regressions from pass 2 §X03 (B01..S02 table). Additional hygiene:
one shared Qt `TestMain` (`internal/qttest`), avoid asserting exact QSS
strings, `t.Context()`/`t.Cleanup` everywhere.

### How to re-measure native growth manually

Add a temporary test looping the hot path, printing `/proc/self/statm` RSS
(page-size multiples). Warm cache: 50 000 `TintedPixmap` calls grow RSS by
~60 B/iter instead of ~45 KB/iter pre-fix.

## X04 — suggested order (merged)

1. **Safety net:** land the missing `test -race` habit + D01 pragma test, then
   B01 (Refresh all-listeners + nil-callback guards) and S01 (aria2 secret).
2. **aria2 correctness:** B03 head-check re-verify, B04 timeout, B05 already
   done — re-verify with the pass-2 test, B06 daemon lifecycle, B15, P06.
3. **UI crashes/footguns:** B08 (pause/resume), B09 (RSwitch click + `super`),
   B11 (file URL), B12 (dialog lifetime + Enter), B13 (main.go panic paths),
   B21 (sidebar wiring), U01 basics.
4. **Foundations:** T01 (a real dispatcher; the signal case is done),
   T02 (Signal[T]), I01, I04, D04.
5. **Performance:** P01 (ViewKey + async) → P03 → P04 → P05 → P06 → P07 (+P08)
   → P09.
6. **Quality:** I02, I03, I05–I07, U02–U04, X02, B14–B22 tails, D02, D03.
7. **Long term:** P10, event-driven service (P01c + I06).

---

## §9.1 The double-free trap (do not repeat)

Deleting a method-returned value crashed with SIGSEGV (QImageReader.Read →
finalizer → double free). Only `NewX(...)` constructor results may be
`Delete()`d. Generated method ends with `_goptr.GoGC()` → do not Delete.

---

## Appendix A — files changed in pass 1

```
widget/ui/icon.go
widget/theme/theme.go
widget/ui/rbutton.go
widget/ui/rswitch.go
widget/ui/rtextfield.go
widget/ui/rdialog.go
widget/ui/icon_test.go
widget/components/sidebar.go
widget/components/sidebar_item.go
widget/components/downloads/common.go
widget/components/downloads/progress.go
widget/components/downloads/speed_chart.go
widget/components/downloads/download_item.go
widget/components/downloads/delete_dialog.go
widget/app/navigation.go
widget/views/download.go
services/download/service.go
services/download/service_test.go
services/download/downloader/aria2.go
services/download/rpc/jsonrpc.go
services/notification/popup.go
services/notification/tray.go
```

## Appendix B — miqt ownership cheat-sheet

| Call | Finalized by MIQT? | Action |
|------|--------------------|--------|
| `NewQPen3`/`NewQBrush3`/`NewQPainter2`/`NewQPainterPath`/`NewQColor*`/`NewQFontMetrics`/`NewQSize2`/`NewQPoint2`/`NewQCursor2`/`NewQKeySequence2`/`NewQVariant*`/`NewQEasingCurve3`/`NewQPixmap*`/`NewQIcon*`/`NewQImageReader*` | No | `Delete()` if local; `GoGC()` if returned |
| `color.DarkerWithInt(n)`, `pixmap.Scaled(...)`, `reader.Read()`, `label.Font()`, `layout.TakeAt(...)` | Yes | **Do not** `Delete()` |
| `NewQWidget*`/`NewQTimer*`/`NewQPropertyAnimation*` **with a parent** | Owned by parent | No action |
| `NewQMenu2()` with no parent | No | `DeleteLater()` or keep alive intentionally |

## Appendix C — pass-2 items not verifiable without a toolchain

(miqt name marks carry over verbatim): `QPalette.SetColor2`,
`QAbstractAnimation` delete-when-stopped overload, `OnEventFilter` on a
constructed QObject, `QRect.Contains(*QPoint)`, `OnHitButton`,
`QSystemTrayIcon_IsSystemTrayAvailable` (already used in main.go — known good),
`QGuiApplication_SetDesktopFileName`, `mainthread` package (known good —
`main.go` uses `mainthread.Start`).
