# Qt Integration: wire the ported backend into the Widgets UI (miqt)

> Prerequisite: `tasks/backend-port/task.md` phases P1–P8 complete. That task deliberately exports
> pure Go logic with typed callbacks and **zero** `miqt` imports. This task consumes it from the
> Qt-Widgets layer (`main.go`, `widget/`, `service/`).
> Python sources of truth for behavior: `rapid/main.py` (composition order), `rapid/services/download/service.py`
> (the `QAbstractListModel` roles + `DownloadFilterProxy`), `rapid/services/clipboard/service.py`,
> `rapid/services/notification/service.py`, `rapid/services/plugin/transport.py` (`QProcess` timeouts).

Already exists (reuse, do not reinvent):
- `main.go`: window setup, tray + menu, stacked navigation, download dialog. Currently uses stub/demo
  views — the download view (`widget/views/download.go`) is still a component demo, not backed by data.
- `service/notification/`: `NewService(Options)`, `SetTrayMenu`, `OnTrayActivated`, `OnOpen(fn)`,
  `Error/Success/Info(title, msg, shouldNotify)`, `Close()`, badge via `Manager`.
- Backend callbacks to consume: `OnAdded/OnChanged/OnRemoved/OnCompleted/OnFailed/OnError/OnResolved/
  OnCountsChanged/OnActiveCountChanged` (download service), `OnDownloadRequested` (browser bridge).
  Pure helpers to reuse: `FilterDownloads`, `FormatSize`, `UniqueName`, `ExtractURL`.

## Q1 — Download list model (replaces `QAbstractListModel` + roles)

- Python exposed one `Download` per row with roles per dataclass field + synthetic `progress`, and
  `files`/`resolved` roles as dicts. In Widgets there is no `ListView`/role lookup — build the thinnest
  thing the views need: a `DownloadsModel` (new file under `widget/components/downloads/` or
  `widget/models/`) holding `[]download.Download` (newest-first, mirroring service order) with
  `Row(i)`, `Len()`, and `Refresh()` pulling from the service; subscribe to `OnAdded/OnChanged/
  OnRemoved` for incremental updates instead of full reloads.
- Decide with the views author whether updates go through `QTimer`-polled refresh or direct
  callback-driven widget rebuilds (miqt widgets are **not** thread-safe — backend callbacks fire on
  poller/resolve goroutines, so marshal to the GUI thread; check how miqt exposes
  `QTimer::singleShot`/queued invocation and use that as the hop).
- `progress` computation stays `Download.Progress()` (no QML `Number(...)||0` coercion needed).

## Q2 — Category filter + search (replaces `DownloadFilterProxy`)

- Reuse backend `FilterDownloads(downloads, category, search)` for matching logic; wire the existing
  sidebar (`widget/components/sidebar.go`) selection and header search field
  (`widget/components/header.go` `OnTextChanged`) to re-filter and rebuild the list.
- No `QSortFilterProxyModel` in Widgets — filtering is a Go slice operation followed by widget updates.

## Q3 — Resolve → dialog → download flow (replaces QML context properties + slots)

- Python flow: `Clipboard.url` prefill → `DownloadService.resolve/resolveRequest` (background thread) →
  `resolved` signal → `DownloadDialog.resolvedUris` → `DownloadService.download(list-of-dicts)`.
- Go flow: dialog (`widget/components/downloads/download_dialog.go`) calls `service.Resolve(url)` /
  `service.ResolveRequest(browser.BrowserRequest)`; `OnResolved(uris []ResolvedURL, errs)` populates the
  dialog with **structs** (no dict round-trip); confirm calls `service.Download(ctx, []ResolvedURL)`.
- `speedHistory`: expose `service.SpeedHistory(ctx, gid) []SpeedSample` to whatever draws the
  throughput graph (check `widget/components/downloads/layout.go` for the current placeholder).
- `pickFolder`: backend `PickFolder` tries zenity/kdialog and falls back to a path string. Decide here
  whether to keep that or use a native `QFileDialog::getExistingDirectory` via miqt
  (Python preferred Qt dialog last-resort; in Widgets the native dialog is cheap — prefer it and
  demote backend `PickFolder` to fallback/headless use).

## Q4 — Clipboard (replaces `ClipboardService(QObject)`)

- Backend ships `ExtractURL(text)` only. Here: miqt `QClipboard` wiring — `Copy(text)` via
  `QGuiApplication::clipboard()->setText`, live `text`/`url` properties replaced by reading clipboard
  on dialog open (or `dataChanged` subscription if miqt exposes the signal).
- Prefill the download dialog input with `ExtractURL(clipboard text)` (mirrors QML `Clipboard.url`).

## Q5 — App composition (replaces `rapid/main.py`)

Port `_run()` ordering to `main.go` (keep current window/tray/navigation setup, replace demo wiring):
1. `settings.Default(baseDir)` → `db.Open(dsn, migrationsDir)` → `store` → `Aria2Downloader` →
   `DownloadService.New(ctx, ...)` → `BrowserIntegration` bridge.
2. Tray menu keeps Open / New download / Quit; badge: `OnActiveCountChanged → notifier` badge update
   (Python: `activeCountChanged.connect(notifications.setBadge)`).
3. Notifications: `OnCompleted → notifier.Success(...)`, `OnFailed → notifier.Error(...)`,
   bridge errors → `notifier.Error("Browser integration", ...)`.
4. Browser bridge: `OnDownloadRequested → service.ResolveRequest` (currently the Python signal fed QML,
   which called back into `resolveRequest` — in Go connect it directly; keep a hook for the dialog
   preview if the UI wants confirmation first).
5. Quit gating: `requestQuit()` — if `ActiveCount() > 0`, show confirmation listing
   `ActiveDownloads()` names; else quit. (Python `QuitConfirmationDialog.qml` → Widgets dialog.)
6. Startup: `service.Start(ctx)` + `bridge.Start(ctx)` after UI shows; shutdown (reverse):
   `bridge.Close(ctx)` → `service.Close()` → `notifier.Close()` → `db.Close()`.
   Handle `SIGINT`/`SIGTERM` → graceful shutdown (Python needed a 1s `QTimer` signal pump because the
   event loop never yielded; verify whether Go/miqt needs the equivalent or `signal.NotifyContext`
   suffices).
7. Daemon lifecycle: `manageDaemon=true` in prod; `service.Start` returning error must surface
   (dialog or log) rather than silently continuing.

## Q6 — Verification

- `go build ./... && go vet ./...`; existing widget tests keep passing (`widget/...` has
  `qt_test.go`/`*_test.go` — check what display they need; backend tests stay display-free).
- Manual pass: new download via dialog (plugin + fallback resolve), pause/resume/stop/delete,
  persistence across restart (aria2 re-adopt + relisten), browser extension POST → download,
  tray badge/counts, quit-with-active dialog, clipboard prefill.
- Delete nothing under `rapid/` until this manual pass is green; then propose Python removal separately.

Status: [ ] Q1 list model · [ ] Q2 filter/search · [ ] Q3 resolve flow · [ ] Q4 clipboard · [ ] Q5 composition · [ ] Q6 verification
