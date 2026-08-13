from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from PySide6.QtCore import QObject, Slot

from rapid.backend.download.models import ResolvedUrl

from . import protocol
from .models import PluginSpec
from .resolver import PluginError, ResolverPlugin

MANIFEST_NAME = "plugin.json"


class PluginManager(QObject):
    """Discovers resolver plugins and runs per-request resolutions.

    Plugins are short-lived executables (``plugin.json`` in each plugin dir)
    that answer ``rapid.*`` line-JSON-RPC over stdio. Discovery is cheap
    (manifest parsing only); executables spawn only when a URL is resolved.
    """

    def __init__(self, dirs: list[Path], parent: QObject | None = None) -> None:
        super().__init__(parent)
        self._plugins: list[PluginSpec] = []
        for base in dirs:
            self._scan(base)

    def _scan(self, base: Path) -> None:
        if not base.is_dir():
            return
        for manifest in base.glob(f"*/{MANIFEST_NAME}"):
            spec = self._loadSpec(manifest)
            if spec is not None:
                self._plugins.append(spec)

    @staticmethod
    def _loadSpec(manifest: Path) -> PluginSpec | None:
        try:
            data = json.loads(manifest.read_text(encoding="utf-8"))
        except (OSError, ValueError):
            return None

        if not isinstance(data, dict) or data.get("type") != "resolver":
            return None

        name = data.get("name")
        command = data.get("command")
        if not isinstance(name, str) or not isinstance(command, str):
            return None

        args = data.get("args")
        schemes = data.get("schemes")
        command_path = Path(command)
        if not command_path.is_absolute():
            command_path = manifest.parent / command_path

        return PluginSpec(
            name=name,
            command=command_path.as_posix(),
            args=tuple(a for a in args if isinstance(a, str)) if isinstance(args, list) else (),
            schemes=tuple(s for s in schemes if isinstance(s, str)) if isinstance(schemes, list) else (),
        )

    @property
    def resolverNames(self) -> list[str]:
        return [r.name for r in self._plugins]

    def resolve(self, uri: str) -> list[ResolvedUrl]:
        results: list[ResolvedUrl] = []
        for plugin in self._plugins:
            resolver = ResolverPlugin(spec=plugin)
            if resolver.shouldResolve(uri):
                results.extend(resolver.resolve(uri))

        return results
