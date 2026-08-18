from __future__ import annotations

import functools
import json
import shutil
import time
from http.server import BaseHTTPRequestHandler, SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from socket import socket
from threading import Thread
from typing import Callable, Iterator, cast
from unittest import mock
from urllib.error import HTTPError, URLError

import pytest
from http.client import HTTPMessage

from rapid.backend.download import aria2_downloader
from rapid.backend.download.aria2_downloader import Aria2Downloader, Aria2Error, Aria2Rpc
from rapid.backend.download.downloader import ErrorCallback, GlobalNotifyCallback, NotifyCallback
from rapid.backend.download.models import Download, ResolvedUrl


class _Resp:
    def __init__(self, body: bytes) -> None:
        self._body = body

    def __enter__(self) -> _Resp:
        return self

    def __exit__(self, *args: object) -> None:
        return None

    def read(self) -> bytes:
        return self._body


class FakeRpc:
    def __init__(self, *responses: object) -> None:
        self.responses: list[object] = list(responses)
        self.calls: list[tuple[str, list[object]]] = []

    def call(self, method: str, params: list[object]) -> object:
        self.calls.append((method, params))
        if self.responses:
            return self.responses.pop(0)
        return {}


def _downloader(
    monkeypatch: pytest.MonkeyPatch,
    rpc: FakeRpc | None = None,
    *,
    onGlobalNotify: GlobalNotifyCallback | None = None,
) -> Aria2Downloader:
    fake = rpc or FakeRpc()
    monkeypatch.setattr(aria2_downloader, "Aria2Rpc", lambda **kwargs: fake)
    return Aria2Downloader(
        manageDaemon=False,
        onGlobalNotify=onGlobalNotify,
    )


def _call_methods(dl: Aria2Downloader) -> list[str]:
    rpc = cast(FakeRpc, dl._rpc)
    return [name for name, _ in rpc.calls]

# --- resolver ------------------------------------------------------------

def test_resolve_returns_resolved_url_and_cleans_up(monkeypatch: pytest.MonkeyPatch) -> None:
    status = {
        "gid": "gid-1",
        "status": "complete",
        "totalLength": "2048",
        "completedLength": "2048",
        "files": [
            {
                "path": "/tmp/song.mp3",
                "length": "2048",
                "uris": [{"uri": "http://cdn/song.mp3", "status": "used"}],
            }
        ],
    }
    dl = _downloader(monkeypatch, FakeRpc("gid-1", status))

    items = dl.resolve("http://start")

    assert len(items) == 1
    item = items[0]
    assert item.url == "http://cdn/song.mp3"
    assert item.filename == "song.mp3"
    assert item.title == "song.mp3"
    assert item.size == 2048
    assert item.category == "audio"
    assert item.mimeType == "audio/mpeg"
    assert _call_methods(dl) == [
        "aria2.addUri",
        "aria2.tellStatus",
        "aria2.remove",
        "aria2.removeDownloadResult",
    ]


def test_resolve_falls_back_to_uri_when_no_file_info(monkeypatch: pytest.MonkeyPatch) -> None:
    dl = _downloader(monkeypatch, FakeRpc("gid-1", {"gid": "gid-1", "status": "complete", "files": []}))

    items = dl.resolve("http://x/video.mp4")

    assert len(items) == 1
    assert items[0].url == "http://x/video.mp4"
    assert items[0].filename == "video.mp4"
    assert items[0].category == "video"


def test_on_resolved_halts_active_download(monkeypatch: pytest.MonkeyPatch) -> None:
    class Rpc(FakeRpc):
        def call(self, method: str, params: list[object]) -> object:
            self.calls.append((method, params))
            if method == "aria2.removeDownloadResult":
                raise Aria2Error("HTTP 400")
            return {}

    rpc = Rpc()
    _downloader(monkeypatch, rpc)._onResolved("gid-1")

    assert rpc.calls == [("aria2.remove", ["gid-1"])] + [
        ("aria2.removeDownloadResult", ["gid-1"])] * 5


def test_resolve_dry_run_option_is_forced(monkeypatch: pytest.MonkeyPatch) -> None:
    dl = _downloader(monkeypatch, FakeRpc("gid-1", {"gid": "gid-1", "status": "complete", "files": []}))
    dl.resolve("http://x/f.mp4", options={"dry-run": "false"})
    _, params = cast(FakeRpc, dl._rpc).calls[0]
    options = cast(dict[str, object], params[1])
    assert options["dry-run"] == "true"
    assert options["use-head"] == "true"


# --- download ------------------------------------------------------------

def test_download_returns_status_without_listening(monkeypatch: pytest.MonkeyPatch) -> None:
    dl = _downloader(monkeypatch, FakeRpc("gid-9", {"gid": "gid-9", "status": "active"}))

    download = dl.download(ResolvedUrl(url="http://x/f.mp4"))

    assert isinstance(download, Download)
    assert download.gid == "gid-9"
    assert dl._listening == set()
    assert _call_methods(dl) == ["aria2.addUri", "aria2.tellStatus"]


def test_listen_registers_callbacks(monkeypatch: pytest.MonkeyPatch) -> None:
    dl = _downloader(monkeypatch)
    notify: NotifyCallback = lambda s: None
    error: ErrorCallback = lambda m: None

    dl.listen("g", notify, error)

    assert dl._listening == {("g", notify, error)}


def test_download_raises_without_gid(monkeypatch: pytest.MonkeyPatch) -> None:
    dl = _downloader(monkeypatch, FakeRpc(None))
    with pytest.raises(Aria2Error, match="GID"):
        dl.download(ResolvedUrl(url="http://x/f.mp4"))


# --- status --------------------------------------------------------------

def test_get_status_parses_payload(monkeypatch: pytest.MonkeyPatch) -> None:
    payload = {
        "gid": "g",
        "status": "complete",
        "totalLength": "100",
        "completedLength": "100",
        "downloadSpeed": "0",
    }
    dl = _downloader(monkeypatch, FakeRpc(payload))

    status = dl.getStatus("g")

    assert status.status == "complete"
    assert status.totalLength == 100
    assert status.progress == 1.0


def test_get_status_non_dict_raises(monkeypatch: pytest.MonkeyPatch) -> None:
    dl = _downloader(monkeypatch, FakeRpc([]))
    with pytest.raises(Aria2Error, match="no status"):
        dl.getStatus("g")


# --- callbacks -----------------------------------------------------------

def test_refresh_notifies_listeners(monkeypatch: pytest.MonkeyPatch) -> None:
    notified: list[Download] = []
    global_seen: list[dict[str, object]] = []
    dl = _downloader(
        monkeypatch,
        FakeRpc({"gid": "g", "status": "active"}, {"downloadSpeed": 123}),
        onGlobalNotify=global_seen.append,
    )
    dl._running = True
    dl.listen("g", notified.append, lambda m: None)

    dl.refresh()

    assert [s.gid for s in notified] == ["g"]
    assert global_seen == [{"downloadSpeed": 123}]


def test_refresh_emits_errors(monkeypatch: pytest.MonkeyPatch) -> None:
    errors: list[str] = []
    dl = _downloader(monkeypatch, FakeRpc([]))
    dl._running = True
    dl.listen("g", lambda s: None, errors.append)

    dl.refresh()

    assert len(errors) == 1
    assert "no status" in errors[0]


def test_refresh_keeps_listening_while_active(monkeypatch: pytest.MonkeyPatch) -> None:
    notified: list[Download] = []
    dl = _downloader(monkeypatch, FakeRpc({"gid": "g", "status": "active"}))
    dl._running = True
    dl.listen("g", notified.append, lambda m: None)

    dl.refresh()

    assert notified[0].status == "active"
    assert dl._listening != set()


def test_refresh_unlistens_after_terminal_error(monkeypatch: pytest.MonkeyPatch) -> None:
    notified: list[Download] = []
    dl = _downloader(
        monkeypatch,
        FakeRpc({"gid": "g", "status": "error", "errorMessage": "status=403"}),
    )
    dl._running = True
    dl.listen("g", notified.append, lambda m: None)

    dl.refresh()

    assert [s.gid for s in notified] == ["g"]
    assert notified[0].status == "error"
    assert dl._listening == set()


def test_refresh_unlistens_when_status_rpc_fails(monkeypatch: pytest.MonkeyPatch) -> None:
    errors: list[str] = []
    dl = _downloader(monkeypatch, FakeRpc([]))
    dl._running = True
    dl.listen("g", lambda s: None, errors.append)

    dl.refresh()

    assert len(errors) == 1
    assert dl._listening == set()


def test_ws_event_triggers_status_push(monkeypatch: pytest.MonkeyPatch) -> None:
    notified: list[Download] = []
    dl = _downloader(
        monkeypatch,
        FakeRpc({"gid": "g", "status": "complete"}),
    )
    dl._running = True
    dl.listen("g", notified.append, lambda m: None)

    dl._onWsMessage(
        json.dumps({"method": "aria2.onDownloadComplete", "params": [{"gid": "g"}]})
    )

    assert [s.gid for s in notified] == ["g"]


def test_ws_non_event_message_is_ignored(monkeypatch: pytest.MonkeyPatch) -> None:
    dl = _downloader(monkeypatch)
    dl._running = True
    dl.listen("g", lambda s: None, lambda m: None)

    dl._onWsMessage(
        json.dumps({"method": "aria2.tellStatus", "params": [{"gid": "g"}]})
    )

    assert cast(FakeRpc, dl._rpc).calls == []


# --- control -------------------------------------------------------------

def test_spawn_daemon_adopts_existing_one(monkeypatch: pytest.MonkeyPatch) -> None:
    dl = _downloader(monkeypatch)
    dl._spawnDaemon()
    assert dl._ownsDaemon is False
    assert dl._process is None
    assert _call_methods(dl) == ["aria2.getVersion"]


def test_spawn_daemon_owns_spawned_process(monkeypatch: pytest.MonkeyPatch) -> None:
    class Rpc(FakeRpc):
        def call(self, method: str, params: list[object]) -> object:
            self.calls.append((method, params))
            # First call (adopt check) fails, readiness checks succeed.
            if sum(1 for m, _ in self.calls if m == "aria2.getVersion") == 1:
                raise Aria2Error("no daemon yet")
            return {}

    class FakePopen:
        def __init__(self, *args: object, **kwargs: object) -> None:
            pass

        def poll(self) -> None:
            return None

    monkeypatch.setattr(aria2_downloader.subprocess, "Popen", FakePopen)
    dl = _downloader(monkeypatch, Rpc())
    dl._spawnDaemon()
    assert dl._ownsDaemon is True
    assert isinstance(dl._process, FakePopen)
    dl.stop()
    assert dl._process is None


def test_pause_resume_remove_purge(monkeypatch: pytest.MonkeyPatch) -> None:
    dl = _downloader(monkeypatch)
    cb = lambda s: None

    dl.listen("g", cb, cb)
    dl.pause("g")
    dl.resume("g")
    dl.remove("g")
    assert dl._listening == set()

    dl.listen("g", cb, cb)
    dl.purge("g")
    assert dl._listening == set()

    assert _call_methods(dl) == [
        "aria2.pause",
        "aria2.unpause",
        "aria2.remove",
        "aria2.removeDownloadResult",
    ]

# --- integration (real aria2c daemon) --------------------------------------

def _free_port() -> int:
    with socket() as s:
        s.bind(("127.0.0.1", 0))
        return int(s.getsockname()[1])


def _wait_port(port: int, timeout: float = 10) -> None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        with socket() as s:
            s.settimeout(0.2)
            try:
                s.connect(("127.0.0.1", port))
                return
            except OSError:
                time.sleep(0.05)
    raise AssertionError(f"aria2 RPC port {port} never opened")


def _wait_until(predicate: Callable[[], bool], timeout: float = 15) -> None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        if predicate():
            return
        time.sleep(0.05)
    raise AssertionError("condition not met before timeout")


def _serve_file(tmp_path: Path, name: str, payload: bytes) -> tuple[ThreadingHTTPServer, int]:
    tmp_path.joinpath(name).write_bytes(payload)
    handler = functools.partial(SimpleHTTPRequestHandler, directory=str(tmp_path))
    server = ThreadingHTTPServer(("127.0.0.1", 0), handler)
    Thread(target=server.serve_forever, daemon=True).start()
    return server, server.server_address[1]


def _spawn(
    tmp_path: Path,
    *,
    onGlobalNotify: GlobalNotifyCallback | None = None,
) -> Aria2Downloader:
    download_dir = tmp_path / "dl"
    download_dir.mkdir(parents=True, exist_ok=True)
    dl = Aria2Downloader(
        port=_free_port(),
        downloadDir=download_dir,
        manageDaemon=True,
        onGlobalNotify=onGlobalNotify,
    )
    dl.start()
    _wait_port(dl._port)
    return dl


@pytest.fixture(scope="module")
def daemon(tmp_path_factory: pytest.TempPathFactory) -> Iterator[Aria2Downloader]:
    dl = _spawn(tmp_path_factory.mktemp("daemon"))
    yield dl
    dl.stop()


@pytest.mark.skipif(shutil.which("aria2c") is None, reason="aria2c not installed")
def test_resolve_returns_metadata(
    tmp_path: Path, daemon: Aria2Downloader
) -> None:
    server, port = _serve_file(tmp_path, "clip.mp4", b"x" * 1024)
    try:
        items = daemon.resolve(f"http://127.0.0.1:{port}/clip.mp4")
    finally:
        server.shutdown()
        server.server_close()

    assert len(items) == 1
    assert items[0].filename == "clip.mp4"
    assert items[0].size == 1024
    assert items[0].category == "video"
    assert items[0].mimeType == "video/mp4"


@pytest.mark.skipif(shutil.which("aria2c") is None, reason="aria2c not installed")
def test_download_completes_and_file_exists(
    tmp_path: Path, daemon: Aria2Downloader
) -> None:
    server, port = _serve_file(tmp_path, "payload.bin", b"y" * (512 * 1024))
    try:
        download = daemon.download(ResolvedUrl(url=f"http://127.0.0.1:{port}/payload.bin"))
        gid = download.gid

        _wait_until(lambda: daemon.getStatus(gid).status == "complete")
        status = daemon.getStatus(gid)
        assert status.progress == 1.0
        assert daemon._downloadDir is not None
        assert (daemon._downloadDir / "payload.bin").exists()
    finally:
        server.shutdown()
        server.server_close()


@pytest.mark.skipif(shutil.which("aria2c") is None, reason="aria2c not installed")
def test_refresh_pushes_status_to_callback(tmp_path: Path) -> None:
    server, port = _serve_file(tmp_path, "note.bin", b"z" * (256 * 1024))
    statuses: list[str | None] = []
    daemon = _spawn(tmp_path)
    try:
        download = daemon.download(ResolvedUrl(url=f"http://127.0.0.1:{port}/note.bin"))
        gid = download.gid
        daemon.listen(gid, lambda s: statuses.append(s.status), lambda m: None)

        def done() -> bool:
            daemon.refresh()
            return "complete" in statuses

        _wait_until(done, timeout=20)
        assert "complete" in statuses
    finally:
        daemon.stop()
        server.shutdown()
        server.server_close()


@pytest.mark.skipif(shutil.which("aria2c") is None, reason="aria2c not installed")
def test_websocket_pushes_status_without_polling(tmp_path: Path) -> None:
    server = _ThrottledServer(b"z" * (1024 * 1024), bytes_per_second=128 * 1024)
    Thread(target=server.serve_forever, daemon=True).start()
    statuses: list[str | None] = []
    daemon = _spawn(tmp_path)
    try:
        download = daemon.download(
            ResolvedUrl(url=f"http://127.0.0.1:{server.port}/slow.bin")
        )
        gid = download.gid
        daemon.listen(gid, lambda s: statuses.append(s.status), lambda m: None)

        _wait_until(lambda: "complete" in statuses, timeout=60)
    finally:
        daemon.stop()
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
        start, end = 0, len(payload) - 1
        status = 200

        rng = self.headers.get("Range")
        if rng and rng.startswith("bytes="):
            a, b = rng[6:].split("-", 1)
            start = int(a) if a else 0
            end = int(b) if b else len(payload) - 1
            status = 206
        end = min(end, len(payload) - 1)
        body = payload[start : end + 1]

        self.send_response(status)
        if status == 206:
            self.send_header("Content-Range", f"bytes {start}-{end}/{len(payload)}")
            self.send_header("Accept-Ranges", "bytes")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        chunk = 64 * 1024
        per_chunk = chunk / throttled.bytes_per_second

        try:
            for i in range(0, len(body), chunk):
                self.wfile.write(body[i : i + chunk])
                self.wfile.flush()
                time.sleep(per_chunk)
        except (BrokenPipeError, ConnectionResetError):
            pass


@pytest.mark.skipif(shutil.which("aria2c") is None, reason="aria2c not installed")
def test_pause_resume(daemon: Aria2Downloader) -> None:
    server = _ThrottledServer(b"p" * (2 * 1024 * 1024), bytes_per_second=64 * 1024)
    Thread(target=server.serve_forever, daemon=True).start()
    try:
        download = daemon.download(ResolvedUrl(url=f"http://127.0.0.1:{server.port}/slow.bin"))
        gid = download.gid

        _wait_until(lambda: (daemon.getStatus(gid).completedLength or 0) > 0)
        daemon.pause(gid)
        _wait_until(lambda: daemon.getStatus(gid).status == "paused")
        daemon.resume(gid)
        _wait_until(lambda: daemon.getStatus(gid).status == "complete", timeout=60)

        assert daemon.getStatus(gid).progress == 1.0
    finally:
        server.shutdown()
        server.server_close()
