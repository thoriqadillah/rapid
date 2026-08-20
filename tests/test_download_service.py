from __future__ import annotations

from pathlib import Path

from PySide6.QtCore import QCoreApplication
from PySide6.QtGui import QGuiApplication

from rapid.backend import DownloadService, DownloadStore
from rapid.backend.database.database import Database
from rapid.backend.download.downloader import (
    Downloader,
    ErrorCallback,
    NotifyCallback,
    Resolver,
)
from rapid.backend.download.models import Download, ResolvedUrl

SAMPLE = Path(__file__).resolve().parent.parent / "rapid" / "plugins"


class FakeDownloader(Downloader, Resolver):
    def __init__(self) -> None:
        self.started = False
        self.stopped = False
        self.added: list[str] = []
        self.removed: list[str] = []
        self.purged: list[str] = []
        self._statuses: dict[str, Download] = {}
        self._listeners: dict[str, tuple[NotifyCallback, ErrorCallback]] = {}

    def start(self) -> None:
        self.started = True

    def stop(self) -> None:
        self.stopped = True

    def shouldResolve(self, uri: str) -> bool:
        return uri.startswith("http")

    def validate(self, uri: str) -> dict[str, str]:
        return {} if uri.startswith("http") else {"url": "Enter a URL"}

    def resolve(self, uri: str) -> list[ResolvedUrl]:
        return [ResolvedUrl(url=uri, title="x", category="other")]

    def download(self, uri: ResolvedUrl) -> Download:
        gid = f"g{len(self.added) + 1}"
        download = Download(gid=gid, status="active", totalLength=100)
        self._statuses[gid] = download
        self.added.append(uri.url)
        return download

    def pause(self, id: str) -> None:
        pass

    def resume(self, id: str) -> None:
        pass

    def remove(self, id: str) -> None:
        self.removed.append(id)
        self._listeners.pop(id, None)

    def purge(self, id: str) -> None:
        self.purged.append(id)
        self._listeners.pop(id, None)

    def getStatus(self, id: str) -> Download:
        return self._statuses[id]

    def listen(self, id: str, onNotify: NotifyCallback, onError: ErrorCallback) -> None:
        self._listeners[id] = (onNotify, onError)

    def refresh(self, id: str | None = None) -> None:
        for gid, (onNotify, _) in self._listeners.items():
            if id is not None and gid != id:
                continue
            onNotify(self._statuses[gid])


def _service(tmp_path: Path) -> tuple[DownloadService, FakeDownloader, DownloadStore]:
    app = QGuiApplication.instance()
    if app is None:
        QGuiApplication([])
    fake = FakeDownloader()
    store = DownloadStore(Database(path=tmp_path / "database.db"))
    service = DownloadService(
        downloadDir=tmp_path,
        pluginDirs=[SAMPLE],
        store=store,
        downloader=fake,
        resolver=fake,
    )
    return service, fake, store


def _await_resolve(service: DownloadService, url: str) -> tuple[list, dict]:
    app = QCoreApplication.instance()
    done: dict = {}

    def on_resolved(uris: list, errors: dict) -> None:
        done["uris"] = uris
        done["errors"] = errors

    service.resolved.connect(on_resolved)
    service.resolve(url)
    while "uris" not in done and app:
        app.processEvents()
    return done["uris"], done["errors"]


def test_start_stops_downloader(tmp_path: Path) -> None:
    service, fake, _ = _service(tmp_path)
    service.start()
    assert fake.started
    service.stop()
    assert fake.stopped


def test_add_url_downloads_and_persists(tmp_path: Path) -> None:
    service, fake, store = _service(tmp_path)

    added: list[str] = []
    service.downloadAdded.connect(added.append)

    resolved = _await_resolve(service, "http://example.com/a.bin")[0]
    service.download(resolved)

    assert fake.added == ["http://example.com/a.bin"]
    assert added == ["g1"]
    assert store.get("g1") == Download(
        gid="g1",
        status="active",
        totalLength=100,
        resolved=ResolvedUrl(url="http://example.com/a.bin", title="x", category="other"),
    )


def test_model_exposes_downloads(tmp_path: Path) -> None:
    service, fake, _ = _service(tmp_path)

    service.download(_await_resolve(service, "http://example.com/a.bin")[0])
    service.download(_await_resolve(service, "http://example.com/b.bin")[0])

    def role(name: str) -> int:
        return {bytes(v).decode("utf-8"): k for k, v in service.roleNames().items()}[name]

    assert service.rowCount() == 2
    assert service.data(service.index(0), role("gid")) == "g1"
    assert service.data(service.index(0), role("status")) == "active"
    assert service.data(service.index(0), role("totalLength")) == 100
    assert service.data(service.index(1), role("gid")) == "g2"
    assert service.data(service.index(2), role("gid")) is None

    service.remove("g1")
    assert service.rowCount() == 1
    assert service.data(service.index(0), role("gid")) == "g2"


def test_poll_propagates_status_and_speed_samples(tmp_path: Path) -> None:
    service, fake, store = _service(tmp_path)
    service.start()
    service.download(_await_resolve(service, "http://example.com/a.bin")[0])

    changed: list[str] = []
    service.downloadChanged.connect(changed.append)

    fake._statuses["g1"] = Download(gid="g1", status="active", totalLength=100, downloadSpeed=512)
    service._poll()
    app = QCoreApplication.instance()
    while not changed and app:
        app.processEvents()

    assert changed == ["g1"]
    assert store.get("g1") == Download(
        gid="g1",
        status="active",
        totalLength=100,
        downloadSpeed=512,
        resolved=ResolvedUrl(url="http://example.com/a.bin", title="x", category="other"),
    )
    assert store.speedHistory("g1")[0].speed == 512
    roles = {bytes(v).decode("utf-8"): k for k, v in service.roleNames().items()}
    assert service.data(service.index(0), roles["downloadSpeed"]) == 512

    service.stop()


def test_completion_keeps_last_speed(tmp_path: Path) -> None:
    service, fake, store = _service(tmp_path)
    service.start()
    service.download(_await_resolve(service, "http://example.com/a.bin")[0])

    fake._statuses["g1"] = Download(
        gid="g1", status="active", totalLength=100, downloadSpeed=512
    )
    service._poll()
    app = QCoreApplication.instance()
    for _ in range(10):
        if app:
            app.processEvents()

    fake._statuses["g1"] = Download(gid="g1", status="complete", totalLength=100)
    service._poll()
    for _ in range(10):
        if app:
            app.processEvents()

    roles = {bytes(v).decode("utf-8"): k for k, v in service.roleNames().items()}
    assert service.data(service.index(0), roles["downloadSpeed"]) == 512
    download = store.get("g1")
    assert download is not None
    assert download.downloadSpeed == 512

    service.stop()


def test_pause_freezes_speed(tmp_path: Path) -> None:
    service, fake, store = _service(tmp_path)
    service.start()
    service.download(_await_resolve(service, "http://example.com/a.bin")[0])

    fake._statuses["g1"] = Download(gid="g1", status="active", totalLength=100, downloadSpeed=512)
    service._poll()
    app = QCoreApplication.instance()
    for _ in range(10):
        if app:
            app.processEvents()

    fake._statuses["g1"] = Download(gid="g1", status="paused", totalLength=100, downloadSpeed=128)
    service._poll()
    for _ in range(10):
        if app:
            app.processEvents()

    roles = {bytes(v).decode("utf-8"): k for k, v in service.roleNames().items()}
    assert service.data(service.index(0), roles["downloadSpeed"]) == 512
    assert store.get("g1").downloadSpeed == 512

    service.stop()


def test_start_relistens_restored_downloads(tmp_path: Path) -> None:
    app = QGuiApplication.instance()
    if app is None:
        QGuiApplication([])
    fake = FakeDownloader()
    store = DownloadStore(Database(path=tmp_path / "relisten.db"))
    store.upsert(Download(gid="g1", status="active", totalLength=100))
    store.upsert(Download(gid="g2", status="complete", totalLength=100))
    fake._statuses["g1"] = Download(gid="g1", status="active", totalLength=100)
    service = DownloadService(
        downloadDir=tmp_path,
        pluginDirs=[SAMPLE],
        store=store,
        downloader=fake,
        resolver=fake,
    )
    assert service.rowCount() == 2

    service.start()

    assert "g1" in fake._listeners  # still live in the daemon -> re-listened
    assert "g2" not in fake._listeners  # terminal -> skipped
    service.stop()


def test_start_skips_restored_downloads_missing_from_daemon(tmp_path: Path) -> None:
    app = QGuiApplication.instance()
    if app is None:
        QGuiApplication([])
    fake = FakeDownloader()
    store = DownloadStore(Database(path=tmp_path / "relisten2.db"))
    store.upsert(Download(gid="g1", status="active", totalLength=100))
    service = DownloadService(
        downloadDir=tmp_path,
        pluginDirs=[SAMPLE],
        store=store,
        downloader=fake,
        resolver=fake,
    )

    service.start()

    assert "g1" not in fake._listeners  # getStatus raised -> not re-listened
    service.stop()


def test_remove_stops_but_keeps_in_store(tmp_path: Path) -> None:
    service, fake, store = _service(tmp_path)
    service.download(_await_resolve(service, "http://example.com/a.bin")[0])

    removed: list[str] = []
    service.downloadRemoved.connect(removed.append)

    service.remove("g1")

    assert fake.removed == ["g1"]
    assert "g1" in store.all()
    assert removed == []


def test_purge_deletes_from_store(tmp_path: Path) -> None:
    service, fake, store = _service(tmp_path)
    service.download(_await_resolve(service, "http://example.com/a.bin")[0])
    service.purge("g1")
    assert fake.purged == ["g1"]
    assert store.all() == {}


def test_resolve_returns_uris_and_errors_tuple(tmp_path: Path) -> None:
    service, _, _ = _service(tmp_path)

    uris, errors = _await_resolve(service, "http://example.com/a.bin")
    assert errors == {}
    assert [u["url"] for u in uris] == ["http://example.com/a.bin"]

    uris, errors = _await_resolve(service, "")
    assert uris == []
    assert errors == {"url": "URL is required"}


def test_format_size(tmp_path: Path) -> None:
    service, _, _ = _service(tmp_path)
    assert service.formatSize(0) == ""
    assert service.formatSize(512) == "512 B"
    assert service.formatSize(1536) == "1.5 KB"
    assert service.formatSize(1024 * 1024) == "1.0 MB"
    assert service.formatSize(1024 ** 4) == "1.0 TB"
