from __future__ import annotations

import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from threading import Thread
from typing import Any
from urllib.parse import urlparse

from PySide6.QtCore import QObject, Signal

_MAX_BODY_BYTES = 2 * 1024 * 1024
_ALLOWED_ORIGIN_SCHEMES = ("chrome-extension://", "moz-extension://")
_BLOCKED_HEADERS = {"connection", "content-length", "cookie", "host", "proxy-connection"}


class BrowserRequestError(ValueError):
    """Raised when the browser extension sends an unsafe or invalid request."""


def _stringMap(value: Any, *, blocked: set[str] | None = None) -> dict[str, str]:
    if not isinstance(value, dict):
        return {}

    result: dict[str, str] = {}
    for key, item in value.items():
        if not isinstance(key, str) or not isinstance(item, str):
            continue

        key = key.strip()
        if not key or (blocked and key.lower() in blocked):
            continue

        result[key] = item

    return result


def normalizeBrowserRequest(value: Any) -> dict[str, Any]:
    if not isinstance(value, dict):
        raise BrowserRequestError("Request body must be a JSON object")

    url = value.get("url")
    if not isinstance(url, str) or urlparse(url).scheme not in {"http", "https", "ftp", "ftps"}:
        raise BrowserRequestError("A downloadable HTTP, HTTPS, FTP, or FTPS URL is required")

    request: dict[str, Any] = {
        "url": url,
        "headers": _stringMap(value.get("headers"), blocked=_BLOCKED_HEADERS),
        "cookies": _stringMap(value.get("cookies")),
    }
    for key in ("referer", "pageUrl", "title", "filename", "mimeType", "category", "source"):
        item = value.get(key)
        if isinstance(item, str):
            request[key] = item

    size = value.get("size")
    if isinstance(size, int) and not isinstance(size, bool) and size >= 0:
        request["size"] = size
    if value.get("browserResolved") is True:
        request["browserResolved"] = True

    return request


def _originAllowed(origin: str | None) -> bool:
    return origin is None or origin.startswith(_ALLOWED_ORIGIN_SCHEMES)


class _BridgeServer(ThreadingHTTPServer):
    daemon_threads = True
    allow_reuse_address = True

    def __init__(self, address: tuple[str, int], service: BrowserIntegration) -> None:
        self.service = service
        super().__init__(address, _BridgeHandler)


class _BridgeHandler(BaseHTTPRequestHandler):
    server: _BridgeServer

    def _cors(self) -> None:
        origin = self.headers.get("Origin")
        if origin and _originAllowed(origin):
            self.send_header("Access-Control-Allow-Origin", origin)
            self.send_header("Vary", "Origin")
        self.send_header("Access-Control-Allow-Headers", "Content-Type, X-Rapid-Extension")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

    def _reply(self, status: int, payload: dict[str, Any]) -> None:
        body = json.dumps(payload, separators=(",", ":")).encode("utf-8")
        self.send_response(status)
        self._cors()
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _trusted(self) -> bool:
        return (
            _originAllowed(self.headers.get("Origin"))
            and self.headers.get("X-Rapid-Extension") == "1"
        )

    def do_OPTIONS(self) -> None:
        requestedHeaders = self.headers.get("Access-Control-Request-Headers", "").lower()
        if (
            not _originAllowed(self.headers.get("Origin"))
            or "x-rapid-extension" not in requestedHeaders
        ):
            self._reply(403, {"ok": False, "error": "Untrusted extension origin"})
            return
        self.send_response(204)
        self._cors()
        self.end_headers()

    def do_GET(self) -> None:
        if self.path != "/health" or not self._trusted():
            self._reply(404, {"ok": False, "error": "Not found"})
            return
        self._reply(200, {"ok": True, "app": "rapid"})

    def do_POST(self) -> None:
        if self.path != "/downloads" or not self._trusted():
            self._reply(403, {"ok": False, "error": "Untrusted extension request"})
            return

        try:
            length = int(self.headers.get("Content-Length", "0"))
        except ValueError:
            length = 0
        if length <= 0 or length > _MAX_BODY_BYTES:
            self._reply(413, {"ok": False, "error": "Invalid request size"})
            return

        try:
            payload = json.loads(self.rfile.read(length))
            request = normalizeBrowserRequest(payload)
        except (ValueError, json.JSONDecodeError) as exc:
            self._reply(400, {"ok": False, "error": str(exc)})
            return

        self.server.service.downloadRequested.emit(request)
        self._reply(202, {"ok": True})

    def log_message(self, format: str, *args: object) -> None:
        return


class BrowserIntegration(QObject):
    """Local, extension-only bridge that forwards browser requests to QML."""

    downloadRequested = Signal(dict)
    errorOccurred = Signal(str)

    def __init__(
        self,
        host: str = "127.0.0.1",
        port: int = 17654,
        parent: QObject | None = None,
    ) -> None:
        super().__init__(parent)
        self._host = host
        self._port = port
        self._server: _BridgeServer | None = None
        self._thread: Thread | None = None

    @property
    def address(self) -> tuple[str, int]:
        if self._server is not None:
            host, port = self._server.server_address[:2]
            return str(host), int(port)

        return self._host, self._port

    def start(self) -> None:
        if self._server is not None:
            return

        try:
            server = _BridgeServer(self.address, self)
        except OSError as exc:
            self.errorOccurred.emit(f"Browser integration could not start: {exc}")
            return

        self._server = server
        self._thread = Thread(target=server.serve_forever, name="rapid-browser-bridge", daemon=True)
        self._thread.start()

    def close(self) -> None:
        server = self._server
        thread = self._thread
        self._server = None
        self._thread = None
        if server is None:
            return

        server.shutdown()
        server.server_close()
        if thread is not None:
            thread.join(timeout=2)
