from __future__ import annotations

from typing import Callable, cast

from PySide6.QtCore import QObject, Property, Signal, Slot
from PySide6.QtGui import QGuiApplication
from PySide6.QtQml import QQmlContext


class ClipboardService(QObject):
    """Reads the system clipboard for QML. Text is live-updated via textChanged."""

    textChanged = Signal()

    @Slot(str)
    def copy(self, text: str) -> None:
        QGuiApplication.clipboard().setText(text)

    @Property(str, notify=cast(Callable[..., str], textChanged))
    def text(self) -> str:
        return QGuiApplication.clipboard().text()

    def __init__(self, parent: QObject | None = None) -> None:
        super().__init__(parent)
        QGuiApplication.clipboard().dataChanged.connect(self.textChanged)
