import os
import signal
import sys
from pathlib import Path

from PySide6.QtCore import QTimer, QUrl, QCoreApplication
from PySide6.QtGui import QGuiApplication
from PySide6.QtQml import QQmlApplicationEngine, QQmlComponent, QQmlContext

from rapid.qml.icons import icons_rc  # noqa: F401  registers qrc resources on import
from rapid.backend import Aria2Downloader, ClipboardService, Database, DownloadFilterProxy, DownloadService, DownloadStore
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


def clipboard_service(context: QQmlContext) -> ClipboardService:
    return ClipboardService()


def main() -> int:
    QCoreApplication.setApplicationName("rapid")
    app = QGuiApplication(sys.argv)
    settings = Settings.default(BASE_DIR)

    signal.signal(signal.SIGINT, lambda *_: app.quit())
    signal.signal(signal.SIGTERM, lambda *_: app.quit())
    # Qt's event loop is native code and never yields to Python, so signals
    # never get delivered without this: wake up periodically to run them.
    signal_pump = QTimer()
    signal_pump.timeout.connect(lambda: None)
    signal_pump.start(1000)

    engine = QQmlApplicationEngine()
    downloader = downloader_service(engine, settings)

    dialogContext = QQmlContext(engine.rootContext(), engine)
    dialogComponent = QQmlComponent(
        engine, QUrl.fromLocalFile(str(BASE_DIR / "qml" / "components" / "download" / "DownloadDialog.qml"))
    )
    clipboard = clipboard_service(dialogContext)
    engine.rootContext().setContextProperty("Clipboard", clipboard)
    downloadDialog = dialogComponent.create(dialogContext)
    if downloadDialog is None:
        return 1

    clipboard.setParent(downloadDialog)
    downloadDialog.setProperty("defaultDir", str(settings.downloadDir))
    engine.rootContext().setContextProperty("DownloadDialog", downloadDialog)

    engine.load(QUrl.fromLocalFile(str(BASE_DIR / "qml" / "Main.qml")))
    if not engine.rootObjects():
        return 1

    downloader.start()
    exit_code = app.exec()
    downloader.close()

    del engine  # tear down QML before app to avoid shutdown warnings
    return exit_code


if __name__ == "__main__":
    os._exit(main())
