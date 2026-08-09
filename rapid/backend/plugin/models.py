from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class PluginSpec:
    name: str
    command: str
    args: tuple[str, ...] = ()
    schemes: tuple[str, ...] = ()

    def may_match(self, url: str) -> bool:
        lowered = url.lower()
        return any(lowered.startswith(s.lower()) for s in self.schemes)
