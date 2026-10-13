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

## Untouched pre-existing failures (clean tree, out of S1/S2 scope)

- `TestSettingsPushKeepsPreviousSidebarSelection` (sidebar nav)
- `TestRTextFieldInputContract` (textfield binding)
