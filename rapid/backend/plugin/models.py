from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class PluginSpec:
    name: str
    command: str
    args: tuple[str, ...] = ()
    schemes: tuple[str, ...] = ()  # array of regex uri strings
