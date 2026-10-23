# Final port: full wiring + missing components (PySide6 → Go + miqt)

## Goal

Finish the port. The list page, components, reactive service, aria2 downloader,
browser bridge HTTP server, notifications, and DB/store are done. What remains
is **not just wiring**: the full DownloadDialog, the whole plugin system, the
quit dialog, clipboard watching, and every service action are unported. This
task covers all of it, ending with a fully working app.

Idiom rule (same as every prior task): **prefer idiomatic Go + miqt over 1:1
QML translation**. Simplicity wins over cleverness. Everything must be
concurrent-safe and must not block the GUI thread.

## How to use this file

This is a pair-programming script. Steps **S0–S19** are ordered by dependency
and each is small enough for one session. For each step:

1. Read the listed source + target files (the "Read first" list).
2. Implement only that step.
3. Run the step's **Verify** block.
4. Commit before moving on. Do not implement ahead of the current step.

If a signature in this file disagrees with the code, **the code wins** — this
file was written against the tree at the time; re-read the file in question.

---

## Current state

### Done (reuse, do not reimplement)

| Area | Where | Notes |
|---|---|---|
| Reactive core | `lib/reactive` | Signal/Computed/Effect/Batch, global mutex, epoch dedup |
| Download list UI | `widget/views/download.go`, `widget/components/downloads/*` | binds `service.Items` / `FilterChanged`; action callbacks are stubs |
| Download service (read-only) | `services/download/service.go` | Items/Counts/filter/Refresh( DB )/SpeedHistory. Ends with a `TODO(full-wire)` listing everything this task adds |
| Store | `services/download/store.go` | getAll/get/addSpeedSample/getSpeedHistory/remove/upsert/syncFiles — all take `bun.IDB`, use `db.DB()` |
| Models | `services/download/api/api.go` | Download + helpers (`CanPause/CanResume/IsError/IsTerminal/Progress/FormatSpeed/...`), `ResolvedURL` |
| aria2 downloader | `services/download/downloader/aria2.go` | complete: daemon spawn/adopt, ws notifications, claim/deliver exactly-once, Resolve with HEAD probe |
| Downloader ifaces | `services/download/downloader/downloader.go` | `Downloader{Start(ctx),Stop(),Download,Pause,Resume,Remove,Purge,GetStatus,Listen(id,onNotify,onError),Unlisten(id),Refresh(ctx,id...)}`, `Resolver{ShouldResolve(uri),Resolve(ctx,uri,opts)}`, `NotifyCallback func(api.Download)`, `ErrorCallback func(error)` |
| RPC client | `services/download/rpc` | `NewJSONRpc(host, port, token)` implements `rpc.Rpc` |
| Browser bridge | `services/browser/bridge.go` | `NewBridge(host, port)` / `NewBridgeWithConfig`, `Start(ctx)`/`Close(ctx)`, `Addr()`, callback fields `OnDownloadRequested func(BrowserRequest)` and `OnError func(string)`; `DefaultAddr = "127.0.0.1:17654"` |
| Notifications | `services/notification/` | Error/Success/Info, tray menu, popups. **No public SetBadge yet** (tray has private `setBadge`) |
| Clipboard regex | `services/clipboard/clipboard.go` | `ExtractURL(text)` only — no QClipboard wiring |
| Settings | `services/settings` | `Settings{DataDir, DownloadDir, PluginDirs, BaseDir, Aria2Host, Aria2Port, Aria2Token, Aria2SessionFile, Aria2SaveSessionInterval, PollIntervalMs}` |
| DB | `db` | bun/SQLite singleton `db.DB()`, migrations done |
| main.go | `main.go` | window + tray + navigation + notification service; **no downloader, no bridge, no service.Start, quit goes straight to Quit** |

### Missing (this task builds all of it)

1. **Download service full-wire**: constructor deps, Start/Close lifecycle, the
   notify/merge path, and every action — pause/resume/stop/delete/download/
   resolve. (The `TODO(full-wire)` block at the bottom of `service.go`.)
2. **Plugin system** (excluded from the port until now): manifest parsing,
   line-delimited JSON-RPC over stdio, short-lived process per resolve,
   directory scan, first-match resolver chain. Python source:
   `rapid/backend/plugin/{manager,protocol,transport,resolver,models}.py`.
3. **Full DownloadDialog**: `download_dialog.go` is a 44-line stub. Needs
   KVEditor, AdvancedOptions, editable resolved-URI rows, debounced resolve,
   submit, folder picker, openFromBrowser, clipboard paste.
4. **QuitConfirmationDialog** + quit guard (active downloads check).
5. **QClipboard watching** (banner visibility, dialog prefill, copy action).
6. **aria2 construction + lifecycle in main** (JSONRpc + Aria2Downloader +
   service.Start/Close), browser bridge startup, badge, completed/failed
   notifications, shutdown ordering.
7. **A serial action queue** (`lib/worker`) — control actions (pause/resume/
   stop/delete/download/poll) currently have no execution mechanism at all;
   fire-and-forget goroutines would race user ordering.
8. **Remaining animations**: AdvancedOptions accordion (100 ms), ResolverUri
   row slide/fade (300 ms) + loading spinner, list add/remove transitions
   (150 ms). The helpers exist (`common.go`); the call sites don't.

### Out of scope (do not build)

- SettingsPage — placeholder ("Settings" label) in the Python app too.
- `asDict`/QML model-role layers — Go passes typed structs.
- Python's zenity/kdialog folder-picker fallback chain — native
  `QFileDialog` replaces all of it.
- Editing `rapid/` Python sources — read-only reference.
- New Go dependencies. Plugin transport is stdlib `os/exec` + `bufio`.

---

## Ground rules

### Threading (read twice — every step depends on this)

`lib/reactive` fires effects **synchronously on the goroutine that called
`Signal.Set`**. All widget access must be on the GUI thread. Therefore:

- **Rule 1 — marshal into the GUI thread.** Downloader callbacks (websocket
  notify goroutine, Refresh tell results) and bridge callbacks (echo handler
  goroutines) run off-thread. First statement inside every such callback:

  ```go
  mainthread.Start(func() { /* touch service state / widgets here */ })
  ```

  (`github.com/mappu/miqt/qt6/mainthread` — already used in `main.go` for the
  signal handler. `mainthread.Start` from a non-GUI goroutine queues onto the
  GUI thread; it is safe to call from the GUI thread too.)

- **Rule 2 — never do network I/O on the GUI thread, and never race the
  user.** Every aria2 JSON-RPC call (`Download/Pause/Resume/Remove/Refresh`)
  and every plugin subprocess run is network/disk I/O — off the GUI thread.
  But plain `go func()` per click is wrong for a second reason: pause→resume
  as two racing goroutines can land at aria2 in the wrong order and leave the
  wrong final state. Control actions go through **one serial action queue**:
  a small reusable `lib/worker.Queue` (built in S2), not a channel embedded
  in the service. The service becomes a user of the queue instead of an
  owner of concurrency machinery.

  - `worker.Queue` API: `NewQueue(depth)`, `Start(ctx, onError)`,
    `Enqueue(task string, fn func(ctx) error) bool`, `Close()`. One drain
    goroutine runs tasks in order; `onError(task, err)` is the single
    error-report point (the service passes one that marshals through
    `mainthread.Start`). `Enqueue` is goroutine-safe (mutex + closed flag)
    and returns false after `Close` — dropping is correct, the app is
    quitting.
  - Public actions enqueue; they never spawn:

    ```go
    func (s *Service) Pause(ctx context.Context, gid string) {
        s.queue.Enqueue("pause", func(ctx context.Context) error {
            _, err := s.downloader.Pause(ctx, gid)
            return err
        })
    }
    ```

  - **Resolve is the exception**: an 8 s plugin budget through the serial
    queue would block a pause click. It is read-only with no ordering
    hazard against control ops, so it keeps its own goroutine (S9).
  - Poll (`Refresh`) enqueues too — its tell results then serialize with
    user actions instead of interleaving.

- **Rule 3 — DB writes stay on the GUI thread** (inside the marshaled
  callback). SQLite is local and fast; this keeps a single-writer invariant
  for free. Do not add locks for it.

- **Rule 4 — service state has no mutex by design.** All mutation happens on
  the GUI thread (Rule 1); reactive's global mutex covers reads. Document
  this invariant in a comment on the Service type; do not add a second lock.

- **Rule 5 — timers.** The poll timer is a QTimer (GUI thread) but its
  handler only enqueues the refresh (Rule 2); the resulting Listen callbacks
  marshal back (Rule 1).

### State vs events (signals vs callbacks)

Split by kind; do not force one mechanism onto both:

- **State — values that change over time → reactive Signal/Computed.**
  `items` already is one; derived values become `Computed`s. **Exception:
  counts with their own subscribers** (`ActiveCount`) must be plain
  `Signal[int]` maintained by the writer — `Computed` invalidates its
  subscribers eagerly on every dependency `Set`, so a badge `Effect` over a
  `Computed[int]` re-runs on every progress tick even when the count is
  unchanged. The merge/SetItems path sets `ActiveCount` only when the value
  actually changed.
- **Events — things that happen once → callback registration.**
  `OnCompleted`/`OnFailed` stay callbacks. A Signal-as-event fires its
  Effect once at startup with the zero `Download` (you end up guarding
  that), and DeepEqual dedup collapses a repeated event (you end up
  guarding that too) — plumbing with extra steps. Each has exactly one
  consumer (main.go); an event bus for that is over-build.

### Errors

- Service/action errors → `notifications.Error("Pause", err.Error())` style
  (title + message), never a panic, never a silent swallow.
- Plugin failures are expected (missing interpreter, bad manifest): log via
  `slog`, skip the plugin, fall through to the next resolver. A broken plugin
  must never break resolving.

### Idioms

- Widgets: struct embedding `*qt.QWidget`, `NewXxx(parent, deps)`, callback
  registration `OnXxx(fn)`, theme constants, `ui.NewRButton/RTextField/RDialog/RSwitch`.
  Copy the style from `widget/components/downloads/` — do not invent new ones.
- Service callbacks the UI subscribes to follow the existing
  `OnChanged(fn) (dispose func())` shape.

---

## Source inventory

### Python reference (read-only)

| File | Role in this task |
|---|---|
| `rapid/main.py` | Wiring blueprint: construction order, badge (`activeCountChanged → setBadge`), browser bridge → dialog `openFromBrowser`, quit guard (`activeCount` → QuitConfirmationDialog), shutdown order |
| `rapid/backend/download/service.py` | Semantics to port into `services/download/service.go`: `_download`, `_notify` merge, `_doResolve` + `_browserResolvedUrl` + `_withRequestContext`, `_uniqueName`, pause/resume/stop/delete (incl. `_deleteFromDisk`), `start` (re-listen restored actives), `close`, `_poll` |
| `rapid/backend/plugin/manager.py` | Dir scan, manifest load, match-by-regex, first-match resolve with fallback |
| `rapid/backend/plugin/protocol.py` | JSON-RPC envelope + `rapid.ping` / `rapid.resolve` methods |
| `rapid/backend/plugin/transport.py` | QProcess stdio transport + the timeout ladder (start 3000 / request 8000 / write 2000 / terminate 2000 ms) — port the *behavior*, use `os/exec` not QProcess |
| `rapid/backend/plugin/{resolver,models}.py` | ResolverPlugin wrapper, PluginManifest / ResolvedItem models |
| `rapid/plugins/sampledemo/{plugin.json,plugin.py}` | Real manifest + plugin for manual testing |
| `rapid/qml/components/download/DownloadDialog.qml` | Dialog behavior: 300 ms debounce resolve, download-button enablement, submit() merging advanced opts + dir into resolved URIs, `openFromBrowser(request)`, Enter/Return shortcuts |
| `rapid/qml/components/download/KVEditor.qml` | Rows + JSON toggle editor for headers/cookies |
| `rapid/qml/components/download/AdvancedOptions.qml` | Accordion wrapping two KVEditors |
| `rapid/qml/components/download/QuitConfirmationDialog.qml` | Quit guard dialog: active names list, warning, Cancel / "Quit anyway" |

### Go targets (create or extend)

| Path | Step |
|---|---|
| `services/download/service.go` | S1–S5, S9 |
| `lib/worker/worker.go` (+ `worker_test.go`) | S2 |
| `db/migrations/00003_drop_updated_at_trigger.sql` | S3 |
| `services/download/helpers.go` (+ `helpers_test.go`) | S4 |
| `services/plugin/spec.go`, `protocol.go`, `transport.go`, `manager.go` (+ `_test.go`) | S6–S8 |
| `main.go` | S10, S17 |
| `widget/views/download.go` | S11, S16 |
| `services/notification/service.go` | S12 (one method) |
| `widget/components/downloads/kv_editor.go` | S13 |
| `lib/validate/validate.go` (+ `validate_test.go`) | S14 |
| `widget/components/downloads/dialog_model.go` (+ `dialog_model_test.go`) | S14 |
| `widget/components/downloads/download_dialog.go` | S15 |
| `services/clipboard/clipboard.go` | S16 |
| `widget/components/downloads/quit_dialog.go` | S18 |

---

## Nuances checklist (do not lose these)

**Merge semantics (`_notify` → S2).** When a downloader notification arrives
for gid G with the fresh `api.Download`:
- Keep fields aria2 does not know: `Resolved`, `Category` — take them from the
  previous in-memory/DB row when the fresh payload carries none (Python merges
  `previous.resolved/category/speed/completed/total`). But while status is
  `active`, aria2's fields are authoritative **including speed 0** — never
  preserve the previous speed for an active row (a stalled download must show
  0, not a stale number).
- Upsert to DB, then update the in-memory slice (replace by gid; keep
  newest-first order — new gids prepend).
- If status is `active` and speed > 0 → `addSpeedSample(ctx, gid, now, speed)`
  (once per notify; the store dedupes on (gid, ts)).
- **Status transitions fire user events exactly once**: previous status !=
  `complete` && new == `complete` → completed event; likewise `error` → failed
  event. The downloader's claim/deliver already guarantees exactly-once
  terminal *notification*; the service still guards the transition itself
  because poll refresh can deliver the same terminal status twice.

**Resolve ordering (`_doResolve` → S7).**
1. If `BrowserResolved` (bridge already did it): build `ResolvedURL` straight
   from the request (`_browserResolvedUrl`) — no subprocess, no aria2 probe.
2. Else: plugin chain first — first plugin whose manifest `match` regexes hit
   the URI wins; run `rapid.resolve`; map items to `api.ResolvedURL`.
3. **A plugin that claimed the URI and then failed is an error, not a
   fallback trigger** — surfacing the plugin's error beats an aria2 dry-run
   "resolving" the page HTML into misleading metadata. The aria2 fallback
   (`downloader.Resolve`, dry-run + HEAD probe) runs only when **no** plugin
   matched.
4. After resolving, apply `_withRequestContext`: merge request headers,
   cookies, referer, filename, title, mimeType, category onto every resolved
   item (request values win over plugin values for filename/title/category
   when present; plugin/probe values fill the gaps). Referer: `req.Referer`
   when set, else the page URL for browser requests (mirror
   `_withRequestContext` exactly). If the merged filename has no extension
   and `mimeType` maps to one (`mime.ExtensionsByType`), append it.

**Unique name (`_uniqueName` → S7).** Before download, if `dir/name` exists on
disk, try `name (1).ext`, `name (2).ext`, ... first free slot. Pure function
`uniqueName(dir, name string) string` + unit test (table-driven: collision,
no-extension name, multiple collisions). No such helper exists in Go yet.
Treat an existing `name.aria2` (aria2 control file) as a collision too — it
means a download of that name is in flight.

**Delete (S4).** Snapshot the row (files included) before enqueuing — the
queued op can't read in-memory state later. Then `downloader.Remove` +
`Purge` → `Unlisten(gid)` → optionally unlink every `file.Path` **and its
`path + ".aria2"` control file** (`_deleteFromDisk`; ignore not-exist
errors) → `removeDownload` from DB → drop from in-memory slice. A
gid-not-found reply from aria2 counts as success for cleanup (extend the
not-found matcher if it only covers one of aria2's phrasings); other errors
abort with a toast and keep the row.

**Start / re-listen (S2).** On `Start`: `downloader.Start(ctx)` (spawns or
adopts the daemon), then for every restored DB row that is still
active/waiting **and** the daemon still knows (GetStatus succeeds) →
`Listen` again. Rows whose gid the daemon forgot stay as-is for now — S5
tightens that to "mark error".

**Plugin manifest (S4).**

```json
{
  "id": "sampledemo", "name": "Sample Demo", "version": "1.0.0",
  "type": "resolver",
  "match": ["https?://(www\\.)?youtube\\.com/.*"],
  "transport": { "command": "plugin.py" }
}
```

- `transport.command` is relative to the manifest's directory.
- `type` must be `"resolver"`; anything else → skip with a log line.
- `match` is a list of regexes; compile once at scan time; invalid regex →
  skip plugin with a log line (a broken plugin must never break the app).
- Dirs: `settings.PluginDirs` in order, first match wins on duplicate id.
- `transport.command` that escapes its manifest dir via `..` (or an absolute
  path) → reject the manifest with a `slog.Warn` — a plugin must not
  execute arbitrary paths outside its own directory.

**Plugin protocol (S4/S5).** Line-delimited JSON-RPC 2.0 over the child's
stdin/stdout, one JSON object per line:
- Request: `{"jsonrpc":"2.0","id":1,"method":"rapid.ping","params":{}}` —
  expect a success response; used as a health probe.
- Request: `{"jsonrpc":"2.0","id":1,"method":"rapid.resolve","params":{"uri":"..."}}`
  → `result: {"items":[{"title":..., "url":..., "filename":..., "category":..., "mimeType":...}]}`.
- Response may carry `error`; empty items list is a valid "can't resolve".
- Ignore stderr (or capture a tail of it for error messages — nice to have).

**Plugin process model (S5).** Short-lived process **per request**: spawn →
ping (optional health gate) → resolve → terminate. No daemon, no pooling.
Go version of the timeout ladder, all via one context:

```go
ctx, cancel := context.WithTimeout(ctx, 8*time.Second) // request budget
defer cancel()
cmd := exec.CommandContext(ctx, ...)
cmd.WaitDelay = 2 * time.Second // grace after ctx kill, mirrors terminate-timeout
```

Write deadline: the params line is tiny; a 2 s `SetWriteDeadline` on the
stdin pipe (or just write inside the ctx budget) is enough. If you find
yourself writing more than ~60 lines for the transport, stop — it is a
`bufio.Reader.ReadString('\n')` loop and two pipes.

**DownloadDialog enablement (S14/S15).** The truth table lives in the
Qt-free `DialogModel` (S14): CanSubmit = URL non-empty && no resolve error
&& ≥1 resolved URI && not currently resolving && not in edit mode.
Enter/Return trigger download when enabled; Cancel is always enabled.
Debounce: 300 ms single-shot QTimer restarted on every URL edit — but
`OpenFromBrowser` never debounces (rows come from the request; filling the
editors must not trigger a re-resolve). Esc while a row editor has focus
closes the editor, not the dialog.

**Badge (S12).** Python: `downloader.activeCountChanged → setBadge`. Go:
`ActiveCount` is a `Signal[int]` maintained by the merge/SetItems path
(count of non-terminal, set only on change — see "State vs events" for why
not `Computed`), and main subscribes with one `reactive.Effect` →
`notifier.SetBadge(n)`.

**Animations (S9/S12).** The animation machinery already exists and is
battle-tested — reuse it, don't rebuild it:

- `widget/components/downloads/common.go` has `animateMaxHeight(w, from, to,
  duration, easing, done)` (layout-safe accordion pattern) and
  `animateOpacity(effect, from, to, ...)` (fade via QGraphicsOpacityEffect).
  Already used by the detail-expand (150 ms), Sidebar width, RSwitch knob
  (120 ms).
- Still to port, with exact QML durations:

| QML | Duration | Go |
|---|---|---|
| AdvancedOptions expand (height + opacity + y) | 100 ms | `animateMaxHeight` + `animateOpacity` (S15) |
| ResolverUri row appear/disappear (slide + fade) | 300 ms | height + opacity helpers on the row (S15) |
| ResolverUri loading spinner (`RotationAnimation`) | 1000 ms loop | `QVariantAnimation` loop + rotated-pixmap paint widget (S15) |
| DownloadPage list add/remove Transition (fade + x-slide) | 150 ms | height + opacity helpers on row wrappers (S11) |

- Substitutions, deliberately: QML `y`/`x` slides become maximumHeight
  animations (animating pos on a layout child fights the layout — the
  existing helper exists precisely for this). Same feel, no layout jitter.
- Rule: stop a running animation before starting its reverse on the same
  widget (overlapping height anims vibrate — documented on the helper).
- Resolver rows that swap on re-resolve chain their animations: run the out
  animation to completion (its `done` callback), then the in animation —
  never a fixed timer guessing when the out finished.

**Shutdown order (S10).** `bridge.Close` → `service.Close` (stop timer,
`downloader.Stop()` which also saves the aria2 session) → `db` close (already
a defer in main). SIGINT/SIGTERM path goes through the same requestQuit guard
as the tray Quit (S18).

---

## Step plan

### S0 — Preflight

Read first: `main.go`, `services/download/service.go`,
`services/download/downloader/downloader.go`, `rapid/backend/download/service.py`.

Verify:

```sh
task build && task test
```

Both must pass before anything else. Commit nothing.

---

### S1 — Fix existing reactive wiring defects

**Goal:** the read-only service/view layer is correct before building on it.

- **`FilterChanged`** (service.go) reads `Items` + `Counts`, so it fires on
  every progress tick. Rebuild it over the filter inputs only:

  ```go
  s.FilterChanged = reactive.NewComputed(func() bool {
      s.Category.Get()
      s.Search.Get()
      return true
  })
  ```

  Item-driven view updates already bind `Items` (`BindItems`); audit the
  `FilterChanged` consumer and confirm nothing relied on item churn.
- **Search casing**: `SetSearch` lowercases on write, destroying the raw
  query. Store the trimmed raw string; lowercase both sides inside
  `matches`.
- **Speed/ETA display gating**: rows show `FormatSpeed`/ETA while
  paused/queued (stale or "0 B/s"). Render the speed/ETA line only when
  `d.Status == "active"` — gate at the widget call site.
- **`store.Remove` returns its error** — service paths surface it through
  `notifyError` instead of discarding it.

Verify: `task build && task test`; manual: search "MP4" and "mp4" both
match; editing search re-fires filter UI but progress ticks don't; paused
rows show no speed line.

---

### S2 — `lib/worker` package + service deps/lifecycle

**Goal:** a reusable serial action queue, then `download.Service` uses it
for lifecycle.

**Part 1 — `lib/worker` (new, ~40 lines + tests, pure Go, no Qt):**

```go
package worker

func NewQueue(depth int) *Queue
func (q *Queue) Start(ctx context.Context, onError func(task string, err error))
func (q *Queue) Enqueue(task string, fn func(ctx context.Context) error) bool
func (q *Queue) Close()
```

- One drain goroutine started by `Start`; tasks run serially, in enqueue
  order, each with the worker ctx. A non-nil error → `onError(task, err)`
  — the single error-report point (callers marshal to GUI themselves).
- `Enqueue` is goroutine-safe (mutex + closed flag) and returns false after
  `Close`. Pending tasks on `Close` are abandoned, not drained — the only
  user quits the process right after.
- Tests (plain `go test`, no offscreen needed): order preservation,
  error reporting, enqueue-after-Close returns false, ctx cancel stops
  the loop.

**Part 2 — service deps + lifecycle:**

- Change `NewService()` → `NewService(cfg ServiceConfig)` with

  ```go
  type ServiceConfig struct {
      Settings  settings.Settings
      Downloader downloader.Downloader // aria2 implementation
      Fallback  downloader.Resolver    // aria2 probe resolver
      Plugins   *plugin.Manager        // nil-safe: nil = no plugins (until S6)
  }
  ```

- Add `Start(ctx context.Context) error`: `queue.Start(ctx, s.onTaskError)`
  where `onTaskError` marshals through `mainthread.Start` into
  `notifyError`; then downloader.Start + re-listen restored actives (see
  checklist). Add `Close()`: `queue.Close()` then `downloader.Stop()`.
- Keep `Refresh` behavior; nothing else changes yet.
- Satisfy the compile with a temporary nil-tolerant plugin field (S8 fills it).

Verify: `task build && task test` (worker tests included); app still
starts, list still loads (`task dev`, quit).

---

### S3 — Notify/merge path + store hardening

**Goal:** downloader notifications flow into DB + memory + user events.

- In `Start`, after re-listening, the service is the `NotifyCallback` owner:
  merge per the checklist (preserve Resolved/Category, upsert, speed sample,
  in-memory replace/prepend, exactly-once completed/failed transitions).
- Add `OnCompleted(fn func(api.Download)) (dispose func)` and `OnFailed(...)` —
  same shape as `OnChanged`.
- Every callback body starts with `mainthread.Start` (Rule 1).
- Reuse `downloadsEqual` (already in service.go) to skip no-op `SetItems`.
- Error callback (`ErrorCallback`) → log + `s.notifyError`.
- **Store hardening (same step):** wrap `upsertDownload` (select + write +
  `syncFiles`) in one `RunInTx` so a crash can't leave the row and its
  files disagreeing. Add migration `00003_drop_updated_at_trigger.sql`
  dropping `trg_downloads_updated_at` — it issues a second UPDATE per write
  in a timestamp format that disagrees with bun's, and nothing reads the
  column — and instead set `UpdatedAt: now` in the Go update path (drop
  `updated_at` from `ExcludeColumn`). Verify `addSpeedSample` prunes to
  `speedSamplesKeep`; add the prune in the same tx if it doesn't.

Verify: `task build && task test`; with `task dev` + `task seed`, starting a
real download from aria2 (or the seed harness) updates progress live.

---

### S4 — Actions + `helpers.go`

**Goal:** all list actions real.

- `Pause/Resume/Stop(ctx, gid)`, `Delete(ctx, gid, fromDisk bool)` per the
  checklist, `ActiveCount() *reactive.Signal[int]` (maintained by the
  writer, set only on change — see "State vs events"),
  `ActiveDownloads() []api.Download`,
  `DownloadName(d api.Download) string` (displayName helper from the UI-port
  nuances: `resolved.title || resolved.filename || basename(files[0].path) || gid`
  — if `widget` already has this, move it into the service and reuse).
- All control ops enqueue through the serial action queue (Rule 2) — pause →
  resume preserves user order at aria2. Delete's disk unlink also runs in
  the queued op (after purge), never on the GUI thread.
- Delete asks the downloader regardless of status (Remove is safe on terminal
  gids via `removeAny`).
- **`services/download/helpers.go` (new):** `SafeFileName(s string) string`
  (strips path separators and NULs — a plugin/title-derived name must never
  escape the download dir) and `Within(dir, path string) bool`
  (`filepath.Rel`-based containment check). `uniqueName` lives here too.
  Table-driven tests for all three.

Verify: `task build && task test`; add a service-level test with a fake
`Downloader` (tiny in-memory struct implementing the interface — keep it in
`service_test.go`, it is also reused by S5 and S9).

---

### S5 — Service hardening (independently scoped for review)

**Goal:** three correctness guards the Python app has; one session, one
commit, reviewable as a unit.

1. **Recent-action guard (stale-poll protection).** Keep
   `lastAction map[string]time.Time` (gid → last user action);
   Pause/Resume/Stop/Delete stamp it. The merge path skips applying
   notifications for gids acted on within `2 * PollIntervalMs` — an
   in-flight poll result must not overwrite the state the user just
   changed. A notify confirming the new status clears the stamp.
2. **Restore/re-listen tightening.** On Start, re-listen restored **active
   and waiting** rows; for each, GetStatus — unknown gid → mark the row
   `error` with "download not found in aria2" and persist it (replaces
   S2's "stay as-is").
3. **Listener error dedupe.** ErrorCallback storms (websocket drop /
   reconnect) spam toasts: dedupe on (gid, error text) within 30 s —
   first occurrence toasts + `slog.Warn`, repeats `slog.Debug` only.

Tests with the fake Downloader; inject a `now func() time.Time` for the
time windows — no real sleeping.

Verify: `task build && task test && task test:race`.

---

### S6 — Plugin spec + protocol types

**Goal:** pure data layer, no processes yet.

- `services/plugin/spec.go`: `Manifest` struct + `LoadManifest(dir string)
  (Manifest, error)` — reads `plugin.json`, validates id/type/match/transport,
  compiles match regexes. `Matches(uri string) bool`.
- `services/plugin/protocol.go`: request/response envelope types + `Ping()`,
  `ResolveRequest{URI}`, `ResolveResult{Items []ResolvedItem}` decode helpers.
  Line-delimited: `json.Encoder`/`json.Decoder` on the pipes handle this.

Verify: `task build && task test` with a table-driven manifest test using
`rapid/plugins/sampledemo/plugin.json` as fixture (copy it into
`services/plugin/testdata/`).

---

### S7 — Plugin transport + resolver

**Goal:** one resolve = one child process.

- `transport.go`: `runResolve(ctx, manifest, uri) ([]ResolvedItem, error)` —
  `exec.CommandContext` per the checklist (command resolved relative to the
  manifest dir; if `command` has no path separator, resolve via `PATH`
  **after** trying the manifest dir — e.g. `python3` must come from PATH,
  `plugin.py` from the dir). Ping is optional; skip it if it complicates —
  resolve failing covers the same ground. (`ponytail:` the ping probe exists
  in Python for QProcess lifecycle reasons that don't apply to os/exec.)
- `resolver.go`: `PluginResolver{manifest, items}` implementing
  `downloader.Resolver` (`ShouldResolve` = `manifest.Matches`,
  `Resolve(ctx, uri, opts)` = transport + map to `api.ResolvedURL`).
- Transport test: a `testdata/echo_plugin` helper — smallest possible program
  (can be a `//go:build ignore` Go file or a shell script) that reads one
  line and answers with a canned resolve result; plus a timeout test with a
  helper that never answers (assert ctx deadline ≈ 8 s — use a shortened
  timeout constant injected for tests rather than sleeping 8 real seconds).

Verify: `task build && task test`.

---

### S8 — Plugin manager

**Goal:** scan + chain.

- `manager.go`: `NewManager(dirs []string) *Manager` — scan every dir, load
  manifests, skip invalid with `slog.Warn`. Dedupe by id (first dir wins).
  Immutable after construction (no lock needed).
- `Resolve(ctx, uri, opts) (items []api.ResolvedURL, claimed bool, err error)`:
  iterate plugins in order; first `Matches` wins. `claimed=true, err!=nil`
  means a plugin matched and failed — the caller surfaces the error and
  **never falls back** (checklist §resolve). `claimed=false` when none match
  (caller falls back to aria2). `Len()` for a debug log line at startup.

Verify: `task build && task test` with a testdata dir containing one valid +
one broken manifest (broken is skipped, valid is found).

---

### S9 — Service resolve + Download

**Goal:** the `_doResolve` + `_download` port.

- `ResolveRequest` struct mirroring what the dialog/bridge provide:
  URL, Headers, Cookies `map[string]string`, Referer, Filename, Title,
  MimeType, Category, SavePath/Dir, Size, BrowserResolved.
- `Resolve(ctx, req ResolveRequest, cb func([]api.ResolvedURL, error))` —
  async by design (Rule 2): **own goroutine, not the action queue** (8 s
  plugin budget must not block a pause click; read-only, no ordering
  hazard). Runs the checklist chain (browserResolved → plugins → fallback →
  request-context merge), then marshals `cb` to the GUI thread.
- `Download(ctx, items []api.ResolvedURL)`: for each item — `uniqueName`,
  then a single **enqueued** op: `downloader.Download` → on success the
  worker marshals back to the GUI thread: upsert, **Listen + enqueued
  `Refresh(ctx, gid)` immediately** (the row shows real status instead of
  waiting a poll tick), prepend to memory (the S3 notify path handles
  subsequent updates). Multi-item downloads continue on per-item failure
  and report a summary error at the end — one bad item must not drop the
  rest.
- Unit tests with the fake Downloader + a fake Resolver: ordering (plugin
  before fallback), plugin-claimed-and-failed → error surfaced, not
  fallback; browserResolved short-circuit; merge precedence; uniqueName
  collision.

Verify: `task build && task test`.

---

### S10 — main.go composition

**Goal:** the `rapid/main.py` blueprint in Go.

```go
client := rpc.NewJSONRpc(s.Aria2Host, s.Aria2Port, s.Aria2Token)
dl := downloader.NewAria2Downloader(s, client, true /*manageDaemon*/, nil /*globalNotify unused for now*/)
plugins := plugin.NewManager(s.PluginDirs)
service := download.NewService(download.ServiceConfig{Settings: s, Downloader: dl, Fallback: dl, Plugins: plugins})
if err := service.Start(ctx); err != nil { /* fatal dialog or log+exit */ }
```

- Insert after navigation setup, before the poll timer would need it.
- Shutdown: on exit `bridge.Close(ctx)` (S17 adds the bridge) then
  `service.Close()` before the db defer runs.
- Replace the direct signal-handler `Quit` with a `requestQuit` func (S18
  fills in the guard; for now it just quits — keeps this step compilable and
  independent).

Verify: `task build`; `task dev` spawns/adopts aria2 (check the daemon
process + session file), quit leaves no orphan aria2.

---

### S11 — View wiring

**Goal:** delete every "not wired" stub in `widget/views/download.go`.

- `OnPause/OnResume/OnStop/OnRemove/OnDeleteRequested` → the S3 service
  methods (Remove callback gets `(gid, deleteFromDisk)` from the existing
  delete dialog — map to `Delete`).
- Poll timer handler: call `service.Poll(ctx)` — one method that enqueues
  the downloader refresh (Rule 2/5). The view must not import `downloader`;
  the wrapper keeps the layering and the poll serialized with user actions.
- List add/remove transitions (DownloadPage.qml `add`/`remove` Transition):
  150 ms fade + slide on row insert/remove. Layout-safe port: `animateMaxHeight`
  0→sizeHint + `animateOpacity` on the row wrapper when it appears, the
  reverse on remove (hide the row when the reverse finishes, then let the
  layout drop it). The QML x-slide becomes a height animation — animating
  `x` on a layout-managed child fights the layout (see the
  `animateMaxHeight` comment); the fade carries the transition visually.
- `banner.OnDownload(func(url string))` → open the dialog with the URL
  prefilled once S15 lands; until then keep opening the dialog (no behavior
  regression).
- `ponytail:` the diff-by-gid widget-row list is the right shape up to a few
  hundred rows; QListView + delegate painting is the upgrade path if the
  list ever needs thousands — not now (rows are interactive composite
  widgets; delegates would mean hand-painting them all).

Verify: `task build`; manual: pause/resume/stop/delete against a seeded list.

---

### S12 — Badge + completed/failed notifications

**Goal:** user-facing feedback.

- Add `func (s *Service) SetBadge(n int)` to `services/notification/service.go`
  delegating to the existing private tray `setBadge` (one line — the tray
  controller already handles Qt temp cleanup).
- In `main.go`:

  ```go
  service.OnCompleted(func(d api.Download) { notifications.Success("Download complete", service.DownloadName(d)) })
  service.OnFailed(func(d api.Download)  { notifications.Error("Download failed", d.ErrorMessage) })
  reactive.Effect(func() { notifications.SetBadge(service.ActiveCount().Get()) }) // Signal — see "State vs events"
  ```

  (Check the exact notification method names in `services/notification` and
  match them; the shape is right. The badge Effect needs no dispose — it
  lives as long as the process.)

Verify: `task build`; manual: finish a small download → toast + badge count.

---

### S13 — KVEditor

**Goal:** `widget/components/downloads/kv_editor.go` porting KVEditor.qml.

- `NewKVEditor(parent, title string) *KVEditor`.
- Rows: slice of `{key, value *qt.QLineEdit}` in a VBox; per-row ghost remove
  button (last remaining row clears instead of removing — QML
  `handleDelete`); "+" ghost button appends.
- JSON toggle: `ui.RSwitch` + monospace `qt.QTextEdit` in JSON mode;
  switching rows→JSON serializes `map[string]string` (2-space indent),
  JSON→rows parses and reports errors in a danger-colored label under the
  area (border turns danger like QML — theme has the colors). JSON values
  must be strings/numbers/bools — objects/arrays become a row-level error,
  not a crash. Header keys pass through `http.CanonicalHeaderKey` on
  rows→JSON. While the JSON is invalid, the switch refuses to toggle back
  to rows (the error label says why) and `Values()` returns empty.
- Public API: `SetRows(rows []KV)`, `Values() []KV` (trims, drops empty keys,
  empty when JSON invalid), `KV struct{ Key, Value string }`.
- No models, no MVC — plain widget rebuilds. Simplicity over QML fidelity.
- On JSON parse error: show `jsonError` text (QML shows the parse error
  message; `encoding/json`'s `SyntaxError` message is fine).

Verify: `task build`; a temporary debug use somewhere (or just run the app —
S12 consumes it next).

---

### S14 — Dialog validation model (minimal vuelidate-style)

**Goal:** the dialog's enablement/decision logic as a Qt-free, testable
model — S15 builds widgets on it. Independently scoped for review.

- `lib/validate` (new, ~30 lines, pure Go over reactive):

  ```go
  type Rule func() string                 // "" = valid, else error text
  func Valid(rules ...Rule) *reactive.Computed[bool]
  func FirstError(rules ...Rule) *reactive.Computed[string]
  ```

  Rules close over signals; calling them inside the `Computed` tracks the
  dependencies, so validity recomputes on any input change — the vuelidate
  shape (validators → computed errors) without a framework.
- `widget/components/downloads/dialog_model.go` (new; imports reactive +
  api + settings only, **no Qt**): holds `URL`, `Folder`, `Rows`,
  `Fetching`, `Editing` signals plus `Err`; exposes `CanSubmit()
  *reactive.Computed[bool]` per the checklist truth table, `Reset()` for
  reopen, and field validators built from `lib/validate` rules.
- **CategoryFromFilename**: static extension→category table (port the
  Python mapping verbatim) + table-driven test; the model defaults Category
  from the resolved filename when nothing else provides one.
- **Button semantics decided here, applied in S15/S18**: dialog Cancel is
  always enabled (it cancels the wait, not the process); quit dialog Cancel
  is the default button and "Quit anyway" is not (Enter-as-reflex must not
  lose data).

Verify: `task build && task test` — pure-Go tests for the truth table,
`FirstError`, and the category table (no offscreen needed).

---

### S15 — Full DownloadDialog

**Goal:** replace the 44-line stub with the real dialog.

Read first: `DownloadDialog.qml` (323 lines), `resolved_uri_panel.go` (the
readonly panel to reuse for display), `service.py::_doResolve` for the
request mapping.

Layout (top to bottom, `ui.RDialog` like DeleteConfirmationDialog):
1. URL field (`RTextField`) + paste-from-clipboard button (live-bound to
   the clipboard service — see S16).
2. Resolved URIs area: reuse `resolved_uri_panel.go` rows; make each row's
   title/filename/url editable (extend the panel or embed it — prefer
   extending with an `editable` flag over a parallel widget) + per-row
   remove; empty-state hint when nothing resolved yet. Rows animate in/out
   — 300 ms slide-up + fade, per ResolverUri.qml (the helpers cover the
   fade; the slide is the same maximumHeight trick on the row widget).
   Re-resolve swaps chain out→in on the animation's `done` callback — no
   fixed timers (checklist).
3. Status line: resolving spinner + text / resolve error (danger color).
   The spinner is QML `RotationAnimation` 0→360 over 1000 ms, looping
   while resolving — a tiny custom widget: `qt.QVariantAnimation` looping
   0→360, `OnValueChanged` sets an angle field and calls `Update()`; paint
   rotates the icon pixmap by that angle (translate to center, rotate,
   translate back). Delete the animation on widget destroy.
4. Folder row: read-only field + browse button →
   `qt.QFileDialog_GetExistingDirectory(parent, "Choose folder", current)`
   (native, replaces zenety/kdialog entirely). Default `settings.DownloadDir`.
5. Advanced options accordion: ghost toggle button + two `KVEditor`s
   (Headers, Cookies), animated with the existing helpers — 100 ms
   `animateMaxHeight` on the content widget + 100 ms `animateOpacity`
   (QML Behavior durations; `OutCubic` reads closest to the QML default).
   Stop a running animation before starting its reverse — see the
   `animateMaxHeight` doc comment about overlapping height anims.
6. Footer: Cancel + Download (`ui.NewRButton` primary).

Behavior:
- Widgets bind to the `DialogModel` (S14); the Download button's enabled
  state is `model.CanSubmit` — no hand-rolled truth table in the dialog.
- URL edit → restart 300 ms single-shot QTimer → `service.Resolve` with the
  advanced values merged into the request → swap rows (clear stale error).
- **Resolve supersession.** The debounce only delays *starting* a resolve; it
  does not cancel one already running. Keep a `resolveID uint64` in the
  dialog, increment it on every resolve fire, and have the callback drop
  results/errors whose ID is no longer current — a slow older resolve must
  never paint over the newer one. A superseded `context.Canceled` is not a
  user-facing error. `OpenFromBrowser` bypasses the debounce and must also
  invalidate any pending resolve.
- **Cancellation lives in the dialog, not the downloader.** Keep the current
  request's `context.CancelFunc`; cancel it before starting a new resolve and
  on dialog close. Then delete `pendingCancel`/`pendingGen` and the gen guard
  from `Aria2Downloader.Resolve` — it becomes
  `rctx, cancel := context.WithCancel(ctx); defer cancel()` around the probe.
  Keep `resolverSem` (defensive concurrency cap, not supersession) and keep
  `onResolved`'s `context.WithoutCancel` cleanup, rewording its comment to
  say a superseded resolve canceled the parent ctx. Land the removal in this
  step so cancellation never disappears between steps.
- Download enabled per the checklist; Enter/Return triggers it. Cancel is
  always enabled (S14). Esc while a row editor has focus closes the editor
  only — give the editor a `Qt.WidgetShortcut`-scoped Esc shortcut so it
  never reaches the dialog's default button.
- Submit: merge folder + advanced headers/cookies onto every resolved item,
  `service.Download(items)`, close dialog.
- `OpenFor(url string)` (banner/tray path) and `OpenFromBrowser(req
  browser.BrowserRequest)` (S17): the browser variant prefills URL, folder
  (`SavePath`), advanced headers/cookies/referer, and marks the request
  `BrowserResolved` — and never triggers the resolve debounce (rows come
  from the request itself).
- Singleton: construct once in the download view's deps; `Open*` resets +
  shows. Python used a singleton dialog; keep that (state persistence across
  opens is a feature: last folder sticks).

Verify: `task build`; manual: paste a URL → resolves → download lands in the
list and progresses; advanced headers reach the request (check aria2 via
`task seed` tooling or log).

---

### S16 — Clipboard service + wiring

**Goal:** `ExtractURL` becomes a live, watched service. The clipboard is
**state**: `text` and `url` update on every system clipboard change (any
app), and every consumer binds to them — nothing polls, nothing subscribes
once. Port of `rapid/backend/clipboard/` (read it first — it's ~50 lines).

- `services/clipboard` service, state as reactive cells per "State vs
  events":

  ```go
  type Service struct { text *reactive.Signal[string]; url *reactive.Computed[string] }
  func NewService() *Service   // QGuiApplication clipboard: ConnectDataChanged → text.Set(current)
  func (s *Service) Text() *reactive.Signal[string]   // raw clipboard text
  func (s *Service) URL() *reactive.Computed[string]  // ExtractURL(text); "" when none
  func (s *Service) Copy(text string)                 // clipboard SetText — also the dismiss path
  ```

  - `DataChanged` arrives on the GUI thread (Qt event loop) — set the
    signal directly, no marshaling. `url` as a Computed over `text` means
    the regex runs once per actual change and dedup is free.
  - This mirrors the Python service exactly: `dataChanged → textChanged →
    urlChanged`, `url` = first `https?://\S+` match, `copy(text)`.
- **Banner** (live binding, not a one-shot): `reactive.Effect` →
  `banner.SetVisible(clipboard.URL().Get() != "")`. Copying a URL anywhere
  on the system shows it; copying non-URL text or empty hides it — with no
  per-consumer watch code.
- **Banner dismiss = a `dismissed` Signal, not a clipboard wipe** (deliberate
  change from Python, which cleared the whole clipboard — that destroys the
  user's clipboard content): the banner keeps `dismissed *reactive.Signal[bool]`;
  its visibility Effect reads `clipboard.URL().Get() != "" && !dismissed.Get()`;
  the ✕ sets `dismissed=true`. A second Effect over `URL` resets
  `dismissed=false` whenever the URL changes (skip the first run) — a new
  copy re-arms the banner without touching the clipboard.
- **Banner Download** → `dialog.OpenFor(clipboard.URL().Peek())`
  (prefills from the current url at click time — DownloadPage.qml:67).
- **Dialog paste button** (S15 item 1): enabled iff
  `clipboard.Text().Get() != ""` via an Effect (live while the dialog is
  open — copy something with it open and the button lights up); click →
  set the URL field to `Text().Get()` (raw text — the resolve debounce
  then runs, Python line 90 does the same).
- **Context menu "Copy URL"** (DownloadItemContextMenu.qml:44) →
  `clipboard.Copy(d.Resolved.URL)` with the nil-guard.
- Known behavior: context-menu Copy URL re-triggers `dataChanged` and sets
  `url` — the banner reappears only when it is a different URL than the
  dismissed one (the reset Effect keys on change). Don't add self-copy
  suppression; note it in review if it annoys.

Verify: `task build`; manual: copy a URL in any other app → banner appears
(without focusing Rapid); copy plain text → banner hides; banner ✕ hides it
while the clipboard still pastes the URL; copying a different URL re-arms
the banner; with the dialog open, copy text → paste button lights up live;
Copy URL from the context menu → banner reappears only for a new URL.

---

### S17 — Browser bridge wiring

**Goal:** extension downloads land in the dialog.

In `main.go` after service start:

```go
bridge := browser.NewBridge("127.0.0.1", 17654)
bridge.OnDownloadRequested = func(req browser.BrowserRequest) {
    mainthread.Start(func() {
        window.ShowNormal() // bring to front, per Main.qml
        window.Raise()
        dialog.OpenFromBrowser(req)
    })
}
bridge.OnError = func(msg string) {
    mainthread.Start(func() { notifications.Error("Browser integration", msg) })
}
if err := bridge.Start(ctx); err != nil { slog.Warn(...) } // don't fatal: port busy ≠ dead app
```

- The HTTP handler must answer immediately: marshal through
  `mainthread.Start` and return — the 202 goes out before the dialog work
  runs. Never wait on the GUI side inside the handler goroutine.

The dialog reference needs to reach main — thread it through the view deps
(the view already owns dialog construction per S15; expose an opener func on
the view or pass a closure when constructing the view). Pick whichever is the
smaller diff.

Verify: `task build`; manual: `curl -X POST` a BrowserRequest-shaped JSON to
`http://127.0.0.1:17654/downloads` (check `bridge.go` for the exact route +
field names) → window comes forward, dialog prefilled, BrowserResolved path
skips resolving.

---

### S18 — Quit guard + dialog

**Goal:** never silently kill active downloads.

- `widget/components/downloads/quit_dialog.go` porting
  QuitConfirmationDialog.qml: title "Quit Rapid?", active names list (use a
  `qt.QLabel` per name or one label with newlines — no HTML needed, plain
  list is fine; still run each name through `html.EscapeString`, QLabel
  rich-text auto-detection would eat filenames containing `<`), the
  resume/expiry warning sentence, Cancel + "Quit anyway" (danger variant
  button if `ui.RButton` supports one — check; else default). Cancel is the
  default button and "Quit anyway" is not (S14).
- `main.go`:

  ```go
  requestQuit := func() {
      if service.ActiveCount().Peek() == 0 { qt.QApplication_Quit(); return }
      quitDialog.SetNames(namesOf(service.ActiveDownloads()))
      quitDialog.Open()
  }
  ```

  Wire tray Quit + SIGINT/SIGTERM to `requestQuit` (replace the current
  direct `mainthread.Start(app.Quit)`). "Quit anyway" → `QApplication_Quit`.
- If the tray is unavailable (creation failed), the main window's close
  must route through `requestQuit` instead of hiding — hide-to-tray with no
  tray icon loses the window.
- Exit path (after Exec returns): bridge.Close → service.Close → db defer.
  App-close on the main window still just hides to tray (already done).

Verify: `task build`; manual: with an active download, tray Quit → dialog
with the name; Cancel keeps running; Quit anyway exits and aria2 session is
saved (check the session file mtime). Without actives → instant quit.

---

### S19 — Final sweep

- `rg -n "not wired|TODO(full-wire)|not ported" widget/ services/ main.go` —
  every hit is either fixed or explicitly re-scoped with a comment saying why.
- Full matrix:

```sh
task build && task vet && task test && task test:race
```

- Manual pass: add URL download / browser request / clipboard banner /
  pause-resume-stop-delete / quit guard / badge / toast on complete+fail /
  restart with active downloads (re-listen restores them) / animations
  (accordion 100 ms, resolver rows 300 ms + spinner, list add/remove 150 ms
  — no layout jitter).
- Update this file's status or delete it once accepted.

---

## Code-quality suggestions (apply during implementation)

These are decisions already made — follow them instead of re-deriving:

1. **No new locks in the service.** The GUI-thread invariant (Rule 1/4)
   replaces a mutex. Write the invariant as a doc comment on `Service` so the
   next reader doesn't "fix" it. If any code path can't prove it's on the GUI
   thread, wrap it in `mainthread.Start` rather than adding a lock.
2. **One marshal helper, not ten.** Every callback body starting with
   `mainthread.Start(func() {...})` is fine — it reads clearly and miqt owns
   the helper. Don't build a `postToGUI` wrapper on top of it.
3. **`service.Poll(ctx)` beats view→downloader reach-through.** The view
   shouldn't import `downloader`. One method, one line, keeps the layering.
4. **Fake Downloader in `service_test.go`** (S4) — ~30 lines, reused by S5
   and S9.
   This is the only test seam the service needs; don't mock-store, the store
   tests already exist.
5. **`uniqueName` as a pure function** with the collision tests in the same
   file. It's the kind of helper that silently regresses.
6. **Plugin manager immutable after scan.** No hot-reload (Python didn't have
   it either — verify against `manager.py` and drop if unsure). Concurrency
   safety by construction, zero locks.
7. **KVEditor without models.** Slice-of-rows + layout rebuild. If you're
   reaching for `QAbstractTableModel`, stop — the dialog has ≤ a handful of
   rows.
8. **Animations reuse `common.go` helpers** — never write a new
   QPropertyAnimation dance inline. If a transition needs something the two
   helpers don't cover (e.g. the spinner rotation), it gets its own small
   helper next to them, with the same Stop-before-reverse discipline.
9. **`SetBadge` as a one-line passthrough** on notification.Service — do not
   restructure the tray controller, it already handles Qt temp-file cleanup.
10. **Bridge start failure is a warning, not fatal.** Port 17654 busy means
    another Rapid instance (or a stale one); the app still works minus the
    extension. Log + toast, continue.
11. **`slog` everywhere for background failures** (plugin skips, re-listen
    misses, speed-sample write errors) with gid/context in the message; user
   -facing toasts only for things the user caused or must act on.
12. **Signals for state, callbacks for events.** Don't convert
    `OnCompleted`/`OnFailed` to Signals: the Effect zero-value initial fire
    and DeepEqual dedup both need guarding — worse plumbing. Don't compute
    derived state (badge) in an `OnChanged` callback either — make it a
    `Computed` and let the reactive graph do the notifying, **except**
    subscriber-facing counters like `ActiveCount`, which are writer-maintained
    `Signal`s (eager `Computed` invalidation would re-fire consumers on every
    items `Set`).
13. **`lib/worker.Queue`, not per-click goroutines.** Ordering is a
    correctness requirement (pause→resume), not a nicety. The queue lives in
    its own package because it is generic machinery; the service stays a
    user of it. If you are about to add a second queue or a per-gid queue,
    don't — one worker covers human-speed input; aria2 ctx timeouts bound
    each op so the queue can't wedge forever.

---

## Verification summary (every step)

```sh
task build        # compiles
task vet          # static checks
task test         # offscreen unit tests
task test:race    # before S3, S5, S9, S19 at minimum
```

Manual `task dev` checks are listed per step; do them as you go — a step
whose manual check fails is not done, even if tests pass.
