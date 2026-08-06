from __future__ import annotations

import json
import logging
import shutil
import time
from pathlib import Path
from typing import Any, Callable

from PySide6.QtCore import QObject, QProcess, QTimer, QUrl, Signal, Slot
from PySide6.QtNetwork import QNetworkAccessManager, QNetworkReply, QNetworkRequest
from PySide6.QtWebSockets import QWebSocket, QWebSocketProtocol

from .store import DownloadStore, _to_int

LOG = logging.getLogger(__name__)
JSONRPC_VERSION = "2.0"

STATUS_KEYS = [
    "bittorrent",
    "bitfield",
    "completedLength",
    "connections",
    "dir",
    "downloadSpeed",
    "errorCode",
    "errorMessage",
    "files",
    "gid",
    "infoHash",
    "numPieces",
    "numSeeders",
    "pieceLength",
    "seeder",
    "status",
    "totalLength",
    "uploadLength",
    "uploadSpeed",
    "verifiedLength",
]


class Aria2Client(QObject):
    """QML-facing wrapper around an aria2 JSON-RPC daemon.

    Manages a local ``aria2c`` daemon via QProcess (or connects to an existing
    one) and talks JSON-RPC over HTTP while subscribing to push notifications
    over WebSocket.
    """

    downloadAdded = Signal(str)
    statusChanged = Signal(str, object)
    globalStatChanged = Signal(object)
    errorOccurred = Signal(str)

    def __init__(
        self,
        *,
        store: DownloadStore,
        host: str = "127.0.0.1",
        port: int = 6800,
        token: str = "",
        download_dir: Path | None = None,
        manage_daemon: bool = True,
        parent: QObject | None = None,
    ) -> None:
        super().__init__(parent)
        self._store = store
        self._host = host
        self._port = port
        self._token = token
        self._download_dir = download_dir
        self._manage_daemon = manage_daemon

        self._rpc_url = self._make_url("http", host, port)
        self._ws_url = self._make_url("ws", host, port)

        self._nam = QNetworkAccessManager(self)
        self._next_id = 0
        self._pending: dict[int, Callable[[Any], Any]] = {}
        self._inflight: dict[int, QNetworkReply] = {}

        self._process: QProcess | None = None
        self._ws: QWebSocket | None = None
        self._progress: dict[str, tuple[int, int | None]] = {}

        self._global_timer = QTimer(self)
        self._global_timer.setInterval(1000)
        self._global_timer.timeout.connect(self._tick)

    @staticmethod
    def _make_url(scheme: str, host: str, port: int) -> str:
        return f"{scheme}://{host}:{port}/jsonrpc"

    def jsonrpc_url(self) -> str:
        return self._rpc_url

    def websocket_url(self) -> str:
        return self._ws_url

    @Slot()
    def start(self) -> None:
        if self._manage_daemon:
            self._spawn_daemon()
        self._connect_websocket()
        self._global_timer.start()
        for gid in self._store.all():
            self._tell_status(gid)

    @Slot()
    def stop(self) -> None:
        self._global_timer.stop()
        if self._ws is not None:
            self._ws.close()
            self._ws = None
        process = self._process
        if process is not None:
            self._process = None
            process.terminate()
            if not process.waitForFinished(2000):
                process.kill()
                process.waitForFinished(2000)
            process.deleteLater()

    def _spawn_daemon(self) -> None:
        program = shutil.which("aria2c")
        if not program:
            self.errorOccurred.emit("aria2c binary not found")
            return
        args: list[str] = ["--enable-rpc", f"--rpc-listen-port={self._port}"]
        if self._token:
            args.append(f"--rpc-secret={self._token}")
        if self._download_dir is not None and str(self._download_dir):
            args.append(f"--dir={self._download_dir}")

        process = QProcess(self)
        process.setProgram(program)
        process.setArguments(args)
        process.errorOccurred.connect(
            lambda err: self.errorOccurred.emit(f"aria2 error: {err}")
        )
        process.finished.connect(
            lambda *_args: self.errorOccurred.emit("aria2 daemon exited")
        )
        process.start()
        self._process = process
        LOG.info("spawned %s %s", program, " ".join(args))

    def _connect_websocket(self) -> None:
        ws = QWebSocket("", QWebSocketProtocol.Version.VersionLatest, self)
        ws.textMessageReceived.connect(self._on_ws_message)
        ws.errorOccurred.connect(lambda err: self.errorOccurred.emit(f"ws error: {err}"))
        ws.open(QUrl(self._ws_url))
        self._ws = ws

    def _on_ws_message(self, message: str) -> None:
        try:
            msg = json.loads(message)
        except ValueError:
            return
        method = msg.get("method")
        params = msg.get("params")
        if not isinstance(method, str):
            return
        payload: dict[str, Any] = {}
        if isinstance(params, list) and params:
            item = params[-1]
            if isinstance(item, dict):
                payload = item
        self._handle_notification(method, payload)

    def _handle_notification(self, method: str, payload: dict[str, Any]) -> None:
        gid = payload.get("gid")
        if gid is None:
            return
        gid_str = str(gid)
        if method == "aria2.onDownloadStart":
            self._tell_status(gid_str)
        elif method in (
            "aria2.onDownloadComplete",
            "aria2.onDownloadPause",
            "aria2.onDownloadStop",
            "aria2.onDownloadError",
            "bt-download-complete",
        ):
            self._tell_status(gid_str)

    # --- JSON-RPC plumbing -------------------------------------------------

    def _call(
        self,
        method: str,
        params: list[Any],
        on_result: Callable[[Any], Any] | None = None,
    ) -> int:
        self._next_id += 1
        rid = self._next_id
        args: list[Any] = list(params)
        if self._token:
            args.insert(0, f"token:{self._token}")
        payload: dict[str, Any] = {
            "jsonrpc": JSONRPC_VERSION,
            "id": rid,
            "method": method,
            "params": args,
        }
        if on_result is not None:
            self._pending[rid] = on_result
        self._post(rid, payload)
        return rid

    def _post(self, rid: int, payload: dict[str, Any]) -> None:
        body = json.dumps(payload).encode("utf-8")
        request = QNetworkRequest(QUrl(self._rpc_url))
        request.setHeader(
            QNetworkRequest.KnownHeaders.ContentTypeHeader, "application/json"
        )
        reply = self._nam.post(request, body)
        self._inflight[rid] = reply
        reply.finished.connect(lambda _r=rid: self._on_reply(_r, reply))

    def _on_reply(self, rid: int, reply: QNetworkReply) -> None:
        self._inflight.pop(rid, None)
        reply.deleteLater()
        callback = self._pending.pop(rid, None)

        if reply.error() != QNetworkReply.NetworkError.NoError:
            self.errorOccurred.emit(reply.errorString())
            if callback:
                callback(None)
            return

        data = bytes(reply.readAll())
        try:
            msg = json.loads(data)
        except ValueError:
            if callback:
                callback(None)
            return

        if not isinstance(msg, dict):
            if callback:
                callback(None)
            return
        err = msg.get("error")
        if isinstance(err, dict):
            self.errorOccurred.emit(str(err.get("message", err)))
            if callback:
                callback(None)
            return
        if callback:
            callback(msg.get("result"))

    # --- High-level RPC (QML slots) ---------------------------------------

    @Slot(str)
    def add_uri(self, uri: str) -> None:
        self._call(
            "aria2.addUri",
            [[uri]],
            lambda gid: self.downloadAdded.emit(self._gid_like(gid)),
        )

    @Slot(str)
    def add_torrent(self, file_path: str) -> None:
        self._call(
            "aria2.addTorrent",
            [file_path],
            lambda gid: self.downloadAdded.emit(self._gid_like(gid)),
        )

    @Slot(str)
    def pause(self, gid: str) -> None:
        self._call("aria2.pause", [gid])

    @Slot(str)
    def unpause(self, gid: str) -> None:
        self._call("aria2.unpause", [gid])

    @Slot(str)
    def remove(self, gid: str) -> None:
        self._call("aria2.remove", [gid])

    @Slot()
    def purge(self) -> None:
        self._call("aria2.purgeDownloadResult", [])

    @Slot(str)
    def tell_status(self, gid: str) -> None:
        self._tell_status(gid)

    def _tell_status(self, gid: str) -> None:
        def emit(result: Any) -> None:
            if isinstance(result, dict):
                resolved = str(result.get("gid", gid))
                self._store.upsert(resolved, result)
                self._sample(resolved, result)
                self.statusChanged.emit(resolved, result)

        self._call("aria2.tellStatus", [gid, STATUS_KEYS], emit)

    @Slot()
    def refresh_active_and_waiting(self) -> None:
        def cb(result: Any) -> None:
            if isinstance(result, list):
                for item in result:
                    if isinstance(item, dict) and isinstance(item.get("gid"), str):
                        gid = item["gid"]
                        self._store.upsert(gid, item)
                        self._sample(gid, item)
                        self.statusChanged.emit(gid, item)

        self._call("aria2.tellActive", [], cb)
        self._call("aria2.tellWaiting", [0, 1000, STATUS_KEYS], cb)

    @Slot()
    def refresh_global(self) -> None:
        self._call(
            "aria2.getGlobalStat",
            [],
            lambda result: self.globalStatChanged.emit(result)
            if isinstance(result, dict)
            else None,
        )

    @Slot()
    def _tick(self) -> None:
        self.refresh_active_and_waiting()
        self.refresh_global()
        cutoff = int(time.time() * 1000) - 3600_000
        self._store.prune_speeds(cutoff)

    @Slot(str, result=list)
    def speed_history(self, gid: str) -> list[dict[str, int]]:
        return self._store.speed_history(gid, limit=60)

    def _sample(self, gid: str, payload: dict[str, Any]) -> None:
        completed = _to_int(payload.get("completedLength"))
        now = int(time.time() * 1000)
        previous = self._progress.get(gid)
        self._progress[gid] = (now, completed)

        speed = _to_int(payload.get("downloadSpeed"))
        if (
            completed is not None
            and previous is not None
            and previous[1] is not None
            and now > previous[0]
        ):
            delta_ms = now - previous[0]
            delta_bytes = completed - previous[1]
            if delta_bytes >= 0 and delta_ms > 0:
                speed = (delta_bytes * 1000) // delta_ms

        if speed is not None:
            self._store.add_speed_sample(gid, now, speed)

    @staticmethod
    def _gid_like(value: Any) -> str:
        if isinstance(value, dict):
            gotten = value.get("gid")
            return str(gotten) if gotten is not None else ""
        return str(value)