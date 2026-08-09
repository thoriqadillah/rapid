from __future__ import annotations

import time
from pathlib import Path

from PySide6.QtCore import QObject, QTimer, Signal, Slot

from ..plugin import PluginManager
from .downloader import Downloader
from .models import Download, ResolvedUrl
from .store import DownloadStore


class DownloadService(QObject):
    """Application-facing download service.

    It coordinates:
      - Downloader
      - DownloadStore
      - PluginManager
      - Qt signals
    """
