from __future__ import annotations

import re
from typing import Callable, cast

from PySide6.QtCore import QObject, Property, Signal, Slot
from PySide6.QtGui import QGuiApplication
from PySide6.QtQml import QQmlContext

_URL_RE = re.compile(r'https?://\S+')


class ClipboardService(QObject):
    """Reads the system clipboard for QML. Text is live-updated via textChanged."""

    textChanged = Signal()
    urlChanged = Signal()

    @Slot(str)
    def copy(self, text: str) -> None:
        QGuiApplication.clipboard().setText(text)

    @Property(str, notify=cast(Callable[..., str], textChanged))
    def text(self) -> str:
        return QGuiApplication.clipboard().text()

    @Property(str, notify=cast(Callable[..., str], urlChanged))
    def url(self) -> str:
        m = _URL_RE.search(QGuiApplication.clipboard().text())
        return m.group(0) if m else ""

    def __init__(self, parent: QObject | None = None) -> None:
        super().__init__(parent)
        QGuiApplication.clipboard().dataChanged.connect(self._onDataChanged)

    def _onDataChanged(self) -> None:
        self.textChanged.emit()
        self.urlChanged.emit()
