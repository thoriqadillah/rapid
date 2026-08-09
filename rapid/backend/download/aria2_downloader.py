from __future__ import annotations
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
    "uploadLength",
    "uploadSpeed",
    "verifiedLength",
]

WS_EVENTS = {
    "aria2.onDownloadComplete",
    "aria2.onDownloadError",
}


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
        host: str = "127.0.0.1",
        port: int = 6800,
        token: str = "",
        downloadDir: Path | None = None,
        manageDaemon: bool = True,
        onGlobalNotify: GlobalNotifyCallback | None = None,
    ) -> None:
        self._host = host
        self._port = port
        self._token = token
        self._downloadDir = downloadDir
        self._manageDaemon = manageDaemon
        self._running = False

        self._process: subprocess.Popen[bytes] | None = None
        self._listening: set[tuple[str, NotifyCallback, ErrorCallback]] = set()
        self._globalNotifyCallback = onGlobalNotify

        self._ws: websocket.WebSocket | None = None
        self._lock = Lock()

        self._rpc = Aria2Rpc(
            host=host,
            port=port,
            token=token,
        )


    def _onResolved(self, gid: str) -> None:
        try:
            self._rpc.call(
                "aria2.removeDownloadResult",
                [gid],
            )
        except Exception:
            # Resolution succeeded, cleanup failure
            # shouldn't hide the actual result.
            LOG.warning(
                "failed to remove resolve GID %s",
                gid,
                exc_info=True,
            )

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
    def _getKind(mimeType: str | None) -> str:
        if not mimeType:
            return "unknown"

        if mimeType.startswith("video/"):
            return "video"

        if mimeType.startswith("audio/"):
            return "audio"

        if mimeType.startswith("image/"):
            return "image"

        if mimeType.startswith("text/"):
            return "document"

        if mimeType in {
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

        args = [
            "--enable-rpc",
            f"--rpc-listen-port={self._port}",
        ]

        if self._token:
            args.append(f"--rpc-secret={self._token}")

        if self._downloadDir is not None:
            args.append(f"--dir={self._downloadDir}")

        import subprocess

        self._process = subprocess.Popen(
            [program, *args],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )

        LOG.info(
            "spawned aria2: %s %s",
            program,
            " ".join(args),
        )

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
        if process is None:
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
                LOG.warning("websocket connect failed", exc_info=True)
                time.sleep(1)
                continue

            self._ws = ws
            try:
                while self._running:
                    message = ws.recv()
                    if message:
                        self._onWsMessage(message)
            except Exception:
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
        _ = uri
        return True

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

            size = self._parseInt(
                file_info.get("length", 0)
            )

            finalUrl = self._getFinalUrl(
                file_info,
                uri,
            )

            mimeType = mimetypes.guess_type(filename or "", strict=False)[0]

            return [ResolvedUrl(
                url=finalUrl,
                title=filename,
                filename=filename,
                mimeType=mimeType,
                size=size,
                kind=self._getKind(mimeType),
                headers=options.get("headers", {}),
                cookies=options.get("cookies", {}),
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

    def pause(self, id: str) -> None:
        self._rpc.call(
            "aria2.pause",
            [id],
        )

    def resume(self, id: str) -> None:
        self._rpc.call(
            "aria2.unpause",
            [id],
        )

    def remove(self, id: str) -> None:
        self._rpc.call(
            "aria2.remove",
            [id],
        )

        with self._lock:
            self._listening = {entry for entry in self._listening if entry[0] != id}

    def purge(self, id: str) -> None:
        self._rpc.call(
            "aria2.removeDownloadResult",
            [id],
        )

        with self._lock:
            self._listening = {entry for entry in self._listening if entry[0] != id}

    def listen(self, id: str, onNotify: NotifyCallback, onError: ErrorCallback) -> None:
        with self._lock:
            self._listening.add((id, onNotify, onError))

    def _notify(self, gid: str, onNotify: NotifyCallback, onError: ErrorCallback) -> None:
        try:
            status = self.getStatus(gid)
            onNotify(status)
        except Aria2Error as exc:
            LOG.error(str(exc))
            onError(str(exc))

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
