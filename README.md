# rapid

PySide6 + QML desktop app, statically typed (mypy strict).

## Prerequisites

- [pyenv](https://github.com/pyenv/pyenv) (Python version pinned in `.python-version`)
- [Poetry](https://python-poetry.org/)

```bash
pyenv install -s   # installs the version from .python-version
```

## Dev

```bash
poetry install
poetry run mypy
poetry run dev
```

Runs the app (`rapid/main.py`, loads `rapid/qml/Main.qml`).

## Build

```bash
poetry run mypy
poetry run poe build
```

Produces a standalone executable at `dist/rapid.bin` (Nuitka onefile build, config in `rapid/pysidedeploy.spec`).
