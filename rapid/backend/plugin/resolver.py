from __future__ import annotations
from rapid.backend.plugin.transport import TransportError

import re
from typing import Any

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
        self._rid = 0

    def _nextId(self) -> int:
        self._rid += 1
        return self._rid

    def _call(self, method: str, params: list[Any] | None = None) -> Any:
        rid = self._nextId()
        payload = protocol.encodeRequest(rid, method, params or [])

        try:
            transport = self._spec.createTransporter()
            resp = transport.request(payload)
        except TransportError:
            raise

        except Exception as exc:
            raise PluginError(f"Transport error: {exc}")

        try:
            return protocol.decodeResponse(resp, rid)
        except Exception as exc:
            raise PluginError(f"Response error: {exc}")

    def shouldResolve(self, uri: str) -> bool:
        return any(re.search(pattern, uri) for pattern in self._spec.match if pattern)

    def resolve(
        self,
        uri: str,
        options: dict[str, Any] | None = None,
    ) -> list[ResolvedUrl]:
        result = self._call(protocol.RESOLVE, [uri, options or {}])
        if not isinstance(result, dict) or not isinstance(result.get("items"), list):
            raise PluginError("Resolve returned invalid result")

        return [ResolvedUrl(**item, resolverName=self._spec.name) for item in result["items"]]
