# Reusable Widgets

The reusable MIQT widgets in this directory are the first-stage port of
`rapid/qml/ui`. Their canonical filesystem icon directory is:

```text
assets/icons/
```

`ResolveIconPath` may locate that same directory relative to the current
working directory, executable, or source tree so development and packaged runs
use the same canonical asset layout. It does not accept an environment
override, alternate root, absolute path, or `qml/icons` fallback.

The old PySide6 resource directory remains as a compatibility copy:

```text
rapid/qml/icons/
```

Keep it synchronized for the PySide6 app. The `regenerate-icons` task generates
from `assets/icons/icons.qrc` and copies the SVGs, QRC manifest, and generated
Python resource module to the compatibility directory. MIQT code must continue
to use only `assets/icons/`.

For a packaged MIQT build, the generated wrapper is checked in as:

```text
ui/resources_generated.go
ui/resources.rcc
```

Regenerate it with:

```bash
go generate ./ui
```

The directive on `ui/icon.go` calls `task build:icon`, which uses `rcc` from
`PATH` or the project `.venv`, embeds the resulting resource data, and removes
the generator's machine-specific directive from the checked-in Go source. The
source manifest is `assets/icons/icons.qrc`. If `rcc` is unavailable, the task
prints:

```bash
python3 -m pip install "PySide6>=6.8,<7.0"
```

The same task can be run directly without `go generate`:

```bash
task build:icon
```
Missing icons return an error from `ResolveIconPath`/`LoadIconChecked` and an
empty path/null-safe icon from the convenience APIs.
