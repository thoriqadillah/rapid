from __future__ import annotations

from PySide6.QtCore import QObject, Signal, Slot, QSize
from PySide6.QtGui import QColor, QFont, QIcon, QPainter, QPixmap
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

    def setBadge(self, count: int) -> None:
        """Overlay a count badge on the tray icon. 0 clears the badge."""
        size = self._notificationIcon.actualSize(QSize(64, 64))
        pixmap = QPixmap(size)
        pixmap.fill(0)
        painter = QPainter(pixmap)
        painter.setRenderHint(QPainter.RenderHint.Antialiasing)
        self._notificationIcon.paint(painter, pixmap.rect())

        if count > 0:
            text = str(count) if count < 100 else "99+"
            badgeSize = max(size.width() // 2, 10)
            font = QFont()
            font.setPixelSize(badgeSize * 3 // 4)
            font.setBold(True)
            painter.setFont(font)
            metrics = painter.fontMetrics()
            textRect = metrics.boundingRect(text)
            bw = max(textRect.width() + 6, badgeSize)
            bh = max(textRect.height() + 2, badgeSize)
            painter.setPen(QColor("white"))
            painter.setBrush(QColor("#e53935"))
            painter.drawRoundedRect(
                size.width() - bw, size.height() - bh, bw, bh, 4, 4
            )
            painter.drawText(
                size.width() - bw,
                size.height() - bh,
                bw,
                bh,
                0x0084,  # AlignCenter
                text,
            )

        painter.end()
        self._trayIcon.setIcon(QIcon(pixmap))

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
