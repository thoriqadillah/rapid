# Backend Port: Python → Go (pure logic, no Qt)

> Scope: port **every** backend functionality from `rapid/services/` (plus `rapid/services/setting/`,
> `rapid/services/seeder/`, plugin sample, browser bridge contract) into idiomatic Go packages.
> **Explicitly out of scope:** Qt/Widgets integration — the `QAbstractListModel` roles,
> `QSortFilterProxyModel`, `QQmlApplicationEngine` context properties, `QProcess`, Qt clipboard/tray,
> `QStandardPaths`, `QTimer`-driven Qt signals, and `main.py` composition all move to
> `tasks/qt-integration/task.md`. Where Python uses Qt only as a transport
> (threads, timers, signals, processes), port the **logic** with stdlib goroutines/channels/`os/exec`/
> `net/http` and expose callbacks the `widget/` layer subscribes to in the Qt-integration phase.
> Do NOT import `github.com/mappu/miqt` in the new backend packages
> (exception: `backend/settings`, which uses `QStandardPaths` so paths match
> the widgets layer — call `Settings.Default` after `SetApplicationName`).

Already exists (reuse, do not reinvent):
- `go.mod` module `rapid`, Go 1.26.8, deps: `bun`, `sqlitedialect`, `sqliteshim`, `goose/v3`, `miqt`.
- `db/db.go`: `Open(dsn)`, `OpenMemory(ctx)` (tests), `Close()`, `DB()`, migrations
> embedded via `go:embed` (`goose.SetBaseFS`, no dir args),
  FK enforcement via `_pragma=foreign_keys(1)` DSN param. All new store code takes a `*bun.DB`.
- `db/migrations/00001_create_downloads.sql`: `downloads`, `download_files`, `file_uris`,
  `speed_samples WITHOUT ROWID`, indexes, FK `ON DELETE CASCADE`. Do NOT hand-write schema in Go;
  schema changes = new goose migration files.
- `service/notification/` (Qt tray/popup) and `main.go` wiring: already ported, skip.
- Python sources of truth: `rapid/services/download/{models,downloader,aria2_downloader,store,service}.py`,
  `rapid/services/plugin/{protocol,models,transport,resolver,manager}.py`,
  `rapid/services/{database/database,setting/models,browser_integration/service,clipboard/service,
  notification/service,seeder/{seeder,download_seeder,__main__}}.py`,
  `rapid/services/{__init__,download/__init__,plugin/__init__,browser_integration/__init__}.py`,
  `rapid/main.py`, `rapid/plugins/sampledemo/{plugin.json,plugin.py}`,
  `browser-extension/src/background.js` (bridge client contract),
  `tests/test_{store,plugin_manager,browser_integration,aria2_downloader,download_service}.py`
  (behavioral spec — port the assertions to Go tests).

## 0. Conventions (applies to all phases)

- ORM: `github.com/uptrace/bun` only. Nullable columns → **pointers** (`*string`, `*int64`,
  `*bool`); `nullzero` was evaluated and is **rejected**: it maps Go zero values to SQL NULL,
  which destroys two load-bearing distinctions — `downloadSpeed=0` is a real value that must
  overwrite (`test_upsert_zero_speed_is_kept`), and the store merge needs presence tracking
  ("only non-nil overwrites", Phase 2). Plain values + `nullzero` cannot tell "aria2 omitted
  this field" from "field is zero". PK/FK/id columns (`gid`, `file_id`, `index`) stay plain
  values. `resolved_url` JSON column → `ResolvedURL` struct implementing
  `driver.Valuer`/`sql.Scanner` (bun marshals it as JSON automatically); `created_at`/
  `updated_at` → `time.Time` UTC (`time.Now().UTC()`).
- JSON, not `fromPayload`/`asDict`: Python's `fromPayload`/`asDict`/`fromDict` exist only to
  bridge two worlds — aria2's string-encoded numbers (`"totalLength": "100"`) and QML's
  `QVariantMap` roles. In Go both disappear: structs carry `json:"totalLength"` tags and a custom
  `UnmarshalJSON` (aria2 payloads) coerces string-or-number via helpers in `lib/transform.go`;
  bun handles the DB JSON column; the widget layer reads Go structs directly (no QVariantMap
  round-trip, no `Download(dict)` slot — see qt-integration task). Do NOT port `fromPayload`/
  `asDict`/`fromDict` as methods.
- Shared coercion lives in `lib/transform.go` (new): the `_str`/`_toInt`/`_toBool` semantics
  from `models.py` as unexported-or-small-exported helpers used by the custom unmarshalers
  (nil/bool→nil; string/float→parse else nil; string bool is `"true"`-only). Check `lib/`
  first — `lib/categories.go` (`Category` type) already exists; reuse `lib.Category` for all
  category fields instead of plain strings.
- `context.Context` is threaded through **every** blocking call: all `Store` methods
  (`ctx` is also required by bun queries), all `Downloader`/`Resolver` methods, `Transport.Request`,
  `Aria2RPC.Call`, `probeHeader`, the websocket read loop (`coder/websocket` is ctx-native:
  `Dial(ctx, url, opts)`, `Read(ctx)`), daemon wait loops, and browser server `Start/Shutdown`.
  Follow stdlib convention: `ctx` first arg; callers pass `context.Background()` at the top level
  (service ticker) and tests pass `context.Background()` or `t.Context()`. Never `context.TODO()`
  in new code.
- Migrations: `github.com/pressly/goose/v3` + `db/migrations/*.sql` only. Never `CREATE TABLE` in Go.
- Errors: sentinel errors (`var ErrX = errors.New(...)`, `Aria2Error`, `PluginError`, `TransportError`,
  `BrowserRequestError`) with `%w` wrapping; never swallow (`LOG.exception` → return error / error callback).
- Concurrency: `context.Context` cancellation + `sync.Mutex` + goroutines. No global singletons
  (Python `Database._default` pattern ends here — inject `*bun.DB` / `Settings` explicitly).
- Qt signal replacements: struct fields like `OnAdded func(gid string)`, `OnChanged func(gid string)`,
  `OnRemoved/OnCompleted func`, `OnFailed func(gid, msg string)`, `OnError func(msg string)`,
  `OnResolved func(uris []ResolvedURL, errs map[string]string)`, `OnCountsChanged func()`,
  `OnActiveCountChanged func(n int)`, `OnDownloadRequested func(req BrowserRequest)`.
  Callbacks carry **typed structs, not `map[string]any`** (no `asDict` round-trip — see above).
  Emit synchronously where Python did (protected by mutex where concurrent); document thread-safety.
- Tests: each phase ships `*_test.go` using `db.OpenMemory(ctx)` (schema comes from the embed)
  mirroring the Python test files above. Must run headless: `go test ./backend/...` with no display,
  no aria2c (mock RPC/transport; only optional integration test gated by `which aria2c`).
- New deps: exactly one — `github.com/coder/websocket` for the aria2 event socket. Everything else
  is stdlib (`net/http`, `mime`, `os/exec`, `regexp`, `encoding/json`). Do NOT add another sqlite
  driver (shim already present) or a gorilla package.

Suggested layout (packages own their models; store overwrites whole rows,
service merges against previous state before persisting):
```
lib/transform.go                  # shared coercion + extension helpers
backend/download/api/api.go       # Download, DownloadFile, FileURI, ResolvedURL, SpeedSample (plain values)
backend/download/downloader/      # Downloader + Resolver interfaces, Aria2Downloader, aria2 DTOs
backend/download/rpc/             # Aria2RPC transport (raw bytes, envelope errors as Go errors)
backend/download/api/api.go       # Download, DownloadFile, FileURI, ResolvedURL, SpeedSample (plain values)
backend/download/downloader/      # Downloader + Resolver interfaces, Aria2Downloader, aria2 DTOs
backend/download/rpc/             # Aria2RPC transport (raw bytes, envelope errors as Go errors)
backend/download/store.go       # bun row funcs (ctx on every call) + file sync
backend/download/transform.go     # row -> api converters, one per model
backend/download/seed.go          # fixture seeder (actives get speed samples)
backend/download/service.go     # pure DownloadService logic (no Qt; typed callbacks; model binding is qt-integration)
backend/plugin/protocol.go      # PING/RESOLVE, encodeRequest, decodeResponse, JsonRpcError
backend/plugin/models.go        # StdIOTransportSpec, PluginSpec (+ Transporter factory)
backend/plugin/transport.go     # Transport iface + ProcessTransport (os/exec, ctx)
backend/plugin/resolver.go      # ResolverPlugin
backend/plugin/manager.go       # PluginManager (scan/load/resolve)
backend/browser/bridge.go       # BrowserRequest type + normalize + HTTP bridge server
backend/settings/settings.go    # Settings struct + Default(baseDir)
backend/seeder/seeder.go        # Seeder iface + SeederService + DownloadSeeder (+ samples + speed curve)
backend/clipboard/clipboard.go  # ExtractURL helper (Copy/Text/live-signal stay in widget/; see qt-integration)
```

---

## Phase 1 — `lib/transform.go` + `backend/download/models.go` (port `download/models.py`, 221 lines)

Usage audit (why no `fromPayload`/`asDict` — verified by grep, do not reintroduce):
- `Download/FileUri/DownloadFile.fromPayload`: sole caller is `Aria2Downloader.getStatus`
  (aria2 `tellStatus` JSON). In Go this is `json.Unmarshal` with a custom `UnmarshalJSON` on
  `Download` that coerces aria2's string-numbers — no intermediate `map[string]any` + manual convert.
- `ResolvedUrl.fromDict`: callers are the store (DB JSON column — bun unmarshals directly into the
  struct) and the `download(list)` slot (QML echoing back the `resolved` signal payload — an artifact
  of the QML boundary that disappears when both sides are Go; the service keeps the struct).
- `asDict`: callers are `service.data()` role values, the `resolved` signal payload, and the
  `speedHistory` slot — all QML-boundary serialization. The Widgets layer reads Go structs directly.

What to build:
- `lib/transform.go`: aria2's JSON is loosely typed — numbers arrive as **strings**
  (`"totalLength": "100"`), occasionally as numbers, sometimes absent; `DownloadFile.selected`
  arrives as `"true"`/`"false"` strings. Standard `encoding/json` into `int64`/`bool` fails on
  those, so this file provides flexible scalar types with lenient `UnmarshalJSON` (accept
  string-or-number-or-null → pointer semantics: absent/null/garbage → nil):
  string-coerced int (the `_toInt` semantics: bool→nil, parse-failure→nil), `"true"`-only bool
  (the `_toBool` semantics), and string-or-nil (the `_str` semantics: non-string→nil).
  Domain structs use these via `json:"totalLength"` tags plus a top-level `UnmarshalJSON` only
  where cross-field defaults are needed (`index` missing→0, `gid` missing→`""`, `category`
  missing→`lib.Unknown`, non-array `files`/`uris`→empty). No `_num`/`_boolStr` encode direction —
  that direction no longer exists.
- Structs with `json:"camelCase"` tags matching the aria2/QML key names exactly:
  `status,dir,category,totalLength,completedLength,downloadSpeed,connections,numPieces,
  pieceLength,verifiedLength,errorCode,errorMessage,files,uris,index,path,length,completedLength,
  selected,uri,url,title,filename,mimeType,size,headers,cookies,referer,resolverName`
  (camelCase — NOT snake_case). `DownloadFiles[].index` missing/invalid → 0 (keep the
  `or 0` fallback); `gid` missing → `""`.
- `Download.UnmarshalJSON`: decode leniently via the `lib/transform.go` scalars
  (string-or-number numerics, `"true"`-only `selected`, `uris`/`files` only when arrays of objects,
  else empty) so a short/malformed aria2 reply degrades to nil fields, never errors.
- `ResolvedURL.Category` is `lib.Category`, defaulting to `lib.Unknown` when absent.
  `Size` is `*int64`. `SpeedSample{TS, Speed int64}` (plain values — never merged, never null).
- `Progress() *float64`: nil when `totalLength` nil or 0, else `float64(completed or 0)/float64(total)`.
- Merge-needing scalars on `Download`/`DownloadFile` stay **pointers** (`*string`, `*int64`, `*bool`)
  — the Phase-2 upsert and Phase-5 notify-merge need presence tracking (see nullzero rejection above).
- Structs immutable-by-convention (return copies; service uses value replace, not mutation).
- Tests: table-driven unmarshal (string-number coercion, bool-on-string, missing→nil/zero, garbage
  arrays→empty), `Progress` nil/0/partial/complete, `ResolvedURL` category default.

## Phase 2 — `backend/download/store.go` (port `download/store.py`, 279 lines; bun)

- bun rows mirror the domain structs (pointers for the same merged scalars — see nullzero rejection
  in Phase 0): `downloadRow` (`downloads`, PK `gid string`), `downloadFileRow` (`download_files`,
  `id int64` autoincrement, `gid`, `index int` column `"index"`, `uris` relation),
  `fileUriRow` (`file_uris`, `file_id`), `speedSampleRow` (`speed_samples`, composite PK `(gid,ts)`).
  Relations: `downloadRow.Files`, `fileRow.URIs` with `on_delete:CASCADE`; eager-load with
  `Relation("Files.URIs")`. `speed_samples` is `WITHOUT ROWID` in SQL — do not fight bun; if bun
  can't express it, leave the DDL to goose (already done) and keep the struct plain.
- `ResolvedURL` on the row implements `driver.Valuer`/`sql.Scanner` (JSON marshal); NULL column →
  nil pointer (resolved absent), never an empty struct.
- Every method takes `ctx context.Context` first (bun requires it anyway):
  `All(ctx)`, `Get(ctx, gid)`, `Upsert(ctx, status)`, `AddSpeedSample(ctx, gid, ts, speed)`,
  `SpeedHistory(ctx, gid, limit=60)`, `Remove(ctx, gid)`, `Clear(ctx)`.
- Transactions: `Upsert` is the ONLY method that uses `db.RunInTx(ctx, ...)` — it writes three
  tables (downloads row + delete/recreate file rows + uri rows) and must be atomic; a crash between
  the file-delete and uri-reinsert would otherwise leave orphaned/half-rebuilt file lists.
  Inside the closure use ONLY the tx handle (never the outer `*bun.DB`). Everything else stays
  tx-free: `Remove`/`Clear` are single `DELETE`s (atomic by themselves, cascade handled by FK),
  `AddSpeedSample` is a single `INSERT ... ON CONFLICT DO NOTHING`, reads need no tx. Do not wrap
  the whole service notify-path in a tx — upsert-then-sample are intentionally separate commits
  (Python committed them separately too).
- Row→model mapping: files sorted by `index`, uris sorted by `id`.
- `All`: `ORDER BY created_at DESC` (newest first — service prepends, tests assert `g2` before `g1`).
- `Get`: nil,nil when missing (NOT an error).
- `Upsert`: fetch-or-insert (set `created_at` only on insert), then apply + `updated_at = UTC now`:
  - Scalars overwrite **only when non-nil** — `downloadSpeed=0` is a real value and MUST overwrite
    (pointers; `test_upsert_zero_speed_is_kept` guards this). Same for `totalLength`,
    `completedLength`, `connections`, `numPieces`, `pieceLength`, `verifiedLength`, `errorCode`,
    `errorMessage`, `status`, `dir`, `category`, `resolved`.
  - Files: key existing by `index`; update non-nil fields per file; **clear + rebuild** uris per file;
    delete file rows whose index disappeared from the payload.
- `AddSpeedSample`: `INSERT ... ON CONFLICT(gid,ts) DO NOTHING`
  (bun: `On("CONFLICT (gid, ts) DO NOTHING")`); duplicate ts keeps the FIRST sample.
- `SpeedHistory`: `ORDER BY ts DESC LIMIT n`, then reverse to ascending before return.
- `Remove`: delete row; FK cascade clears files/uris/samples (remove-cascade test). `Clear()`:
  delete all downloads. `Save()`: DROP — it is a no-op in Python kept for a Qt autosave hook that
  no longer exists; do not port it.
- Timestamps UTC; FK enforcement comes from `db.go` DSN pragma — no extra PRAGMA code.
- Tests (mirror `tests/test_store.py`, all with `ctx = context.Background()`): empty, upsert/get,
  scalar merge, zero-speed kept, category, files+uris roundtrip (index 1 preserved, uri order),
  missing→nil, resolved roundtrip (headers/cookies/referer/resolverName), cross-instance persistence,
  remove, clear, speed roundtrip/limit+order/duplicate-ignored/cascade-on-remove.

## Phase 3 — `backend/download/downloader.go` + `backend/download/aria2.go` (port `downloader.py` 84 lines + `aria2_downloader.py` 746 lines)

`downloader.go` — interfaces (method names CamelCase in Go, keep Python order; **ctx on every
blocking method** so cancellation propagates daemon → service → UI):
```go
type NotifyCallback func(Download); type ErrorCallback func(string); type GlobalNotifyCallback func(map[string]any)
type Downloader interface {
  Start(ctx context.Context) error; Stop() error
  Download(ctx context.Context, uri ResolvedURL) (Download, error)
  Pause(ctx context.Context, id string) (Download, error)
  Resume(ctx context.Context, id string) (Download, error)
  Remove(ctx context.Context, id string) (Download, error)
  Purge(ctx context.Context, id string) error
  GetStatus(ctx context.Context, id string) (Download, error)
  Listen(id string, onNotify NotifyCallback, onError ErrorCallback); Unlisten(id string)
  Refresh(ctx context.Context, id *string)
}
type Resolver interface {
  ShouldResolve(uri string) bool
  Resolve(ctx context.Context, uri string, options map[string]any) ([]ResolvedURL, error)
}
```
`options` nil-able; `Refresh(ctx, nil)` = all. `Listen/Unlisten` stay synchronous (mutex-guarded
registry, no I/O) so they take no ctx. NOTE vs Python: `start/stop` return errors instead of
logging-and-continuing (`LOG.error` → caller decides).

`aria2.go` — constants `JSONRPCVersion="2.0"`, `StatusKeys=[completedLength,connections,dir,
downloadSpeed,errorCode,errorMessage,files,gid,numPieces,pieceLength,status,totalLength,verifiedLength]`,
`WSEvents={aria2.onDownloadComplete,aria2.onDownloadError}`.
- `Aria2RPC{host,port,token,httpClient, mu, nextID}`: `Call(ctx, method, params) (any, error)` —
  POST `http://host:port/jsonrpc` via `http.NewRequestWithContext` (replaces the fixed 10s
  `urlopen` timeout; caller controls deadline), prepend `token:SECRET` when set, `id` under mutex,
  `Content-Type: application/json`; map transport failure → `Aria2Error("cannot connect to aria2:
  ...")`, non-2xx → `Aria2Error("aria2 HTTP error CODE: ...")`, bad JSON → `Aria2Error("aria2
  returned invalid JSON")`, non-object → `"invalid JSON-RPC response"`, `error` dict → its message.
  Inject `http.Client` / endpoint for tests (FakeRpc equivalent). Response body is
  `json.Unmarshal`ed straight into `Download` (Phase-1 unmarshaler) at the call sites — no
  `map[string]any` + `FromPayload` step.
- `Aria2Downloader{settings, manageDaemon, running, process *os.Process/exec.Cmd, listening map[string]
  of callback pairs + mutex, globalNotify, resolver semaphore(2), ws conn + goroutine, lock,
  lastSpawn (monotonic), ownsDaemon, rpc}`. `New(settings, manageDaemon, onGlobalNotify)`.
  `Start(ctx)` / `Stop()` (see daemon bullet). The ws goroutine owns a `context.WithCancel`
  derived at `Start`; `Stop` cancels it and joins (`<-(done)` channel, not `Thread.join`).
- Pure helpers (unit-test without daemon, `ctx` only where I/O happens): `parseInt` is replaced by
  the `lib/transform.go` coercion (reuse it — do not duplicate); `finalURL(fileInfo, fallback)`:
  first uri with `status=="used"`, else `uris[0]`, else fallback; `probeHeader(ctx, url, headers)` —
  HEAD with 5s timeout via `http.NewRequestWithContext` (ctx wins over the 5s default), lowercase
  keys, return nil on ANY failure (best-effort); `category(mime, filename)`:
  guess via `mime.TypeByExtension`/`path.Ext`; if either mime empty → `lib.Unknown`; checks in order
  video/audio/image/document(text/*,pdf,json,xml)/compressed(zip,rar,7z,gzip,tar)→`application`
  (return `lib.Category` values, not raw strings);
  `filenameOf(uri)`: `url.Parse` path → `path.Base(unquote)` or nil.
- Daemon: `spawnDaemon(ctx)` — `exec.LookPath("aria2c")` (absent → return descriptive error, NOT
  fatal; `Start` logs and continues degraded like Python's `LOG.error` path);
  running-process check; **adopt-if-listening**: `getVersion` success → `ownsDaemon=false`, return;
  else spawn `--enable-rpc --rpc-listen-port=PORT --dir=DIR --save-session=F --input-file=F
  [--rpc-secret=T]` with `Setsid` (`SysProcAttr{Setpgid: true}`), stdout/stderr DEVNULL; 5s readiness
  loop on `getVersion` with `ctx` cancellation checked each 100ms tick
  (exit-during-startup → clear handle); `ownsDaemon=true`. `Start(ctx)` (once; spawn if managed;
  launch ws goroutine). `Stop()` (cancel ws ctx, clear listeners, close ws, join goroutine, only
  `terminate→wait 2s→kill` when `ownsDaemon`), `ensureDaemon(ctx)` (only managed+owned+dead, throttle
  5s monotonic — missing binary retries at most every 5s).
- WS (`github.com/coder/websocket` — pinned, ctx-native API): `wsURL()` =
  `ws://host:port/jsonrpc[?token=]`; `wsRun(ctx)` reconnect loop: `websocket.Dial(ctx, url, nil)` fail →
  `ensureDaemon(ctx)` + warn + `sleep 1s` (select on `ctx.Done()`); recv loop via `conn.Read(ctx)`
  (replaces `ws.recv()` + `WebSocketTimeoutException` — cancellation/timeout is just ctx); on message
  `onWsMessage(ctx, msg)`; `defer conn.Close(websocket.StatusNormalClosure, "")` (replaces try/finally
  `ws.close()`). `onWsMessage` ignores unparsable/non-object/unknown-method, extracts
  `gid = params[last].gid`, calls `Refresh(ctx, &gid)`. NOTE: coder/websocket `Read` returns
  `(MessageType, []byte, error)` — accept text AND binary messages (parse the payload bytes either
  way); `Close` with `StatusGoingAway` on shutdown path.
- `ShouldResolve(uri)`: `regexp.MatchString(pat, uri)` for each of `http|https|ftp|ftps` substrings —
  preserve the loose **substring** semantics (do NOT "fix" to scheme parsing; resolver tests depend on it).
- `pollStatus(ctx, gid)`: `tellStatus [gid,[gid,status,totalLength,completedLength,files]]` +
  up-to-2s 100ms poll until length appears (skip when terminal complete/error); each tick selects on
  `ctx.Done()` so shutdown interrupts the poll.
- `doRequest(ctx, uri, options, reqHeaders)`: run `addUri` + `probeHeader(ctx, ...)` concurrently
  (goroutines + channel), cancel semantics for stale resolve futures via `ctx` (each new `Resolve`
  cancels the previous resolve's context — replaces `_pendingResolveFutures` cancel).
- `Resolve(ctx, uri, options)`: build `requestContext` (headers/cookies copies, referer =
  `referer||pageUrl`), `requestHeaders` += Referer/Cookie-joined; `ariaOptions` = non-context keys +
  `_aria2Options` + forced `dry-run/use-head=true` (override caller); `gid, headers = doRequest`;
  `defer onResolved(ctx, gid)` (cleanup); map status→`ResolvedURL{finalURL,title=filename,filename,
  mimeType (HEAD content-type else guess),size(content-length else fileSize),category,headers,cookies,
  referer, resolverName:"Rapid"}`; append `guess_extension`-style suffix when filename has no alnum suffix.
  `onResolved(ctx, gid)`: `aria2.remove` (ignore err) then 5× `removeDownloadResult` 100ms apart
  (tick on `ctx.Done()`), warn on failure.
- `GetStatus(ctx, ...)`: `tellStatus [id, StatusKeys]`; empty/non-object result →
  `Aria2Error("aria2 returned no status for ID")`; unmarshal straight into `Download`.
  `_aria2Options(r)`: `out` (filename), `dir`, `header: ["K: V"...]`
  incl. Referer + appended `Cookie: k=v; ...`.
- `Download(ctx, ...)` / `Pause` / `Resume` / `Remove` / `Purge` / `_removeAny(ctx, ...)` — all ctx;
  `Pause/Unpause→tellStatus`; `Remove`: `_removeAny(id, [remove, removeDownloadResult])`
  + `GetStatus`; `Purge`: `_removeAny(id, [removeDownloadResult, remove])` (**order differs** —
  keep it); `_removeAny` tries each, returns on first success.
- `Listen/Unlisten` (mutex-guarded map, no ctx — no I/O); `_notify(ctx, gid, ...)`:
  `GetStatus` err → `onError` + unlisten; else `onNotify` + unlisten when terminal
  (`error|removed|complete`). `Refresh(ctx, id)`: snapshot listeners, filter by id; when id==nil also
  `getGlobalStat` → `globalNotify` if result is object (log-and-swallow RPC errors, never propagate).
  NOTE vs Python: no `if not running: return` guard inside `Refresh` — the poll loop's ctx
  cancellation (owned by the service ticker) is the lifecycle control; document that `Refresh` on a
  stopped downloader still works (needed for tests and the qt-integration shutdown flush).
- Tests (mirror `tests/test_aria2_downloader.py` unit half): resolve metadata+mime/category/cleanup call
  order `[addUri,tellStatus,remove,removeDownloadResult]`; empty-files fallback; halt-retry 5×;
  forced dry-run; browser-context header forwarding; download/no-listen; missing-gid error; status
  parse/non-dict; refresh notify/error/keep-while-active/unlisten-on-terminal/unlisten-on-rpc-fail;
  ws event push vs ignore; adopt-vs-own spawn; stop joins goroutine; pause/resume/remove/purge call
  order. Integration tests (real aria2c + httptest file server incl. throttled Range server): gate
  with `if _, err := exec.LookPath("aria2c"); err != nil { t.Skip }`.

## Phase 4 — `backend/plugin/*` (port `plugin/` 5 files; no Qt)

- `protocol.go`: `Ping="rapid.ping"`, `Resolve="rapid.resolve"`; `type JsonRpcError struct{Code int;
  Message string}` (`Error()`); `EncodeRequest(id, method, params) string` compact JSON + `\n`;
  `DecodeResponse(line string, expectedID int) (any, error)`: invalid JSON → -32700, non-object or
  id-mismatch → -32600, `error` dict → its code/message.
- `models.go`: `StdIOTransportSpec{Command string; Args []string}`,
  `BuiltInTransportSpec{}`, `TransportSpec` sealed interface (or `any` + type switch),
  `PluginSpec{ID, Version, Name string; Transport TransportSpec; Match []string}`,
  `CreateTransporter()` → `ProcessTransport` for stdio, `TransportError` otherwise.
- `transport.go`: `TransportError`, `Transport interface{ Request(ctx context.Context, payload string) (string, error) }`,
  `ProcessTransport{Command, Args, StartTimeout=3s, RequestTimeout=8s, WriteTimeout=2s,
  TerminateTimeout=2s}` via `os/exec` + `exec.CommandContext`: start, stdin write + deadline
  (select write-completion vs `ctx.Done()`/`time.After`), stdout single-line read
  (`bufio.Reader.ReadString('\n')` in a goroutine selected against ctx; empty output → `"Plugin
  returned nothing"`, over-deadline → `"Plugin response timeout"`, pre-exit → break), `teardown`
  (`Signal(Interrupt/SIGTERM)`→wait→Kill→wait). No Qt — this replaces `QProcess`. Timeouts and ctx
  compose: ctx cancellation aborts even a pending read.
- `resolver.go`: `PluginError`; `ResolverPlugin{spec, rid}` (implements `Resolver` iface from
  download pkg — import it, don't duplicate); `_call(ctx, ...)` = nextID→encode→transport→decode with
  Transport→wrap rest as `PluginError("Transport error: ..."/"Response error: ...")`;
  `ShouldResolve`: regex search per non-empty pattern; `Resolve(ctx, uri, options)`: call
  `rapid.resolve [uri, options||{}]`, require dict with `items []any` else `PluginError("Resolve
  returned invalid result")`; each item is `json.Marshal`ed back to bytes then `json.Unmarshal`ed
  into `ResolvedURL` (+ `resolverName=spec.name`) so plugin items reuse the Phase-1 unmarshaler
  instead of manual field mapping.
  NOTE: Python `__init__` takes `timeout_ms/start_ms` but ignores them — do NOT reintroduce; timeouts
  live on `ProcessTransport`.
- `manager.go`: `ManifestName="plugin.json"`; `createTransportSpec(manifestPath, data)`:
  transport must be dict, command str (relative → `manifest.parent/command`), args filtered to
  strings; `loadSpec(manifest)` → nil,nil on: unreadable/bad-JSON, non-dict, `type != "resolver"`,
  name non-string, bad transport; id = `id` str else name; version default `"0.0.0"`; match filtered
  to strings; `Scan(base)` skips non-dirs, globs `*/plugin.json`; `Get(id)`, `ResolverNames()`,
  `Resolve(ctx, uri, options)`: per plugin `ShouldResolve` → `resolve` → inherit headers/cookies
  (when resolved nil and context holds `map[string]string`) and referer (`referer||pageUrl` str).
  Plugin crashes/invalid replies propagate as errors (Python lets them bubble out of `resolve` too).
- Keep `rapid/plugins/sampledemo/{plugin.json,plugin.py}` untouched as the test fixture
  (transport command `plugin.py` is relative → resolve against manifest dir; must be executable).
- Tests (mirror `tests/test_plugin_manager.py`): discover sampledemo; missing dir → empty; supported
  URL → 2 items with resolverName + `example.com/video/` prefix; context inheritance; unsupported →
  empty; `shouldResolve` true/true/false; loadSpec rejects 4 bad manifests; relative command; id/version/
  transport; args+match string-filtering. Transport test: stub executable (sh script) for
  line-echo, timeout, and no-output cases.

## Phase 5 — `backend/download/service.go` pure logic (port `download/service.py` ~534 lines minus Qt)

Pure `DownloadService{settings, store, downloader, resolver, pluginManager, mu, downloads []Download
(newest-first), workerSem chan(2), pollTicker *time.Ticker, pollCancel context.CancelFunc,
callbacks...}`. No Qt: **model/view binding is NOT here** — the miqt list model, the
`DownloadFilterProxy` equivalent, and `main.go`-style wiring are specified in
`tasks/qt-integration/task.md`. This phase exposes typed state + callbacks; that task consumes them.
- `New(ctx, settings, store, downloader, resolver)`: `downloads = store.All(ctx)`;
  `pluginManager = NewPluginManager(settings.PluginDirs)`; worker pool = buffered semaphore(2)
  (replaces `ThreadPoolExecutor(2)`).
- `Start(ctx)`: `downloader.Start(ctx)` (return error — Python had no failure path here, Go does);
  for restored `status=="active"`: `GetStatus(ctx, ...)` try — skip silently on error (daemon forgot
  it) — else `Listen(gid, emitNotify, emitError)`; start ticker (`settings.PollIntervalMs`) whose
  goroutine calls `_poll` with a derived ctx. `Close()`: stop ticker, call `pollCancel`, wait
  workers, `downloader.Stop()`.
- Read API (plain Go values, no roles): `Counts() map[string]int` (`all` + per-category non-empty),
  `ActiveCount() int`, `ActiveDownloads() []string`, `DownloadName(gid)` (unknown gid → gid;
  resolved.title→filename; else `files[0].path` basename after last `/`; else gid),
  `SpeedHistory(ctx, gid) []SpeedSample` (typed — NOT `[]map`; the `asDict` slot wrapper is gone),
  `FormatSize(n int64) string` (`0 → "—"`, units B/KB/MB/GB/TB ÷1024, `0` decimals when ≥100 or B
  else 1), `PickFolder(ctx, startDir)` — try `zenity --file-selection --directory`, then
  `kdialog --getexistingdirectory` (`exec.LookPath`, `exec.CommandContext` 60s timeout, trim,
  non-zero→""), else return `startDir || settings.DownloadDir` (no Qt fallback here; native dialog
  ownership is decided in qt-integration).
- `Resolve(url)` / `ResolveRequest(req BrowserRequest)` enqueue `_doResolve` async (worker sem + ctx):
  - blank url → `OnResolved([], {"url":"URL is required"})` (error map stays — it is the extension
    error contract, not QML serialization).
  - `browserResolved==true` → `_browserResolvedURL` + `UniqueName` when filename set (dest =
    `resolved.dir || settings.DownloadDir`) → emit single struct.
  - else `pluginManager.Resolve(ctx, ...)` → if empty, `resolver.Resolve(ctx, url[, options])`
    (options only when non-nil — keep the conditional); per item `_withRequestContext` +
    `UniqueName(dest=downloadDir)` → emit `[]ResolvedURL`; on error → `OnResolved([], {"url": err})`.
- `_browserResolvedURL(url, options) ResolvedURL` (pure — directly unit-test): str-map filter
  headers/cookies; filename from `filename`/`savePath` (savePath parent → dir); mime =
  explicit else `mime.TypeByExtension(ext of filename||url path)`; extension fix when suffix empty/
  non-alnum; category = explicit (non-empty) else mime-prefix audio/video/image else `lib.Unknown`;
  size = int ≥0 non-bool else nil; referer = `referer||pageUrl` (str); title = `title||filename`
  with originalFilename==title refresh; `resolverName="Browser"`.
- `UniqueName(dest, filename)`: `dest/filename` free → as-is; else `stem (1).ext`, `(2)`… (first
  candidate is `(1)` — note loop quirk `n>1` formatting but starting n=1 yields `(1)` first).
  `_withRequestContext(r, options)`: fill headers/cookies/referer/filename/title/mimeType ONLY when
  resolved field nil (options type-asserted: headers/cookies `map`, referer/filename/title/mime `string`).
- Mutations (all take ctx, all operate on structs — QML `Slot(list-of-dicts)` entry points do NOT
  exist; the widget layer passes `[]ResolvedURL`): `Download(ctx, uris)` → per-item `_download`;
  `Pause(ctx, ...)` = downloader.Pause + unlisten + notify; `Resume(ctx, ...)` = downloader.Resume +
  notify + listen; `Stop(ctx, gid)` ("removed" retained, NOT deleted) = downloader.Remove + unlisten +
  notify; `Delete(ctx, gid, deleteFromDisk)` = downloader.Purge + unlisten + optional `_deleteFromDisk`
  (per-file `os.Remove`, ignore `os.IsNotExist`, swallow other errors) + `store.Remove(ctx, ...)` +
  slice-remove + `OnRemoved`.
- `_download(ctx, r)`: `downloader.Download(ctx, ...)` → set `resolved/category` → `store.Upsert(ctx, ...)`
  → `Listen` → prepend `_insert` → `_notify` → `OnAdded`. `_notify(ctx, d)`: merge with previous when
  known — `resolved/category` fallback, `downloadSpeed` = previous when `status != "active"` else
  `d.speed || previous`, `completed/total` = `d || previous`; `store.Upsert`; sample when
  `active && speed` (`ts = now_ms` via `time.Now().UnixMilli()`); replace; `OnCountsChanged` on
  category change, `OnActiveCountChanged` + `OnCompleted`/`OnFailed(gid,msg||"")` on status change;
  ALWAYS `OnChanged`. `_poll(ctx)` = `downloader.Refresh(ctx, nil)` (async via worker sem).
- `FilterDownloads(downloads, category, search) []Download` (pure helper, replaces the
  `DownloadFilterProxy` *matching logic*; the actual Qt model/proxy lives in qt-integration):
  category exact-match when non-empty; search trimmed+lowercased, matches resolved.title/filename or
  `files[0].path` basename; empty search → pass. Keep name-resolution identical (split on last `/`).
- Tests (mirror `tests/test_download_service.py` with FakeDownloader implementing both ifaces, ctx =
  `context.Background()`): start/close flags; resolve-context inheritance (compare STRUCTS, not
  dicts — `asDict` equality becomes `reflect.DeepEqual`/`==` on `ResolvedURL`); preresolved skip +
  struct equality; mime-extension fix; image-from-URL inference; download persist (`g1`
  active/category/resolved); ordering (`g2,g1`, OOB→nil-equivalent) + stop→`removed` retained;
  activeDownloads/names/count; poll→changed+store+sample; complete keeps last speed; paused freezes
  speed; start relistens live-only; start skips daemon-forgotten; stop-keeps-in-store vs
  purge-deletes-file+row; empty-URL error tuple; FormatSize vectors
  (`0→—`, `512→"512 B"`, `1536→"1.5 KB"`, `1MiB→"1.0 MB"`, `1TiB→"1.0 TB"`).

## Phase 6 — `backend/browser/bridge.go` (port `browser_integration/service.py` 201 lines)

Contract source: `browser-extension/src/background.js` sends `POST /downloads` + `GET /health`
with `X-Rapid-Extension: 1` and extension `Origin`s.
- `MaxBodyBytes = 2MiB`; `AllowedOriginSchemes = [chrome-extension://, moz-extension://]`;
  `BlockedHeaders = {connection,content-length,cookie,host,proxy-connection}`;
  `type BrowserRequestError struct{Msg}`.
- `type BrowserRequest struct` (typed — replaces the `dict` that flowed `extension → normalize →
  downloadRequested → resolveRequest`): `URL string; Headers, Cookies map[string]string; Referer,
  PageURL, Title, Filename, MIMEType, Category, Source, SavePath string; Size *int64;
  BrowserResolved bool`. `Size` stays a pointer (absent vs 0 matters to `_browserResolvedURL`).
- `StringMap(v any, blocked) map[string]string`: non-dict → empty; keep str→str only, trim key,
  skip empty + blocked(lowercased).
- `NormalizeBrowserRequest(v any) (BrowserRequest, error)`: non-map → error; url scheme must be
  http/https/ftp/ftps (parsed via `net/url`) else `BrowserRequestError("A downloadable HTTP, HTTPS,
  FTP, or FTPS URL is required")`; optional str keys `referer,pageUrl,title,filename,mimeType,
  category,source,savePath` + `size` (int ≥0, reject bool) + `browserResolved` (only when `== true`).
  Extra keys dropped. The service's `ResolveRequest` takes this struct directly — no map re-encoding.
- `originAllowed(origin)`: empty/none → true; else has-prefix check (NOT url parsing).
- HTTP server (`net/http`, `127.0.0.1:17654` default, `Start(ctx)` once on first call returning bind
  errors instead of emitting Qt signals, `Close(ctx)` via `http.Server.Shutdown(ctx)` with 2s timeout
  ctx, `Addr()` actual bound address for port-0 tests, `OnDownloadRequested func(BrowserRequest)`,
  `OnError func(string)` for runtime (serve-loop) failures only):
  - CORS helper: echo `Origin` + `Vary: Origin` when allowed; always
    `Allow-Headers: Content-Type, X-Rapid-Extension`, `Allow-Methods: GET, POST, OPTIONS`.
  - `trusted(r)`: origin allowed AND `X-Rapid-Extension == "1"`.
  - `OPTIONS /downloads`: 204 when origin allowed AND `Access-Control-Request-Headers` contains
    `x-rapid-extension` (case-insensitive substring), else 403 `{ok:false,error:"Untrusted extension origin"}`.
  - `GET /health`: trusted + exact path → 200 `{ok:true,app:"rapid"}`, else 404.
  - `POST /downloads`: path+trusted else 403 `{ok:false,error:"Untrusted extension request"}`;
    Content-Length ≤0 or >2MiB → 413; parse+normalize failure → 400 with err text; success →
    `OnDownloadRequested(req)` + 202 `{ok:true}`. All replies compact JSON + `Content-Type:
    application/json`. Disable access logging. `MaxBytesReader` enforces the 2MiB cap at the
    transport level too (not just Content-Length header check).
- Tests (mirror `tests/test_browser_integration.py` + table): normalize keeps context / strips
  blocked (Host, Content-Length) / preresolved meta / rejects `blob:`; live server (port 0):
  preflight 204 then POST 202 + callback payload; untrusted origin/header → 403; oversized → 413;
  `GET /health` 200 vs untrusted 404.

## Phase 7 — `backend/settings`, `backend/seeder`, `backend/clipboard`

- `backend/settings/settings.go` (port `setting/models.py` 39 lines): `Settings{DataDir, DownloadDir
  string; PluginDirs []string; BaseDir string; Aria2Host string; Aria2Port int; Aria2Token,
  Aria2SessionFile string; Aria2SaveSessionInterval, PollIntervalMs int}` + `Default(baseDir)`:
  appData = `os.UserCacheDir()` (fallback `~/.cache`) + `rapid` (document: replaces Qt
  `AppDataLocation`); `MkdirAll`; downloadDir = user `~/Downloads` (document: replaces Qt
  `DownloadLocation`; check `XDG_DOWNLOAD_DIR`/`~/Downloads` fallback); session file
  `appData/aria2.session` touched; pluginDirs = `[base/plugins, appData/plugins]`;
  aria2 defaults `127.0.0.1:6800`, token `""`, interval `1`, poll `1000`. Keep JSON/key names stable.
- `backend/seeder/seeder.go` (port `seeder.py` + `download_seeder.py` 224 lines + `__main__.py`):
  `Seeder interface{ Name() string; Seed(ctx context.Context) error }`,
  `SeederService{seeders}` (`Seed(ctx)` prints `"Seeding: NAME"` then runs, first error aborts).
  `DownloadSeeder{store}` seeds the SAME 7 fixtures with identical
  gids/statuses/categories/sizes (copy the table: `d6b8a91c…` active/video 2GiB,
  `9c3d7e1a…` complete/video, `5f0a2b8c…` paused/audio, `a1b2c3d4…` active/compressed 4GiB,
  `e7f8091a…` error/document code 1 "URI not found", `11223344…` complete/application,
  `f1e2d3c4…` waiting/document) + active ones get 60 speed samples (`now-60s+i*1s`,
  `_speedCurve(base, seed)` = ramp `min(1,t*4)` × burst `0.25 sin(i/9)+0.15 sin(i/3.7)` × stall
  (`0.9`, 5% → `0.15`) × jitter `U[0.8,1.15)`, seeded `math/rand`). `cmd/seed` (or `go run
  ./backend/seeder`) wires `db.Open` + store + service — replaces `python -m rapid.backend.seeder`.
  Remove the hardcoded `/home/thoriqadillah/Downloads` dir (use `settings.DownloadDir`).
- `backend/clipboard/clipboard.go` (port `clipboard/service.py` 38 lines, logic only):
  `URLRegex = https?://\S+`, `ExtractURL(text string) string` (first match else `""`); no Qt —
  copy/text/live-signal stay in `widget/` (miqt clipboard). Unit-test the regex vectors.
- Python `notification/service.py` → already covered by `service/notification/` in Go; do NOT port.
  `rapid/main.py` composition (`downloader_service`, tray wiring, quit-dialog gating on
  `activeDownloads`, startup `downloader.start()/browser.start()`, shutdown close order) is specified
  in `tasks/qt-integration/task.md` — do not reimplement here.

## Phase 8 — Wiring, `go vet`, and parity checklist

1. `go get github.com/coder/websocket && go mod tidy && go build ./... && go vet ./...`.
   No other new deps.
2. `go test ./backend/... ./lib/...` green, headless (no Qt imports in backend).
3. Parity checklist against Python exports (`backend/__init__.py`, `download/__init__.py`,
   `plugin/__init__.py`, `browser_integration/__init__.py`): every symbol has a Go counterpart except
   the deliberately dropped Qt-bound surface — `DownloadService(QAbstractListModel)` roles,
   `DownloadFilterProxy(QSortFilterProxyModel)`, `ClipboardService(QObject)`, `NotificationService`,
   `ProcessTransport(QProcess)`, `Database(QStandardPaths)`, `Settings.default(QStandardPaths)`,
   `fromPayload/asDict/fromDict` (replaced by `encoding/json` + bun — see Phase 1 audit), and
   `DownloadStore.save()` (no-op for a dead Qt autosave hook). Qt-side counterparts (if any) live in
   `tasks/qt-integration/task.md`.
4. Delete nothing under `rapid/` (Python stays until widget cutover); new Go code must not import it.
5. Update this file's status boxes as phases land.

Status: [x] P1 models+transform · [x] P2 store · [x] P3 aria2 · [x] P4 plugin · [x] P5 service · [x] P6 browser · [x] P7 settings/seeder/clipboard · [x] P8 wiring/parity
