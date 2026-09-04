import os
import signal
import sys
from pathlib import Path

from PySide6.QtCore import QTimer, QUrl, QCoreApplication
from PySide6.QtGui import QIcon
from PySide6.QtQml import QQmlApplicationEngine, QQmlComponent, QQmlContext
from PySide6.QtWidgets import QApplication

from rapid.qml.icons import icons_rc  # noqa: F401  registers qrc resources on import
from rapid.backend import (
    Aria2Downloader,
    BrowserIntegration,
    ClipboardService,
    Database,
    DownloadFilterProxy,
    DownloadService,
    DownloadStore,
    NotificationService,
)
from rapid.backend.setting import Settings

BASE_DIR = Path(__file__).resolve().parent

def downloader_service(engine: QQmlApplicationEngine, settings: Settings) -> DownloadService:
    store = DownloadStore(Database())
    downloader = Aria2Downloader(settings=settings)
    service = DownloadService(settings, store, downloader, downloader)
    engine.rootContext().setContextProperty("DownloadService", service)

    downloadFilter = DownloadFilterProxy(service)
    downloadFilter.setSourceModel(service)
    engine.rootContext().setContextProperty("DownloadFilter", downloadFilter)
    return service


def _run() -> int:
    QCoreApplication.setApplicationName("rapid")
    app = QApplication(sys.argv)
    app.setQuitOnLastWindowClosed(False)
    app.setWindowIcon(QIcon(":/icons/rapid.svg"))
    settings = Settings.default(BASE_DIR)

    signal.signal(signal.SIGINT, lambda *_: app.quit())
    signal.signal(signal.SIGTERM, lambda *_: app.quit())
    # Qt's event loop is native code and never yields to Python, so signals
    # never get delivered without this: wake up periodically to run them.
    signal_pump = QTimer()
    signal_pump.timeout.connect(lambda: None)
    signal_pump.start(1000)

    engine = QQmlApplicationEngine()
    notifications = NotificationService(engine)
    engine.rootContext().setContextProperty("NotificationService", notifications)
    downloader = downloader_service(engine, settings)
    downloader.activeCountChanged.connect(notifications.setBadge)

    browserIntegration = BrowserIntegration(parent=engine)
    browserIntegration.errorOccurred.connect(
        lambda message: notifications.error("Browser integration", message, True)
    )
    engine.rootContext().setContextProperty("BrowserIntegration", browserIntegration)

    dialogContext = QQmlContext(engine.rootContext(), engine)
    dialogComponent = QQmlComponent(
        engine, QUrl.fromLocalFile(str(BASE_DIR / "qml" / "components" / "download" / "DownloadDialog.qml"))
    )

    clipboard = ClipboardService()
    engine.rootContext().setContextProperty("Clipboard", clipboard)
    downloadDialog = dialogComponent.create(dialogContext)
    if downloadDialog is None:
        return 1

    clipboard.setParent(downloadDialog)
    downloadDialog.setProperty("defaultDir", str(settings.downloadDir))
    engine.rootContext().setContextProperty("DownloadDialog", downloadDialog)

    quitDialogComponent = QQmlComponent(
        engine,
        QUrl.fromLocalFile(str(BASE_DIR / "qml" / "components" / "download" / "QuitConfirmationDialog.qml")),
    )

    def requestQuit() -> None:
        if not downloader.activeCount:
            app.quit()
            return

        owner = engine.rootObjects()[0]
        quitDialog = quitDialogComponent.create()
        if quitDialog is None:
            return

        quitDialog.setProperty("activeNames", downloader.activeDownloads())
        quitDialog.quitConfirmed.connect(app.quit)  # type: ignore
        quitDialog.openFor(owner)  # type: ignore

    notifications.quitRequested.connect(requestQuit)

    engine.load(QUrl.fromLocalFile(str(BASE_DIR / "qml" / "Main.qml")))
    if not engine.rootObjects():
        return 1

    downloader.start()
    browserIntegration.start()
    exit_code = app.exec()
    browserIntegration.close()
    downloader.close()
    notifications.close()

    del engine  # tear down QML before app to avoid shutdown warnings
    return exit_code


def main() -> None:
    # PySide can crash in Shiboken's interpreter-finalization GC after native
    # tray/QML objects have existed. Services are closed by _run first; bypass
    # Python finalization for both the console entry point and module execution.
    os._exit(_run())


if __name__ == "__main__":
    main()
