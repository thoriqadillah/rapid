from __future__ import annotations
from socket import socket

import functools
import shutil
import sys
import time
from pathlib import Path
from threading import Thread
from typing import Iterator, Callable

import pytest
from http.server import BaseHTTPRequestHandler, SimpleHTTPRequestHandler, ThreadingHTTPServer
from PySide6.QtGui import QGuiApplication
from typing import cast

from rapid.backend.download import Aria2Client, DownloadStore


def _app() -> QGuiApplication:
    existing = QGuiApplication.instance()
    if existing is not None and isinstance(existing, QGuiApplication):
        return existing
    return QGuiApplication(sys.argv)


def _free_port() -> int:
    with socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def _wait_daemon(app: QGuiApplication, port: int, timeout: float) -> None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        app.processEvents()
        with socket() as s:
            s.settimeout(0.2)
            try:
                s.connect(("127.0.0.1", port))
                return
            except OSError:
                time.sleep(0.05)


def _wait_until(app: QGuiApplication, predicate: Callable[[], bool], timeout: float) -> bool:
    deadline = time.time() + timeout
    while time.time() < deadline:
        app.processEvents()
        if predicate():
            return True
        time.sleep(0.02)

    app.processEvents()
    return predicate()


@pytest.fixture(scope="module")
def daemon(
    tmp_path_factory: pytest.TempPathFactory,
) -> Iterator[tuple[QGuiApplication, Aria2Client, DownloadStore, list[str]]]:
    app = _app()
    base = tmp_path_factory.mktemp("daemon")
    (base / "dl").mkdir()
    store = DownloadStore(path=base / "downloads.db")
    port = _free_port()
    client = Aria2Client(
        store=store, host="127.0.0.1", port=port, download_dir=base / "dl"
    )
    errors: list[str] = []
    client.errorOccurred.connect(errors.append)
    client.start()
    _wait_daemon(app, port, timeout=20)
    yield app, client, store, errors
    client.stop()


def test_make_url() -> None:
    assert Aria2Client._make_url("http", "127.0.0.1", 6800) == "http://127.0.0.1:6800/jsonrpc"
    assert Aria2Client._make_url("ws", "127.0.0.1", 6800) == "ws://127.0.0.1:6800/jsonrpc"


def test_gid_like() -> None:
    assert Aria2Client._gid_like("abc") == "abc"
    assert Aria2Client._gid_like({"gid": "def"}) == "def"
    assert Aria2Client._gid_like({}) == ""
    assert Aria2Client._gid_like(None) == "None"


@pytest.mark.skipif(shutil.which("aria2c") is None, reason="aria2c not installed")
def test_daemon_download_persists_status(
    tmp_path: Path, daemon: tuple[QGuiApplication, Aria2Client, DownloadStore, list[str]]
) -> None:
    app, client, store, errors = daemon

    payload = b"x" * (1024 * 1024)
    file_path = tmp_path / "payload.bin"
    file_path.write_bytes(payload)

    handler = functools.partial(SimpleHTTPRequestHandler, directory=str(tmp_path))
    server = ThreadingHTTPServer(("127.0.0.1", 0), handler)
    http_port = server.server_address[1]
    thread = Thread(target=server.serve_forever, daemon=True)
    thread.start()

    try:
        client.add_uri(f"http://127.0.0.1:{http_port}/payload.bin")

        def done() -> bool:
            items = store.all()
            return any(
                item.status == "complete" and item.total_length == len(payload)
                for item in items.values()
            )

        assert _wait_until(app, done, timeout=15), f"timeout; errors={errors} store={store.all()}"

        saved = list(store.all().values())[0]
        assert saved.status == "complete"
        assert saved.download_speed in (0, None)
    finally:
        server.shutdown()
        server.server_close()


class _ThrottledServer(ThreadingHTTPServer):
    daemon_threads = True

    def __init__(self, payload: bytes, bytes_per_second: int) -> None:
        self.payload = payload
        self.bytes_per_second = bytes_per_second
        super().__init__(("127.0.0.1", 0), _ThrottledHandler)

    @property
    def port(self) -> int:
        return self.server_address[1]


class _ThrottledHandler(BaseHTTPRequestHandler):
    def log_message(self, format: str, *args: object) -> None:
        pass

    def do_GET(self) -> None:
        throttled = cast(_ThrottledServer, self.server)
        payload: bytes = throttled.payload
        self.send_response(200)
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        chunk = 64 * 1024
        per_chunk = chunk / throttled.bytes_per_second

        try:
            for i in range(0, len(payload), chunk):
                self.wfile.write(payload[i : i + chunk])
                self.wfile.flush()
                time.sleep(per_chunk)
        except (BrokenPipeError, ConnectionResetError):
            pass


@pytest.mark.skipif(shutil.which("aria2c") is None, reason="aria2c not installed")
def test_large_download_records_speed_samples(
    daemon: tuple[QGuiApplication, Aria2Client, DownloadStore, list[str]],
) -> None:
    app, client, store, errors = daemon

    payload = b"y" * (3 * 1024 * 1024)
    server = _ThrottledServer(payload, bytes_per_second=256 * 1024)
    thread = Thread(target=server.serve_forever, daemon=True)
    thread.start()

    try:
        gid: list[str] = []
        client.downloadAdded.connect(lambda g: gid.append(g))
        client.add_uri(f"http://127.0.0.1:{server.port}/big.bin")

        def enough_samples() -> bool:
            return bool(gid) and len(store.speed_history(gid[0])) >= 4

        assert _wait_until(app, enough_samples, timeout=20), (
            f"timeout; errors={errors} gid={gid}"
        )

        history = store.speed_history(gid[0])
        assert len(history) >= 4
        assert all(s.speed > 0 for s in history)
        assert all(0 < s.ts <= int(time.time() * 1000) for s in history)
    finally:
        server.shutdown()
        server.server_close()
