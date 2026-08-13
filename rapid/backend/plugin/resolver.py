from __future__ import annotations

import re
import time
from typing import Any

from PySide6.QtCore import QProcess

from rapid.backend.download.models import ResolvedUrl
from rapid.backend.download.downloader import Resolver

from . import protocol
from .models import PluginSpec


class PluginError(Exception):
    """Raised when a resolver plugin cannot run or answers invalidly."""


class ResolverPlugin(Resolver):
    """Runs one plugin executable per request over line-delimited JSON-RPC.

    The plugin is spawned for each request and terminated afterwards; it is a
    short-lived stdio worker, not a persistent daemon.
    """

    def __init__(
        self,
        *,
        spec: PluginSpec,
        timeout_ms: int = 8000,
        start_ms: int = 3000,
    ) -> None:
        self._spec = spec
        self._timeout_ms = timeout_ms
        self._start_ms = start_ms
        self._rid = 0
        self._proc: QProcess | None = None
        self._buf = bytearray()

    def _call(self, method: str, params: list[Any] | None = None) -> Any:
        proc = QProcess()
        proc.setProcessChannelMode(QProcess.ProcessChannelMode.SeparateChannels)
        proc.start(self._spec.command, self._spec.args)
        if not proc.waitForStarted(self._start_ms):
            raise PluginError(f"plugin did not start: {self._spec.command}")

        self._proc = proc
        self._buf = bytearray()
        rid = self._nextId()

        try:
            proc.write(protocol.encodeRequest(rid, method, params or []).encode("utf-8"))
            if not proc.waitForBytesWritten(2000):
                raise PluginError("plugin stdout write timeout")

            line = self._readLine()
            if line is None:
                raise PluginError("plugin returned nothing")

            return protocol.decodeResponse(line, rid)
        finally:
            self._teardown()

    def _readLine(self, timeout: int | None = None) -> str | None:
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

    def _nextId(self) -> int:
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

    def shouldResolve(self, uri: str) -> bool:
        return any(re.search(pattern, uri) for pattern in self._spec.schemes if pattern)

    def resolve(self, uri: str) -> list[ResolvedUrl]:
        result = self._call(protocol.RESOLVE, [uri])
        if not isinstance(result, dict) or not isinstance(result.get("items"), list):
            raise PluginError("resolve returned invalid result")

        return [ResolvedUrl(**item, resolverName=self._spec.name) for item in result["items"]]
