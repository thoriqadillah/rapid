from __future__ import annotations
from rapid.backend.plugin.transport import Transport, ProcessTransport, TransportError
from typing import TypeIs

from dataclasses import dataclass

@dataclass(frozen=True)
class StdIOTransportSpec:
    command: str
    args: tuple[str, ...] = ()

@dataclass(frozen=True)
class BuiltInTransportSpec:
    pass

type TransportSpec = StdIOTransportSpec | BuiltInTransportSpec

@dataclass(frozen=True)
class PluginSpec:
    id: str
    version: str
    name: str
    transport: TransportSpec
    match: tuple[str, ...] = ()  # array of regex uri strings

    def createTransporter(self) -> Transport:
        if isinstance(self.transport, StdIOTransportSpec):
            return ProcessTransport(
                command=self.transport.command,
                args=self.transport.args,
            )

        raise TransportError("Unsupported transport spec")
