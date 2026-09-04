from __future__ import annotations
import mimetypes
import shutil
import subprocess
import time
from pathlib import Path
from typing import Any, cast, Callable, Union
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
    Property,
    QSortFilterProxyModel,
    QTimer,
    Qt,
    Signal,
    Slot,
)

from ..database.database import Database
from ..plugin import PluginManager
from ..setting.models import Settings
from .aria2_downloader import Aria2Downloader
from .downloader import Downloader, Resolver
from .models import Download, ResolvedUrl
from .store import DownloadStore

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
    downloadCompleted = Signal(str)  # gid
    downloadFailed = Signal(str, str)  # gid, errorMessage
    errorOccurred = Signal(str)
    resolved = Signal(list, dict)  # resolvedUris, errors
    countsChanged = Signal()  # sidebar badges
    activeCountChanged = Signal(int)
    _notified = Signal(object)  # internal: Download, always delivered on the main thread


    def __init__(
        self,
        settings: Settings,
        store: DownloadStore,
        downloader: Downloader,
        resolver: Resolver,
        parent: QObject | None = None,
    ) -> None:
        super().__init__(parent)
        self._settings = settings
        self._store = store
        self._downloader = downloader
        self._resolver = resolver
        self._pool = ThreadPoolExecutor(max_workers=2)
        self._downloads: list[Download] = self._store.all()
        self._notified.connect(self._notify)
        self._timer = QTimer(self)
        self._timer.setInterval(settings.pollIntervalMs)
        self._timer.timeout.connect(self._poll)
        self._pluginManager = PluginManager(settings.pluginDirs, self)

    def start(self) -> None:
        self._downloader.start()
        for download in self._downloads:
            if download.status != "active":
                continue
            # Restored from the DB on a fresh app run; resume live updates
            # only for downloads the daemon still knows about.
            try:
                self._downloader.getStatus(download.gid)
            except Exception:
                continue
            self._downloader.listen(download.gid, self._notified.emit, self._error)
        self._timer.start()

    def close(self) -> None:
        self._timer.stop()
        self._pool.shutdown(wait=True, cancel_futures=True)
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

    @Property(dict, notify=cast(Callable[[], None], countsChanged))
    def counts(self) -> dict[str, int]:
        """Live download counts by category, e.g. {"all": 12, "video": 4}."""
        counts: dict[str, int] = {"all": len(self._downloads)}
        for download in self._downloads:
            if download.category:
                counts[download.category] = counts.get(download.category, 0) + 1
        return counts

    @Property(int, notify=cast(Callable[[int], None], activeCountChanged))
    def activeCount(self) -> int:
        return sum(1 for d in self._downloads if d.status == "active")

    @Slot(str, result=str)
    def downloadName(self, gid: str) -> str:
        """Return a human-readable name for the download identified by *gid*."""
        row = self._row_of(gid)
        if row is None:
            return gid

        download = self._downloads[row]
        if download.resolved:
            name = download.resolved.title or download.resolved.filename
            if name:
                return name

        if download.files:
            path = download.files[0].path or ""
            name = path.rsplit("/", 1)[-1] if "/" in path else path
            if name:
                return name

        return gid

    @Slot(str)
    def resolve(self, url: str) -> None:
        """Resolve a URL in a background thread; result arrives via `resolved`."""
        self._pool.submit(self._doResolve, url, None)

    @Slot(dict)
    def resolveRequest(self, request: dict[str, Any]) -> None:
        """Resolve a browser request while preserving its HTTP context."""
        url = request.get("url")
        self._pool.submit(self._doResolve, url if isinstance(url, str) else "", request)

    def _doResolve(self, url: str, options: dict[str, Any] | None) -> None:
        if not url.strip():
            self.resolved.emit([], {"url": "URL is required"})
            return

        try:
            if options and options.get("browserResolved") is True:
                resolved = self._browserResolvedUrl(url, options)
                if resolved.filename:
                    dest = (
                        Path(resolved.dir)
                        if resolved.dir
                        else Path(self._settings.downloadDir)
                    )
                    resolved = replace(resolved, filename=self._uniqueName(dest, resolved.filename))
                self.resolved.emit([resolved.asDict()], {})
                return

            pluginUris = self._pluginManager.resolve(url, options)
            fallbackUris = [] if pluginUris else (
                self._resolver.resolve(url, options) if options else self._resolver.resolve(url)
            )
            dest = Path(self._settings.downloadDir)
            resolvedUris = pluginUris or fallbackUris
            uris = []
            for r in resolvedUris:
                r = self._withRequestContext(r, options)
                if r.filename:
                    r = replace(r, filename=self._uniqueName(dest, r.filename))
                uris.append(r.asDict())

            self.resolved.emit(uris, {})
        except Exception as exc:
            self.resolved.emit([], {"url": str(exc)})
            return

    @staticmethod
    def _browserResolvedUrl(url: str, options: dict[str, Any]) -> ResolvedUrl:
        headersValue = options.get("headers")
        cookiesValue = options.get("cookies")
        headers = (
            {key: value for key, value in headersValue.items() if isinstance(key, str) and isinstance(value, str)}
            if isinstance(headersValue, dict)
            else None
        )
        cookies = (
            {key: value for key, value in cookiesValue.items() if isinstance(key, str) and isinstance(value, str)}
            if isinstance(cookiesValue, dict)
            else None
        )
        filenameValue = options.get("filename")
        filename = filenameValue if isinstance(filenameValue, str) and filenameValue else None
        originalFilename = filename
        savePath = options.get("savePath")
        destDir = None
        if isinstance(savePath, str) and savePath:
            saveFile = Path(savePath)
            destDir = str(saveFile.parent)
            if not filename:
                filename = saveFile.name
        mimeValue = options.get("mimeType")
        mimeType = (
            mimeValue
            if isinstance(mimeValue, str) and mimeValue
            else mimetypes.guess_type(filename or urlparse(url).path, strict=False)[0]
        )
        if filename and mimeType:
            suffix = Path(filename).suffix
            if not (suffix and suffix[1:].isalnum()):
                filename += mimetypes.guess_extension(mimeType.split(";", 1)[0]) or ""

        categoryValue = options.get("category")
        category = categoryValue if isinstance(categoryValue, str) and categoryValue else "unknown"
        if category == "unknown" and mimeType:
            prefix = mimeType.split("/", 1)[0]
            category = prefix if prefix in {"audio", "video", "image"} else "unknown"
        sizeValue = options.get("size")
        size = sizeValue if isinstance(sizeValue, int) and not isinstance(sizeValue, bool) and sizeValue >= 0 else None
        refererValue = options.get("referer") or options.get("pageUrl")
        titleValue = options.get("title")
        title = titleValue if isinstance(titleValue, str) and titleValue else filename
        if title == originalFilename:
            title = filename

        return ResolvedUrl(
            url=url,
            title=title,
            filename=filename,
            dir=destDir,
            mimeType=mimeType,
            size=size,
            category=category,
            headers=headers,
            cookies=cookies,
            referer=refererValue if isinstance(refererValue, str) else None,
            resolverName="Browser",
        )

    def _uniqueName(self, dest: Path, filename: str) -> str:
        target = dest / filename
        if not target.exists():
            return filename
        stem, ext = Path(filename).stem, Path(filename).suffix
        n = 1
        while True:
            candidate = f"{stem} ({n}){ext}" if n > 1 else f"{stem} (1){ext}"
            if not (dest / candidate).exists():
                return candidate
            n += 1

    @staticmethod
    def _withRequestContext(
        resolved: ResolvedUrl,
        options: dict[str, Any] | None,
    ) -> ResolvedUrl:
        if not options:
            return resolved
        headers = options.get("headers")
        cookies = options.get("cookies")
        referer = options.get("referer") or options.get("pageUrl")
        filename = options.get("filename")
        title = options.get("title")
        mimeType = options.get("mimeType")
        return replace(
            resolved,
            headers=resolved.headers if resolved.headers is not None else headers if isinstance(headers, dict) else None,
            cookies=resolved.cookies if resolved.cookies is not None else cookies if isinstance(cookies, dict) else None,
            referer=resolved.referer if resolved.referer is not None else referer if isinstance(referer, str) else None,
            filename=resolved.filename or (filename if isinstance(filename, str) else None),
            title=resolved.title or (title if isinstance(title, str) else None),
            mimeType=resolved.mimeType or (mimeType if isinstance(mimeType, str) else None),
        )

    @Slot(list)
    def download(self, resolvedUri: list[dict[str, Any]]) -> None:
        for resolved in resolvedUri:
            self._download(ResolvedUrl.fromDict(resolved))

    @Slot(str, result=list)
    def speedHistory(self, gid: str) -> list[dict[str, int]]:
        return [sample.asDict() for sample in self._store.speedHistory(gid)]

    @Slot('qint64', result=str)  # type: ignore
    def formatSize(self, bytes_: int) -> str:
        """Human-readable size, e.g. 1536 -> "1.5 KB"."""
        if not bytes_:
            return "—"
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

        from PySide6.QtWidgets import QFileDialog
        return QFileDialog.getExistingDirectory(
            None, "Select destination", start_dir or str(self._settings.downloadDir)
        )

    @Slot(str)
    def pause(self, gid: str) -> None:
        d = self._downloader.pause(gid)
        self._downloader.unlisten(gid)
        self._notify(d)

    @Slot(str)
    def resume(self, gid: str) -> None:
        d = self._downloader.resume(gid)
        self._notify(d)
        self._downloader.listen(gid, self._notify, self._error)

    @Slot(str)
    def stop(self, gid: str) -> None:
        """Stop the download and retain its removed state until it is purged."""
        d = self._downloader.remove(gid)
        self._downloader.unlisten(gid)
        self._notify(d)

    @Slot(str, bool)
    def delete(self, gid: str, deleteFromDisk: bool = True) -> None:
        self._downloader.purge(gid)
        self._downloader.unlisten(gid)
        if deleteFromDisk:
            self._deleteFromDisk(gid)
        self._store.remove(gid)
        self._remove(gid)
        self.downloadRemoved.emit(gid)

    # --- internals ----------------------------------------------------------

    def _deleteFromDisk(self, gid: str) -> None:
        row = self._row_of(gid)
        if row is not None:
            for file in self._downloads[row].files:
                path = file.path
                if path:
                    try:
                        Path(path).unlink(missing_ok=True)
                    except OSError:
                        pass

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
        download = replace(download, resolved=resolved, category=resolved.category)
        self._store.upsert(download)
        self._downloader.listen(download.gid, self._notified.emit, self._error)
        self._insert(download)
        self._notify(download)
        self.downloadAdded.emit(download.gid)
        return download.gid

    def _replace(self, row: int, download: Download) -> None:
        self._downloads[row] = download
        index = self.index(row)
        self.dataChanged.emit(index, index, list(_ROLE_NAMES))

    def _insert(self, download: Download) -> None:
        self.beginInsertRows(QModelIndex(), 0, 0)
        self._downloads.insert(0, download)
        self.endInsertRows()
        self.countsChanged.emit()
        self.activeCountChanged.emit(self.activeCount)

    def _remove(self, gid: str) -> None:
        row = self._row_of(gid)
        if row is None:
            return
        self.beginRemoveRows(QModelIndex(), row, row)
        del self._downloads[row]
        self.endRemoveRows()
        self.countsChanged.emit()
        self.activeCountChanged.emit(self.activeCount)

    def _notify(self, download: Download) -> None:
        row = self._row_of(download.gid)
        if row is not None:
            previous = self._downloads[row]
            download = replace(
                download,
                resolved=download.resolved or previous.resolved,
                category=download.category or previous.category,
                downloadSpeed=(
                    previous.downloadSpeed
                    if download.status != "active"
                    else download.downloadSpeed or previous.downloadSpeed
                ),
                completedLength=download.completedLength or previous.completedLength,
                totalLength=download.totalLength or previous.totalLength,
            )

            self._store.upsert(download)
            if download.status == "active" and download.downloadSpeed:
                self._store.addSpeedSample(download.gid, int(time.time() * 1000), download.downloadSpeed or 0)

            self._replace(row, download)
            if download.category != previous.category:
                self.countsChanged.emit()
            if previous.status != download.status:
                self.activeCountChanged.emit(self.activeCount)
                if download.status == "complete":
                    self.downloadCompleted.emit(download.gid)
                elif download.status == "error":
                    self.downloadFailed.emit(download.gid, download.errorMessage or "")
        self.downloadChanged.emit(download.gid)

    def _error(self, message: str) -> None:
        self.errorOccurred.emit(message)

    def _poll(self) -> None:
        self._pool.submit(self._downloader.refresh)


class DownloadFilterProxy(QSortFilterProxyModel):
    """Filters the download list by category; empty category shows everything."""

    def __init__(self, parent: QObject | None = None) -> None:
        super().__init__(parent)
        self._category = ""
        self._search = ""

    @Slot(str)
    def setCategory(self, category: str) -> None:
        if category == self._category:
            return
        self._category = category
        self.invalidateRowsFilter()

    @Slot(str)
    def setSearch(self, search: str) -> None:
        search = search.strip().lower()
        if search == self._search:
            return
        self._search = search
        self.invalidateRowsFilter()

    def filterAcceptsRow(self, source_row: int, source_parent: Union[QModelIndex, QPersistentModelIndex]) -> bool:
        model = self.sourceModel()
        index = model.index(source_row, 0, source_parent)

        if self._category:
            if model.data(index, _ROLE_BY_NAME["category"]) != self._category:
                return False

        if self._search:
            resolved = model.data(index, _ROLE_BY_NAME["resolved"])
            if resolved:
                title = (resolved.get("title") or resolved.get("filename") or "").lower()
                if self._search in title:
                    return True

            files = model.data(index, _ROLE_BY_NAME["files"])
            if files:
                name = (files[0].get("path") or "").rsplit("/", 1)[-1].lower()
                if self._search in name:
                    return True

            return False

        return True
