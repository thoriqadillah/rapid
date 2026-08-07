from __future__ import annotations

import time
from typing import Any

from PySide6.QtCore import QObject, QProcess

from . import protocol


class PluginError(Exception):
    """Raised when a resolver plugin cannot run or answers invalidly."""


class ResolverProcess(QObject):
    """Runs one plugin executable per request over line-delimited JSON-RPC.

    The plugin is spawned for each request and terminated afterwards; it is a
    short-lived stdio worker, not a persistent daemon.
    """

    def __init__(
        self,
        *,
        command: str,
        args: list[str],
        timeout_ms: int = 8000,
        start_ms: int = 3000,
    ) -> None:
        super().__init__()
        self._command = command
        self._args = args
        self._timeout_ms = timeout_ms
        self._start_ms = start_ms
        self._rid = 0
        self._proc: QProcess | None = None
        self._buf = bytearray()

    def call(self, method: str, params: list[Any] | None = None) -> Any:
        proc = QProcess(self)
        proc.setProcessChannelMode(QProcess.ProcessChannelMode.SeparateChannels)
        proc.start(self._command, self._args)
        if not proc.waitForStarted(self._start_ms):
            proc.deleteLater()
            raise PluginError(f"plugin did not start: {self._command}")

        self._proc = proc
        self._buf = bytearray()
        rid = self._next_id()

        try:
            proc.write(protocol.encode_request(rid, method, params or []).encode("utf-8"))
            if not proc.waitForBytesWritten(2000):
                raise PluginError("plugin stdout write timeout")

            line = self._read_line()
            if line is None:
                raise PluginError("plugin returned nothing")

            return protocol.decode_response(line, rid)
        finally:
            self._teardown()

    def _read_line(self, timeout: int | None = None) -> str | None:
        deadline = time.monotonic() + (timeout or self._timeout_ms) / 1000
        proc = self._proc
        if proc is None:
            return None

        while time.monotonic() < deadline:
            if proc.waitForReadyRead(50):
                self._buf += bytes(proc.readAllStandardOutput())
                if b"\n" in self._buf:
                    raw, rest = self._buf.split(b"\n", 1)
                    self._buf = bytearray(rest)
                    return raw.decode("utf-8", "replace")
            elif proc.state() == QProcess.ProcessState.NotRunning:
                break

        return None

    def _next_id(self) -> int:
        self._rid += 1
        return self._rid

    def _teardown(self) -> None:
        proc = self._proc
        self._proc = None
        if proc is None:
            return

        proc.terminate()
        if not proc.waitForFinished(2000):
            proc.kill()
            proc.waitForFinished(2000)
        proc.deleteLater()
