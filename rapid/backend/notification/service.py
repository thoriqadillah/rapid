from __future__ import annotations

from PySide6.QtCore import QObject, Signal, Slot
from PySide6.QtGui import QIcon
from PySide6.QtWidgets import QMenu, QSystemTrayIcon


class NotificationService(QObject):
    """Publishes notifications to both QML and the desktop."""

    notificationRequested = Signal(str, str, str)  # type, title, message
    openRequested = Signal()
    newDownloadRequested = Signal()
    quitRequested = Signal()

    def __init__(
        self,
        parent: QObject | None = None,
        trayIcon: QSystemTrayIcon | None = None,
        notificationIcon: QIcon | None = None,
        trayMenu: QMenu | None = None,
    ) -> None:
        super().__init__(parent)
        self._notificationIcon = (
            notificationIcon
            if notificationIcon is not None
            else QIcon(":/icons/rapid.svg")
        )
        self._trayIcon = (
            trayIcon
            if trayIcon is not None
            else QSystemTrayIcon(self._notificationIcon, self)
        )
        self._ownsTrayMenu = trayMenu is None
        self._closed = False
        self._trayMenu = trayMenu if trayMenu is not None else QMenu()
        self._openAction = self._trayMenu.addAction("Open Rapid")
        self._newDownloadAction = self._trayMenu.addAction("New download")
        self._quitAction = self._trayMenu.addAction("Quit")

        self._openAction.triggered.connect(
            lambda _checked=False: self.openRequested.emit()
        )
        self._newDownloadAction.triggered.connect(
            lambda _checked=False: self.newDownloadRequested.emit()
        )
        self._quitAction.triggered.connect(
            lambda _checked=False: self.quitRequested.emit()
        )

        self._trayIcon.setToolTip("Rapid")
        self._trayIcon.setContextMenu(self._trayMenu)
        self._trayIcon.activated.connect(self._onTrayActivated)
        self._trayIcon.show()

    def close(self) -> None:
        """Release native tray resources before QApplication is destroyed."""
        if self._closed:
            return
        self._closed = True

        self._trayIcon.hide()

        self._trayMenu.close()
        if self._ownsTrayMenu:
            self._trayMenu.deleteLater()

    def _onTrayActivated(self, reason: QSystemTrayIcon.ActivationReason) -> None:
        if reason in (
            QSystemTrayIcon.ActivationReason.Trigger,
            QSystemTrayIcon.ActivationReason.DoubleClick,
        ):
            self.openRequested.emit()

    @Slot(str, str, bool)
    def error(self, title: str, message: str, shouldNotify: bool = False) -> None:
        self._publish("error", title, message, shouldNotify)

    @Slot(str, str, bool)
    def success(self, title: str, message: str, shouldNotify: bool = False) -> None:
        self._publish("success", title, message, shouldNotify)

    @Slot(str, str, bool)
    def info(self, title: str, message: str, shouldNotify: bool = False) -> None:
        self._publish("info", title, message, shouldNotify)

    def _publish(
        self,
        kind: str,
        title: str,
        message: str,
        shouldNotify: bool = True,
    ) -> None:
        self.notificationRequested.emit(kind, title, message)
        if shouldNotify:
            self._desktopNotify(title, message)

    def _desktopNotify(
        self,
        title: str,
        message: str,
    ) -> None:
        self._trayIcon.showMessage(title, message, self._notificationIcon, 5000)
