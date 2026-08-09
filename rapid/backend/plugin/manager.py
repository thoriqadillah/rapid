from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from PySide6.QtCore import QObject, Slot

from . import protocol
from .models import PluginSpec
from .process import PluginError, ResolverProcess

MANIFEST_NAME = "plugin.json"


class PluginManager(QObject):
    """Discovers resolver plugins and runs per-request resolutions.

    Plugins are short-lived executables (``plugin.json`` in each plugin dir)
    that answer ``rapid.*`` line-JSON-RPC over stdio. Discovery is cheap
    (manifest parsing only); executables spawn only when a URL is resolved.
    """

    def __init__(self, dirs: list[Path], parent: QObject | None = None) -> None:
        super().__init__(parent)
        self._resolvers: list[PluginSpec] = []
        for base in dirs:
            self._scan(base)

    def _scan(self, base: Path) -> None:
        if not base.is_dir():
            return
        for manifest in base.glob(f"*/{MANIFEST_NAME}"):
            spec = self._load_spec(manifest)
            if spec is not None:
                self._resolvers.append(spec)

    @staticmethod
    def _load_spec(manifest: Path) -> PluginSpec | None:
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
    def resolver_names(self) -> list[str]:
        return [r.name for r in self._resolvers]

    @Slot(str, result=list)
    def available_for(self, url: str) -> list[str]:
        """Names of resolvers that report they can handle ``url``."""
        matching: list[str] = []
        for spec in self._resolvers:
            if spec.schemes and not spec.may_match(url):
                continue
            try:
                result = ResolverProcess(command=spec.command, args=list(spec.args)).call(
                    protocol.MATCH, [url]
                )
            except PluginError:
                continue

            if isinstance(result, dict) and result.get("supported") is True:
                matching.append(spec.name)

        return matching

    @Slot(str, str, result=list)
    def resolve(self, url: str, plugin_name: str) -> list[dict[str, str]]:
        spec = next((r for r in self._resolvers if r.name == plugin_name), None)
        if spec is None:
            return []

        proc = ResolverProcess(command=spec.command, args=list(spec.args))
        if not self._is_supported(proc, url):
            raise PluginError(f"{plugin_name} does not support {url}")

        result = proc.call(protocol.RESOLVE, [url])
        if not isinstance(result, dict):
            return []

        items = result.get("items")
        if not isinstance(items, list):
            return []

        resolved: list[dict[str, str]] = []
        for item in items:
            if isinstance(item, dict) and isinstance(item.get("title"), str) and isinstance(item.get("url"), str):
                resolved.append(
                    {"title": item["title"], "url": item["url"], "kind": str(item.get("kind", "other"))}
                )

        return resolved

    @staticmethod
    def _is_supported(proc: ResolverProcess, url: str) -> bool:
        result = proc.call(protocol.MATCH, [url])
        return isinstance(result, dict) and result.get("supported") is True
