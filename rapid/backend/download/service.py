from __future__ import annotations
from pprint import pprint
import shutil
import subprocess
import time
from pathlib import Path
from typing import Any
from dataclasses import fields as _fields, replace
from concurrent.futures import ThreadPoolExecutor
from urllib.parse import urlparse

from PySide6.QtQml import QQmlApplicationEngine


from PySide6.QtCore import (
    QAbstractListModel,
    QByteArray,
    QModelIndex,
    QObject,
    QPersistentModelIndex,
    QTimer,
    Qt,
    Signal,
    Slot,
)
from PySide6.QtWidgets import QFileDialog

from ..database.database import Database
from ..plugin import PluginManager
from .aria2_downloader import Aria2Downloader
from .downloader import Downloader, Resolver
from .models import Download, ResolvedUrl
from .store import DownloadStore

POLL_INTERVAL_MS = 1000

_ROLE_BY_NAME: dict[str, int] = {
    field.name: int(Qt.ItemDataRole.UserRole) + i + 1
    for i, field in enumerate(_fields(Download))
}
_ROLE_BY_NAME["progress"] = int(Qt.ItemDataRole.UserRole) + len(_ROLE_BY_NAME) + 1
_NAME_BY_ROLE: dict[int, str] = {role: name for name, role in _ROLE_BY_NAME.items()}
_ROLE_NAMES: tuple[int, ...] = tuple(_ROLE_BY_NAME.values())


class DownloadService(QAbstractListModel):
    """Application-facing download service.

    It coordinates:
      - Downloader
      - DownloadStore
      - PluginManager
      - Qt signals

    The download list is exposed directly as a Qt list model, so QML can
    bind a ``ListView`` to the service and read per-download roles.
    """

    downloadAdded = Signal(str)
    downloadChanged = Signal(str)
    downloadRemoved = Signal(str)
    errorOccurred = Signal(str)
    resolved = Signal(list, dict)  # resolvedUris, errors
    _notified = Signal(object)  # internal: Download, always delivered on the main thread


    def __init__(
        self,
        downloadDir: Path,
        store: DownloadStore,
        downloader: Downloader,
        resolver: Resolver,
        pluginDirs: list[Path] | None = None,
        parent: QObject | None = None,
    ) -> None:
        super().__init__(parent)
        self._downloadDir = downloadDir
        self._store = store
        self._downloader = downloader
        self._resolver = resolver
        self._pool = ThreadPoolExecutor(max_workers=2)
        self._downloads: list[Download] = list(self._store.all().values())
        self._notified.connect(self._onNotify)
        self._timer = QTimer(self)
        self._timer.setInterval(POLL_INTERVAL_MS)
        self._timer.timeout.connect(self._poll)
        self._pluginManager = PluginManager(pluginDirs or [], self)

    def start(self) -> None:
        self._downloader.start()
        for download in self._downloads:
            if download.status in ("complete", "error", "removed"):
                continue
            # Restored from the DB on a fresh app run; resume live updates
            # only for downloads the daemon still knows about.
            try:
                self._downloader.getStatus(download.gid)
            except Exception:
                continue
            self._downloader.listen(download.gid, self._notified.emit, self._onError)
        self._timer.start()

    def stop(self) -> None:
        self._timer.stop()
        self._downloader.stop()

    # --- model interface --------------------------------------------------

    def rowCount(self, parent: QModelIndex | QPersistentModelIndex = QModelIndex()) -> int:
        return 0 if parent.isValid() else len(self._downloads)

    def data(self, index: QModelIndex | QPersistentModelIndex, role: int = Qt.ItemDataRole.DisplayRole) -> Any:
        download = self._downloadAt(index)
        if download is None:
            return None
        if role == _ROLE_BY_NAME["progress"]:
            return download.progress
        name = _NAME_BY_ROLE.get(role)
        if name is None:
            return None
        value = getattr(download, name)
        if name == "files":
            return [f.asDict() for f in value]
        if name == "resolved":
            return value.asDict() if value else None
        return value

    def roleNames(self) -> dict[int, QByteArray]:
        return {role: QByteArray(name.encode("utf-8")) for name, role in _ROLE_BY_NAME.items()}

    # --- QML-facing slots --------------------------------------------------

    @Slot(str)
    def resolve(self, url: str) -> None:
        """Resolve a URL in a background thread; result arrives via `resolved`."""
        self._pool.submit(self._doResolve, url)

    def _doResolve(self, url: str) -> None:
        if not url.strip():
            self.resolved.emit([], {"url": "URL is required"})
            return

        try:
            uris = [r.asDict() for r in self._resolver.resolve(url)]
            uris.extend([r.asDict() for r in self._pluginManager.resolve(url)])
            self.resolved.emit(uris, {})
        except Exception as exc:
            self.resolved.emit([], {"url": str(exc)})
            return

    @Slot(list)
    def download(self, resolvedUri: list[dict[str, Any]]) -> None:
        for resolved in resolvedUri:
            self._download(ResolvedUrl.fromDict(resolved))

    @Slot(str, result=list)
    def speedHistory(self, gid: str) -> list[dict[str, int]]:
        return [sample.asDict() for sample in self._store.speedHistory(gid)]

    @Slot(int, result=str)
    def formatSize(self, bytes_: int) -> str:
        """Human-readable size, e.g. 1536 -> "1.5 KB"."""
        if not bytes_:
            return ""
        units = ("B", "KB", "MB", "GB", "TB")
        value = float(bytes_)
        i = 0
        while value >= 1024 and i < len(units) - 1:
            value /= 1024
            i += 1
        digits = 0 if value >= 100 or i == 0 else 1
        return f"{value:.{digits}f} {units[i]}"

    @Slot(str, result=str)
    def pickFolder(self, start_dir: str) -> str:
        """Open the OS-native directory chooser and return the picked path."""
        for tool, args in (
            ("zenity", ["--file-selection", "--directory"]),
            ("kdialog", ["--getexistingdirectory"]),
        ):
            program = shutil.which(tool)
            if program is None:
                continue

            result = subprocess.run([program, *args], capture_output=True, timeout=60)
            return result.stdout.decode().strip() if result.returncode == 0 else ""

        return QFileDialog.getExistingDirectory(
            None, "Select destination", start_dir or str(self._downloadDir)
        )

    @Slot(str)
    def pause(self, gid: str) -> None:
        self._downloader.pause(gid)

    @Slot(str)
    def resume(self, gid: str) -> None:
        self._downloader.resume(gid)

    @Slot(str)
    def remove(self, gid: str) -> None:
        self._downloader.remove(gid)
        self._store.remove(gid)
        self._remove(gid)
        self.downloadRemoved.emit(gid)

    @Slot(str)
    def purge(self, gid: str) -> None:
        self._downloader.purge(gid)
        self._store.remove(gid)
        self._remove(gid)
        self.downloadRemoved.emit(gid)

    # --- internals ----------------------------------------------------------

    def _downloadAt(self, index: QModelIndex | QPersistentModelIndex) -> Download | None:
        if not index.isValid():
            return None
        row = index.row()
        if not 0 <= row < len(self._downloads):
            return None
        return self._downloads[row]

    def _row_of(self, gid: str) -> int | None:
        for i, download in enumerate(self._downloads):
            if download.gid == gid:
                return i
        return None

    def _download(self, resolved: ResolvedUrl) -> str:
        download = self._downloader.download(resolved)
        download = replace(download, resolved=resolved)
        self._store.upsert(download)
        self._downloader.listen(download.gid, self._notified.emit, self._onError)
        self._insert(download)
        self.downloadAdded.emit(download.gid)
        return download.gid

    def _insert(self, download: Download) -> None:
        row = len(self._downloads)
        self.beginInsertRows(QModelIndex(), row, row)
        self._downloads.append(download)
        self.endInsertRows()

    def _remove(self, gid: str) -> None:
        row = self._row_of(gid)
        if row is None:
            return
        self.beginRemoveRows(QModelIndex(), row, row)
        del self._downloads[row]
        self.endRemoveRows()

    def _onNotify(self, download: Download) -> None:
        row = self._row_of(download.gid)
        if row is not None:
            if download.resolved is None:
                download = replace(download, resolved=self._downloads[row].resolved)
            if download.status == "complete" and not download.downloadSpeed:
                download = replace(download, downloadSpeed=self._downloads[row].downloadSpeed)

            self._store.upsert(download)
            self._store.addSpeedSample(download.gid, int(time.time() * 1000), download.downloadSpeed or 0)
            self._downloads[row] = download
            index = self.index(row)
            self.dataChanged.emit(index, index, list(_ROLE_NAMES))
        self.downloadChanged.emit(download.gid)

    def _onError(self, message: str) -> None:
        self.errorOccurred.emit(message)

    def _poll(self) -> None:
        self._pool.submit(self._downloader.refresh)
