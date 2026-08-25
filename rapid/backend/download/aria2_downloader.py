from __future__ import annotations
from pprint import pprint
import re
import mimetypes
from urllib.parse import urlparse, unquote
import subprocess

import base64
import json
import logging
import shutil
import time
import websocket
from dataclasses import replace
from pathlib import Path
from threading import Lock, Thread
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

from .downloader import (
    Downloader,
    Resolver,
    ErrorCallback,
    GlobalNotifyCallback,
    NotifyCallback,
)
from .models import Download, ResolvedUrl
from ..setting.models import Settings

LOG = logging.getLogger(__name__)

JSONRPC_VERSION = "2.0"

STATUS_KEYS = [
    "completedLength",
    "connections",
    "dir",
    "downloadSpeed",
    "errorCode",
    "errorMessage",
    "files",
    "gid",
    "numPieces",
    "pieceLength",
    "status",
    "totalLength",
    "verifiedLength",
]

WS_EVENTS = {
    "aria2.onDownloadComplete",
    "aria2.onDownloadError",
}

_VALID_SCHEMES = {"http", "https", "ftp", "ftps"}


class Aria2Error(RuntimeError):
    pass


class Aria2Rpc:
    def __init__(
        self,
        *,
        host: str,
        port: int,
        token: str = "",
        timeout: float = 10.0,
    ) -> None:
        import json

        self._url = f"http://{host}:{port}/jsonrpc"
        self._token = token
        self._timeout = timeout
        self._next_id = 0
        self._lock = Lock()
        self._json = json

    def call(self, method: str, params: list[Any]) -> Any:
        from urllib.request import Request
        from urllib.request import urlopen
        from urllib.error import HTTPError
        from urllib.error import URLError

        with self._lock:
            self._next_id += 1
            request_id = self._next_id

        rpcParams = list(params)
        if self._token:
            rpcParams.insert(0, f"token:{self._token}")

        payload = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": method,
            "params": rpcParams,
        }

        request = Request(
            self._url,
            method="POST",
            data=self._json.dumps(payload).encode("utf-8"),
            headers={
                "Content-Type": "application/json",
            },
        )

        try:
            with urlopen(request, timeout=self._timeout) as response:
                data = response.read()
        except HTTPError as exc:
            raise Aria2Error(
                f"aria2 HTTP error "
                f"{exc.code}: {exc.reason}"
            ) from exc

        except URLError as exc:
            raise Aria2Error(
                f"cannot connect to aria2: "
                f"{exc.reason}"
            ) from exc

        try:
            message = self._json.loads(data)
        except Exception as exc:
            raise Aria2Error("aria2 returned invalid JSON") from exc

        if not isinstance(message, dict):
            raise Aria2Error("invalid JSON-RPC response")

        error = message.get("error")
        if isinstance(error, dict):
            raise Aria2Error(str(error.get("message", error)))

        return message.get("result")


class Aria2Downloader(Downloader, Resolver):
    """
    Downloader implementation backed by a local aria2c daemon.
    """

    def __init__(
        self,
        *,
        settings: Settings,
        manageDaemon: bool = True,
        onGlobalNotify: GlobalNotifyCallback | None = None,
    ) -> None:
        self._settings = settings
        self._host = settings.aria2Host
        self._port = settings.aria2Port
        self._token = settings.aria2Token
        self._downloadDir = settings.downloadDir
        self._manageDaemon = manageDaemon
        self._running = False

        self._process: subprocess.Popen[bytes] | None = None
        self._listening: set[tuple[str, NotifyCallback, ErrorCallback]] = set()
        self._globalNotifyCallback = onGlobalNotify

        self._ws: websocket.WebSocket | None = None
        self._lock = Lock()
        self._lastSpawn = 0.0
        self._ownsDaemon = False

        self._rpc = Aria2Rpc(
            host=settings.aria2Host,
            port=settings.aria2Port,
            token=settings.aria2Token,
        )


    def _onResolved(self, gid: str) -> None:
        # Dry-run downloads may still be active when the resolve
        # poll deadline expires; removeDownloadResult only accepts
        # finished downloads (active ones fail with HTTP 400), so
        # halt first.
        try:
            self._rpc.call("aria2.remove", [gid])
        except Aria2Error:
            pass

        # Halting is async; retry briefly in case the result is
        # not yet visible to the RPC thread.
        for _ in range(5):
            try:
                self._rpc.call(
                    "aria2.removeDownloadResult",
                    [gid],
                )
                return
            except Aria2Error:
                time.sleep(0.1)

        LOG.warning("failed to remove resolve GID %s", gid)

    @staticmethod
    def _parseInt(value: Any) -> int | None:
        if value is None:
            return None

        try:
            return int(value)
        except (
            TypeError,
            ValueError,
        ):
            return None

    @staticmethod
    def _getFinalUrl(fileInfo: dict[str, Any], fallback: str) -> str:
        uris = fileInfo.get("uris", [])
        if not uris:
            return fallback

        # aria2's URI list tells us which URI
        # was actually used.
        for item in uris:
            if item.get("status") == "used":
                return str(item.get("uri", fallback))

        return str(uris[0].get("uri", fallback))

    @staticmethod
    def _probeHeader(url: str, options: dict[str, Any] = {}) -> dict[str, str] | None:
        try:
            with urlopen(Request(url, method="HEAD", headers=options.get("headers", {})), timeout=5) as resp:
                pprint(resp)
                pprint(resp.headers)
                return resp.headers
        except (HTTPError, URLError, OSError):
            # The HEAD probe is best-effort metadata; a failure just means
            # we fall back to aria2's file info and MIME guessing.
            return None

    @staticmethod
    def _getCategory(mimeType: str | None, fileName: str | None) -> str:
        if fileName:
            guessMimeType = mimetypes.guess_type(fileName, strict=False)[0]

        if not mimeType or not guessMimeType:
            return "unknown"

        if mimeType.startswith("video/") or guessMimeType.startswith("video/"):
            return "video"

        if mimeType.startswith("audio/") or guessMimeType.startswith("audio/"):
            return "audio"

        if mimeType.startswith("image/") or guessMimeType.startswith("image/"):
            return "image"

        if mimeType.startswith("text/") or guessMimeType.startswith("text/") or (mimeType in {
            "application/pdf",
            "application/json",
            "application/xml",
        }) or guessMimeType in {
            "application/pdf",
            "application/json",
            "application/xml",
        }:
            return "document"

        if mimeType in {
            "application/zip",
            "application/x-rar-compressed",
            "application/x-7z-compressed",
            "application/gzip",
            "application/x-tar",
        } or guessMimeType in {
            "application/zip",
            "application/x-rar-compressed",
            "application/x-7z-compressed",
            "application/gzip",
            "application/x-tar",
        }:
            return "compressed"

        return "application"

    @staticmethod
    def _getFilename(uri: str) -> str | None:
        path = urlparse(uri).path
        if not path:
            return None

        filename = Path(unquote(path))
        return filename.name or None

    def _spawnDaemon(self) -> None:
        program = shutil.which("aria2c")
        if not program:
            LOG.error("aria2c binary not found")
            return
        if self._process is not None and self._process.poll() is None:
            LOG.warning("aria2 already running")
            return

        # If a daemon is already serving our RPC port (e.g. an orphan left
        # by a previous crashed run), adopt it instead of spawning a second
        # one that fails to bind and immediately exits rc=1.
        try:
            self._rpc.call("aria2.getVersion", [])
        except Aria2Error:
            pass
        else:
            self._ownsDaemon = False
            LOG.info("adopting existing aria2 on port %s", self._port)
            return

        args = [
            "--enable-rpc",
            f"--rpc-listen-port={self._port}",
            f"--dir={self._downloadDir}",
            f"--save-session={self._settings.aria2SessionFile}",
            f"--input-file={self._settings.aria2SessionFile}",
        ]

        if self._token:
            args.append(f"--rpc-secret={self._token}")

        import subprocess

        self._process = subprocess.Popen(
            [program, *args],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )

        deadline = time.time() + 5.0
        while time.time() < deadline:
            try:
                self._rpc.call("aria2.getVersion", [])
                break
            except Aria2Error:
                if self._process.poll() is not None:
                    LOG.error("aria2 exited during startup: %s", " ".join(args),)
                    self._process = None
                    return
                time.sleep(0.1)
        else:
            LOG.error("aria2 did not become ready on port %s", self._port,)
            self._process = None
            return

        self._ownsDaemon = True
        LOG.info("spawned aria2: %s %s", program, " ".join(args),)

    def start(self) -> None:
        if self._running:
            return

        if self._manageDaemon:
            self._spawnDaemon()

        self._running = True
        Thread(
            target=self._wsRun,
            name="aria2-websocket",
            daemon=True,
        ).start()

    def stop(self) -> None:
        self._running = False
        with self._lock:
            self._listening.clear()

        ws = self._ws
        self._ws = None
        if ws is not None:
            try:
                ws.close()
            except Exception:
                pass

        process = self._process
        if process is None or not self._ownsDaemon:
            return

        self._process = None
        try:
            process.terminate()
            process.wait(timeout=2)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=2)
        except Exception:
            LOG.exception(
                "failed to stop aria2",
            )

    def _ensureDaemon(self) -> None:
        """Re-spawn the daemon if it is missing or died.

        aria2 (or the app) can crash at any point; without this the
        downloader would retry a dead socket forever. Throttled so a
        missing binary only retries every few seconds.
        """
        if not self._manageDaemon:
            return
        if not self._ownsDaemon:
            return
        if self._process is not None and self._process.poll() is None:
            return
        now = time.monotonic()
        if now - self._lastSpawn < 5.0:
            return
        self._lastSpawn = now
        self._spawnDaemon()

    def _wsUrl(self) -> str:
        url = f"ws://{self._host}:{self._port}/jsonrpc"
        if self._token:
            url += f"?token={self._token}"
        return url

    def _wsRun(self) -> None:
        while self._running:
            try:
                ws = websocket.create_connection(self._wsUrl(), timeout=5)
            except Exception:
                self._ensureDaemon()
                LOG.warning("websocket connect failed, retrying")
                time.sleep(1)
                continue

            self._ws = ws
            try:
                while self._running:
                    try:
                        message = ws.recv()
                    except websocket.WebSocketTimeoutException:
                        continue
                    if message:
                        self._onWsMessage(message)
            except Exception:
                if self._running:
                    LOG.warning("websocket error", exc_info=True)
            finally:
                try:
                    ws.close()
                except Exception:
                    pass
                self._ws = None
            time.sleep(1)

    def _onWsMessage(self, message: str) -> None:
        try:
            msg = json.loads(message)
        except (ValueError, TypeError):
            return

        if not isinstance(msg, dict):
            return

        # aria2 pushes lifecycle notifications over WebSocket; each one
        # means the download's state just changed, so push a fresh status
        # to its listeners. Progress still requires polling (no such event).
        if msg.get("method") not in WS_EVENTS:
            return

        gid: str | None = None
        params = msg.get("params")
        if isinstance(params, list) and params:
            item = params[-1]
            if isinstance(item, dict):
                gid = item.get("gid")
        self.refresh(gid if isinstance(gid, str) else None)

    def shouldResolve(self, uri: str) -> bool:
        return any(re.search(pattern, uri) for pattern in _VALID_SCHEMES)

    def resolve(self, uri: str, options: dict[str, Any] = {}) -> list[ResolvedUrl]:
        options = {
            **options,
            "dry-run": "true",
            "use-head": "true",
        }

        gid = self._rpc.call(
            "aria2.addUri",
            [
                [uri],
                options,
            ],
        )

        headers = self._probeHeader(uri, options)

        try:
            fields = [
                "gid",
                "status",
                "totalLength",
                "completedLength",
                "files",
            ]
            status = self._rpc.call(
                "aria2.tellStatus",
                [
                    gid,
                    fields,
                ],
            )

            # Dry-run resolution is asynchronous: aria2 fills in length
            # only after the HEAD request completes. Poll briefly for it.
            deadline = time.time() + 2.0
            while (
                status.get("status") not in ("complete", "error")
                and time.time() < deadline
            ):
                time.sleep(0.1)
                status = self._rpc.call(
                    "aria2.tellStatus",
                    [
                        gid,
                        fields,
                    ],
                )

            files = status.get("files", [])
            file_info = (
                files[0]
                if files
                else {}
            )

            path = file_info.get("path")
            filename = (
                Path(path).name
                if path
                else self._getFilename(uri)
            )

            fileSize = self._parseInt(file_info.get("length", 0))

            finalUrl = self._getFinalUrl(
                file_info,
                uri,
            )

            mimeType = headers.get("Content-Type") if headers else mimetypes.guess_type(filename or "", strict=False)[0]
            size = self._parseInt(headers.get("Content-Length", 0) if headers else fileSize)

            if mimeType and filename:
                suffix = Path(filename).suffix
                if not (suffix and suffix[1:].isalnum()):
                    filename += mimetypes.guess_extension(mimeType) or ""

            return [ResolvedUrl(
                url=finalUrl,
                title=filename,
                filename=filename,
                mimeType=mimeType,
                size=size,
                category=self._getCategory(mimeType, filename),
                headers=options.get("headers", {}),
                cookies=options.get("cookies", {}),
                resolverName="Rapid",
            )]

        finally:
            self._onResolved(gid)

    def getStatus(self, id: str) -> Download:
        result = self._rpc.call(
            "aria2.tellStatus",
            [
                id,
                STATUS_KEYS,
            ],
        )

        if not isinstance(result, dict):
            raise Aria2Error(
                f"aria2 returned no status "
                f"for {id}",
            )

        return Download.fromPayload(result)

    def _aria2Options(self, resolved: ResolvedUrl) -> dict[str, Any]:
        options: dict[str, Any] = {}
        headers = dict(resolved.headers or {})

        if resolved.filename:
            options["out"] = resolved.filename

        if resolved.referer:
            headers["Referer"] = resolved.referer

        if headers:
            options["header"] = [
                f"{key}: {value}"
                for key, value
                in headers.items()
            ]

        cookies = resolved.cookies or {}
        if cookies:
            options["header"] = [
                *options.get("header", []),
                "Cookie: "
                + "; ".join(
                    f"{key}={value}"
                    for key, value
                    in cookies.items()
                ),
            ]

        return options

    def download(self, uri: ResolvedUrl) -> Download:
        options = self._aria2Options(uri)
        result = self._rpc.call(
            "aria2.addUri",
            [
                [uri.url],
                options,
            ],
        )

        gid = str(result) if result else ""
        if not gid:
            raise Aria2Error("aria2 did not return a GID")

        return self.getStatus(gid)

    def pause(self, id: str) -> Download:
        self._rpc.call(
            "aria2.pause",
            [id],
        )

        return self.getStatus(id)

    def resume(self, id: str) -> Download:
        self._rpc.call(
            "aria2.unpause",
            [id],
        )

        return self.getStatus(id)

    def remove(self, id: str) -> Download:
        self._removeAny(id, ("aria2.remove", "aria2.removeDownloadResult"))
        return self.getStatus(id)

    def purge(self, id: str) -> None:
        self._removeAny(id, ("aria2.removeDownloadResult", "aria2.remove"))

    def _removeAny(self, id: str, methods: tuple[str, ...]) -> None:
        # The gid may be active, finished, or already gone from aria2
        # entirely; each RPC only accepts one of those states.
        for method in methods:
            try:
                self._rpc.call(method, [id])
                return
            except Aria2Error:
                continue

    def listen(self, id: str, onNotify: NotifyCallback, onError: ErrorCallback) -> None:
        with self._lock:
            self._listening.add((id, onNotify, onError))

    def unlisten(self, id: str) -> None:
        with self._lock:
            self._listening = {entry for entry in self._listening if entry[0] != id}

    def _notify(self, gid: str, onNotify: NotifyCallback, onError: ErrorCallback) -> None:
        try:
            status = self.getStatus(gid)
        except Aria2Error as exc:
            LOG.error(str(exc))
            onError(str(exc))
            self.unlisten(gid)
            return
        onNotify(status)
        if status.status in ("error", "removed", "complete"):
            self.unlisten(gid)

    def refresh(self, id: str | None = None) -> None:
        if not self._running:
            return

        with self._lock:
            entries = tuple(self._listening)

        for gid, onNotify, onError in entries:
            if id is not None and gid != id:
                continue
            self._notify(gid, onNotify, onError)

        if id is not None:
            return

        try:
            result = self._rpc.call("aria2.getGlobalStat", [])
            globalCallback = self._globalNotifyCallback
            if (globalCallback is not None and isinstance(result, dict)):
                globalCallback(result)

        except Aria2Error as exc:
            LOG.error(str(exc))
