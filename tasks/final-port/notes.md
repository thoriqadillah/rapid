# Final-port deviations from task.md

Decisions made during implementation that disagree with the step plan.
Code wins per the task file's own rule; this file records _what_ and _why_.

## S1

- **SetCategory/SetSearch deleted (user).** Signals are the API; views bind
  `Category`/`Search` directly. Normalization moved into `matches`
  (trim + lowercase both sides, `"all"` counts as all), so raw queries are
  preserved verbatim in the signals. Task S1 assumed the setters exist.
- **Speed/ETA gating lives in `api.FormatSpeed`/`FormatEstimation` (user),**
  not at the widget call site as S1 prescribed. Non-`active` returns `"—"`.
  `TestEstText` fixtures gained `Status: "active"`.
- **`TestFilterChangedOnlyTracksFilters` added (agent).** Not in the task;
  locks the S1 fix (item churn must not fire `FilterChanged`).

## S2

- **Queue has no error plumbing (user + agent).** `Enqueue(fn func(ctx)) bool`,
  `Start(ctx)` — no task names, no error return, no central `onError`. Tasks
  handle their own errors inline (the closure has more context, e.g. gid)
  and marshal to `notifyError`. Task S2 / Rule 2 spec'd
  `Enqueue(task, fn) error` + `onError(task, err)`; S4/S9 snippets in the
  task file still show the old shape and need re-reading when we get there.
- **No `ServiceConfig` (user).** `NewService(settings)` builds the queue,
  the aria2 downloader (`rpc` client + `NewAria2Downloader`, same instance
  also serves as `Fallback`), and later the plugin manager, all internally.
  Task S2/S10 spec'd config injection from main; S10's composition block
  shrinks to `NewService(s)` + `Start`.
- **`Refresh` deleted (user).** The polling full-reload is gone;
  `handleNotify` does an in-memory replace-by-gid instead
  (`iter.FindIndex`, unknown gids ignored until S3). `downloadsEqual` went
  with it (`Signal.Set` dedups anyway). View polling removed, pending
  service-managed `Poll` (S11). Affected tests rewritten, not just patched:
  `TestServiceUnsubscribeAndNoOpRefresh` → DB-free `TestServiceUnsubscribe`,
  `TestServiceRefreshFromDB` folded into `TestServiceStartLoadsAndClose`.
- **Fake `Downloader` pulled forward into S2 tests (agent).** S4 planned it;
  needed now because `Start`/`Close` have no nil guards (unit tests never
  boot a daemon). S4/S5/S9 extend it with scripted behaviors.
- **`notifyError` is slog-only for now (stepping stone).** Toasts arrive with
  user-facing actions (S4) / completion events (S12), which is when the
  service has something user-caused to report.

## S3

- **`Refresh` + `downloadsEqual` deleted (user), `downloadsEqual` re-added
  for the merge skip (agent).** Polling full-reload is gone (view timer
  removed); notify path is in-memory replace + S3 merge. Service-managed
  `Poll` (S11) still pending.
- **Merge follows the checklist over Python on one point:** active rows take
  aria2's speed verbatim including 0 (Python kept stale speed on 0).
  Resolved/Category fall back to the previous row; everything else is
  authoritative fresh. Unknown gids prepend newest-first.
- **Events are callback slices with index-nil dispose** (`OnCompleted` /
  `OnFailed`); `OnChanged`/`SetItems` are gone, tests subscribe via raw
  `reactive.Effect`. Fired on GUI thread from inside the merge.
- **00002 stays deleted, no 00003 (user: dev phase).** DBs are wiped and
  reseeded, so no trigger exists anywhere to drop. If a trigger-bearing DB
  ever needs upgrading again, re-add the drop then.
- **Tx boundary lives in the service (user), not the store.** Store funcs
  are plain queries on `bun.IDB` (`bun.Tx` satisfies it); `mergeDownload`
  wraps upsert + speed sample in one `RunInTx` so a notify persists
  atomically. `UpdatedAt` still Go-owned on the update path.
- **`notifyError` removed (user); failures slog inline at the call site**
  with gid context. Revisit if S4/S12 want toasts for these paths.
- **Store tx details:** `mergeDownload` owns one `RunInTx` over upsert +
  speed sample; store helpers stay tx-free. `UpdatedAt` set in Go on the
  update path (`ExcludeColumn("created_at")` only).
  `TestUpdatedAtOwnedByTrigger` → `TestUpdatedAtOwnedByGoUpdate`.
- **No explicit equality skip in the merge (user).** `Signal.Set` already
  dedups via `DeepEqual` (and `Download.Equal` is `DeepEqual`), so the
  `downloadsEqual` guard the task prescribed walked every notify twice.
  Both the check and the helper are gone.
- **Events use a `lib/signal` package (user), not callback slices.**
  `signal.Event[T]` with Connect (returns disconnect) / Emit; no value, no
  initial fire, no dedup — every Emit reaches every slot once. Service keeps
  thin `OnCompleted`/`OnFailed` wrappers so S12's call shape is unchanged.
- **`OnFailed` and `OnError` merged into one `Failed Event[error]`
  (user).** Terminal aria2 failures emit `"<name>: <aria2 message>"`
  (`"unknown error"` fallback); persist failures emit the wrapped DB error.
  Main toasts all of them as `"Download error"`. `emitError` and its test
  are gone; the package test covers the mechanics.
- **Merge entry is a method, `s.upsertDownload` (user),** sharing its name
  with the store function — reads fine at call sites, slightly confusing
  in the store file. Left as is.
## S4

- **Active = status `active` only (Python), not non-terminal.** Task S4/S12
  says non-terminal, but Python counts strictly `active` for both the badge
  and the quit-guard names, and S18's title agrees. Paused/waiting rows
  neither badge nor block quit.
- **ActiveCount is a `Computed[int]` field, Reduce-powered (user).**
  Writer-Signal + `setItems` came and went: per-tick repaint is unfelt,
  silent-stale is not. No `ActiveCount()`/`ActiveDownloads()` methods —
  S12/S18 read the field / derive from `Items`. Against the task's
  "State vs events" prescription, on evidence. Active-only counting
  unchanged.
- **Delete = Remove + Purge (checklist)**, not Python's purge-only (which
  leaves actives running). Unknown gids tolerate not-found on both calls;
  `IsAria2NotFound` already covers both aria2 phrasings — no extension.
- **`.aria2` control files unlinked on delete** (checklist addition over
  Python's path-only unlink). Other unlink errors log and continue.
- **Pause unlistens, Resume re-listens** (Python). Stop retains the
  `removed` row; only Delete purges + drops.
- **No `DownloadName` wrapper:** `api.Download.Name()` already implements
  the exact title/filename/basename/gid table — S12 should call `d.Name()`.
- **Action failures carry display names** (`failAction` looks up `Name()`,
  falls back to gid) via the Failed event → `"Download error"` toast.
- **Tests are two-tier (no Qt loop in service tests):** downloader-side
  effects through the real queue + barrier (`newActionService` boots a
  real `Start` against the fake); GUI-marshaled tails are dropped without
  a pumped loop, so memory/DB tails are covered by direct method tests
  (`dropFromMemory`, merge) and store tests. `mainthread.Start` without a
  QApplication silently drops — hence the split, documented on `syncQueue`.
- **Queue `min(1, depth)` → `max` (agent fix):** the clamp was inverted and
  capped the buffer at 1, wedging the second rapid action.
- **Disk unlink lives in helpers as `removeFiles` (user)** — moved out of
  the service with its behavior unchanged.
- **Memory drop uses `iter.Filter` (user)** — no clone/index juggling.
- **Merge fallback is conditional-ternary (settled):** previous row's
  values apply only when fresh lacks them (checklist), written with
  `bools.Ternary` to match surrounding style. Identical behavior today.
- **`toResolvedModel` nil-means-missing (empty URL → nil)** caught one test
  fixture: resolved fixtures need a real URL to survive the DB round trip.
- **`TestServiceOnChanged` fixed (agent):** the post-`OnChanged`-deletion
  version subscribed with a dependency-less `Effect` that could never
  re-fire; now reads `Items`+`Counts` like the old method did.

## S5

- **Stamps carry click-time status, not just time (agent).** The task's
  `map[gid]time` can't implement its own "confirming notify clears" rule;
  `actionStamp{at, status}` skips same-state echoes within `2×Poll`,
  applies anything else. Sweep on stamp so deleted rows don't leak entries.
  Clock injected (`now` field) — tests use a manual clock, no sleeps.
- **Forgotten gids fail fast (agent):** Start marks them `error` +
  persists; persist failure returns (DB-broken means the app can't run).
  Exact message `"download not found in aria2"`.
- **Dedupe keys need gid the callback lacks (agent):** `ErrorCallback` is
  `func(error)`, so `Listen` sites wrap per-gid adapter closures — no
  interface change. `(gid, text)/30s`, first toasts + warns, repeats debug.
  Core split into `listenerError` (Qt-free tests; marshal drops without a
  loop, same two-tier story as S4).
- **Yours, kept as found:** `queue.Go` rename, `OnFailed`→`OnError`,
  `emitError(action, gid, err)`, merge as `s.upsertDownload`, ternary
  fallbacks, `iter.Find`, `cloneItems`, `removeItem`.

## S6

- **Resolve params are a list, not a dict (real plugins win).** The task
  shows `params:{"uri":...}`, but the Python manager sends `[uri, options]`
  and real plugins read `params[0]` — dict params would break every
  existing plugin. `ResolveRequest` builds `[uri, options]`.
- **Plugin `PluginError` carries identity (user):** `{PluginID, Err}` with
  `Unwrap`, so `lib/errors` matchers and `errors.Is/As` traverse it. That's
  the one thing the generic lib types lack (who failed). No test (user).
- **Load-time errors are `*PluginError` (user):** identity is the manifest
  path (id/name may be absent exactly when load fails); the `plugin %s:`
  message prefixes are gone since the type carries it, and `checkCommand`
  lost its manifest param. Rejects test pins the wrap via `ErrorAs`.
- **Manifest match is strict `[]string` (user):** non-string, empty, or
  non-list match all fail the load (an empty regex would match everything,
  which the earlier lenient filter risked). Decoder rejects them, not code.
  Args joined match in strictness (user): transport decodes inline in one
  pass (`Args []string`, no `json.RawMessage`/second unmarshal/`any`
  filtering — `parseTransport` deleted); non-string/non-list args now fail
  the load like match does. id →
  name fallback, version → `"0.0.0"`, regexes compiled at load, absolute /
  `..`-escaping commands rejected.
- **Protocol is plain structs only (user):** no decode wrappers, no
  `check` — S7 unmarshals into `PingResponse`/`ResolveResponse` directly
  and validates inline. `ResolveResult{Items}` wrapper kept: real plugins
  send `result` as an object, so a flat `[]ResolvedItem` can't decode them
  (your deviation, kept honest).
- **Empty match patterns rejected (user):** an empty regex matches
  everything, so `""` fails the load instead of being skipped.

## S7

- **`runResolve` also takes options (agent).** Task signature is
  `runResolve(ctx, manifest, uri)`; plugins receive `[uri, options]` (Python
  manager), so a fourth argument forwards headers/cookies/referer context.
  Without it auth-gated URIs resolve with the wrong request context.
- **No ping probe at all (user/task-optional).** One spawn per resolve;
  a failed resolve is the health signal. The Python ping existed for QProcess
  startup lifecycle, which `os/exec` does not need in a separate round trip.
- **Transport errors carry no plugin identity (user).** The package-local
  `PluginError` was removed with `errors.go`; transport failures now surface
  as raw `lib/errors` types (`TransportError`, `JsonRpcError`) or raw
  `encoding/json` errors. Invalid JSON is a plain unmarshal error, not a
  synthesized `-32700` `JsonRpcError`; callers cannot tell which plugin from
  the error alone (the manager/site logs know the manifest).
- **`PluginResolver` holds only the manifest (agent).** Task wrote
  `{manifest, items}`; items are a per-call return, not resolver state.
- **Plugin cwd is the manifest dir (agent).** Python inherited the app's
  cwd. Anchoring the child at its own dir makes relative plugin resources
  predictable; the command itself is still resolved per the task rule.
- **Timeout injection is a package var (agent).** `resolveTimeout` (8 s) is
  swapped by the timeout test instead of threading a duration through every
  call, so the test never sleeps 8 real seconds.
- **Fixtures are command manifests, not scripts (agent).** `testdata/{echo,
  error,hang}` use `printf`/`sleep` from `PATH` with the canned response in
  `transport.args`. An external formatter kept pretty-printing the JSON
  inside `.sh` fixtures onto multiple lines, which breaks the one-line
  protocol; command args survive reformatting. Same coverage as the task's
  `testdata/echo_plugin` idea, without files that can be rewritten.

## S8

- **Plugin package moved to `services/download/plugin` (user).** Task put it
  at `services/plugin`; it now lives under the download service with the
  import path updated. Layout only — package contents are unchanged.
- **Manager is constructed in `NewService` (user).** Consistent with the
  S2 deviation that dropped `ServiceConfig`: `NewService(settings)` builds
  the manager from `settings.PluginDirs` itself. `Start` logs `Len()` at
  debug level; `Resolve` consumes it in S9.
- **No `claimed` bool (user).** `Resolve(ctx, uri, opts)` returns
  `([]api.ResolvedURL, error)`: an error means a plugin matched and failed
  (surface it, never fall back); nil items + nil error means no plugin
  matched. Caveat: a matching plugin that successfully returns zero items is
  indistinguishable from "no match" and falls through to aria2 — fine while
  an empty item list means "can't resolve".
- **Nil manager is unsupported (user).** The nil-receiver guards and the
  "nil = no plugins" contract were dropped; `Resolve`/`Len` dereference the
  receiver. S10 always constructs a manager, so the service field is never
  nil in practice.
- **Scan covers child dirs only (agent).** Mirrors Python's `*/plugin.json`
  glob: a `plugin.json` directly inside a `PluginDirs` entry is ignored;
  only `plugins/<name>/plugin.json` is discovered.
- **Deterministic plugin order (agent).** `os.ReadDir` sorts entries, so the
  chain is dir order then directory name. Python relied on glob/dict order,
  which is not an explicit guarantee.
- **Tests build temp scan roots (agent).** Valid and broken plugin dirs are
  created under `t.TempDir()` around copies of the `testdata/{echo,error}`
  manifests instead of committing a `testdata/scan` tree; the broken
  manifest is written inline.

## S9

- **No `Refresh` after add (agent).** The downloader interface lost `Refresh`
  in S2 (user deviation), and `Aria2Downloader.Download` already returns
  `GetStatus` after `addUri`, so the row is fresh without an extra enqueued
  refresh. The S9 task snippet still shows `Refresh(ctx, gid)`.
- **One queued op per batch, not per item (agent).** `Download` enqueues a
  single closure that loops the items. This keeps the batch atomic with
  respect to user actions and lets the per-item failure loop collect into
  one summary error; per-item ops would need a barrier/counter to summarize.
- **`ResolveRequest` adds `PageURL` (agent).** Task lists `Referer`; the
  bridge carries both `referer` and `pageUrl`, and Python fell back
  `referer or pageUrl`. `PageURL` preserves that for browser requests.
- **Request category/size are typed (`string`/`int64`) (agent).** The bridge
  decodes `headers`/`cookies` as `map[string]any` and `size` as `*int64`;
  S17 converts (via `maps.StringMap` / nil check) instead of making every
  request field `any` like Python's dict.
- **Browser-resolved category inference only for audio/video/image (agent).**
  Matches Python: unknown + mime prefix in that set; `application/*` stays
  unknown.
- **Unique-name stays at Download time (task) rather than resolve time
  (Python).** Preview filenames are not pre-renamed; collisions are applied
  to the item actually sent to aria2, including browser-resolved items.
  `TestDownloadUniqueNameAndContinueOnFailure` covers the integration;
  `uniqueName` table tests already existed.
- **Resolve is split for testability (agent).** Exported `Resolve` owns the
  goroutine + `mainthread.Start` marshal; `resolve` is Qt-free and is what
  tests call (same two-tier pattern as S4/S5). `Download`'s GUI tail
  (Listen/upsert) is not asserted in service tests for the same reason.
- **Plugin-chain tests use a real `plugin.Manager` (agent).** Manifests with
  `printf` transports are written inline in `resolve_test.go`; only the
  fallback is a `fakeResolver`. The task asked for a fake resolver (used)
  and a fake Downloader (extended with `downloads`/`downloadErr`), not a
  fake manager.
- **Plugin claimed + error is detected by the error itself.** With `claimed`
  gone, a non-nil manager error short-circuits before the fallback; a
  matching plugin that returns zero items falls through, per the S8 caveat.
- **`PickFolder` still TODO.** S9 only covers resolve/download; the comment
  now names only `PickFolder`, which S15 needs.

## Untouched pre-existing failures (clean tree, out of S1/S2 scope)

- `TestSettingsPushKeepsPreviousSidebarSelection` (sidebar nav)
- `TestRTextFieldInputContract` (textfield binding)
