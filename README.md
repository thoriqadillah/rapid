# rapid

Go + Qt6 desktop app (Widgets), using the [miqt](https://github.com/mappu/miqt) bindings against system Qt.

## Prerequisites

- Go toolchain
- System Qt6 dev libraries (Fedora: `qt6-qtbase-devel`, Debian/Ubuntu: `qt6-base-dev`, Arch: `qt6-base`)
- [go-task](https://taskfile.dev/) for task management

The first `go build` compiles the miqt bindings and takes a while; subsequent builds are cached.

## Dev

```bash
task dev
```

Runs the app (`main.go`, shows an empty `QMainWindow`).

## Build

```bash
task build
```

Produces a standalone executable at `dist/rapid`.

## Test

```bash
task test
```