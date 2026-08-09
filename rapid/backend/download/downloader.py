from __future__ import annotations
from abc import ABC, abstractmethod

import json
import logging
import shutil
import time
from dataclasses import replace
from pathlib import Path
from typing import Any, Callable, Optional

from .models import Download, ResolvedUrl

NotifyCallback = Callable[[Download], None]
GlobalNotifyCallback = Callable[[dict[str, object]], None]
ErrorCallback = Callable[[str], None]


class Downloader(ABC):
    @abstractmethod
    def start(self) -> None:
        """Start the downloader and begin listening for events."""
        pass

    @abstractmethod
    def stop(self) -> None:
        """Stop the downloader."""
        pass

    @abstractmethod
    def download(self, uri: ResolvedUrl) -> Download:
        """Start a download and return its initial state."""
        pass

    @abstractmethod
    def pause(self, id: str) -> None:
        pass

    @abstractmethod
    def resume(self, id: str) -> None:
        pass

    @abstractmethod
    def remove(self, id: str) -> None:
        pass

    @abstractmethod
    def purge(self, id: str) -> None:
        pass

    @abstractmethod
    def getStatus(self, id: str) -> Download:
        pass

    @abstractmethod
    def listen(self, id: str, onNotify: NotifyCallback, onError: ErrorCallback) -> None:
        """Start listening for status changes for a download."""
        pass


class Resolver(ABC):
    @abstractmethod
    def shouldResolve(self, uri: str) -> bool:
        """Return whether the resolver should handle the given URI."""
        pass

    @abstractmethod
    def resolve(self, uri: str) -> list[ResolvedUrl]:
        """Resolve a URI into one or more downloadable resources."""
        pass
