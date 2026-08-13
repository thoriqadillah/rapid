from __future__ import annotations
from rapid.backend.plugin.transport import TransportError

import json
from pathlib import Path
from typing import Any

from PySide6.QtCore import QObject, Slot

from rapid.backend.download.models import ResolvedUrl

from . import protocol
from .models import PluginSpec, StdIOTransportSpec, TransportSpec
from .resolver import PluginError, ResolverPlugin

MANIFEST_NAME = "plugin.json"


# create transport spec from manifest data (only stdio is supported for now)
def _createTransportSpec(manifest: Path, data: dict[str, Any]) -> TransportSpec:
    """Build a transport spec from manifest data (only stdio is supported)."""
    transport = data.get("transport")
    if not isinstance(transport, dict):
        raise TransportError("Invalid transport specification")

    args = transport.get("args")
    command = transport.get("command")
    if not isinstance(command, str):
        raise TransportError("Command must be a string")

    command_path = Path(command)
    if not command_path.is_absolute():
        command_path = manifest.parent / command_path

    return StdIOTransportSpec(
        command=command_path.as_posix(),
        args=tuple(a for a in args if isinstance(a, str)) if isinstance(args, list) else (),
    )


class PluginManager(QObject):
    """Registry of resolver plugins, keyed by id.

    Plugins are short-lived executables (``plugin.json`` in each plugin dir)
    that answer ``rapid.*`` line-JSON-RPC over stdio. Discovery is cheap
    (manifest parsing only); executables spawn only when a URL is resolved.
    """

    def __init__(self, dirs: list[Path], parent: QObject | None = None) -> None:
        super().__init__(parent)
        self._plugins: dict[str, PluginSpec] = {}
        for base in dirs:
            self._scan(base)

    def _scan(self, base: Path) -> None:
        if not base.is_dir():
            return
        for manifest in base.glob(f"*/{MANIFEST_NAME}"):
            spec = self._loadSpec(manifest)
            if spec is not None:
                self._plugins[spec.id] = spec

    @staticmethod
    def _loadSpec(manifest: Path) -> PluginSpec | None:
        try:
            data = json.loads(manifest.read_text(encoding="utf-8"))
        except (OSError, ValueError):
            return None

        if not isinstance(data, dict) or data.get("type") != "resolver":
            return None

        name = data.get("name")
        if not isinstance(name, str):
            return None

        id_ = data.get("id")
        version = data.get("version")
        match = data.get("match")

        try:
            transport = _createTransportSpec(manifest, data)
        except TransportError:
            return None

        return PluginSpec(
            id=id_ if isinstance(id_, str) else name,
            name=name,
            version=version if isinstance(version, str) else "0.0.0",
            transport=transport,
            match=tuple(s for s in match if isinstance(s, str)) if isinstance(match, list) else (),
        )

    def get(self, id: str) -> PluginSpec | None:
        return self._plugins.get(id)

    @property
    def resolverNames(self) -> list[str]:
        return [spec.name for spec in self._plugins.values()]

    def resolve(self, uri: str) -> list[ResolvedUrl]:
        results: list[ResolvedUrl] = []
        for plugin in self._plugins.values():
            resolver = ResolverPlugin(spec=plugin)
            if resolver.shouldResolve(uri):
                results.extend(resolver.resolve(uri))

        return results
