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


@dataclass(frozen=True)
class ResolvedUrl:
    title: str
    url: str
    kind: str

    def as_dict(self) -> dict[str, str]:
        return {"title": self.title, "url": self.url, "kind": self.kind}