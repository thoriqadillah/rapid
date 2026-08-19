import signal
import sys
from pathlib import Path

from PySide6.QtCore import QStandardPaths, QTimer, QUrl, QCoreApplication
from PySide6.QtWidgets import QApplication
from PySide6.QtQml import QQmlApplicationEngine, QQmlComponent, QQmlContext

from .backend import Aria2Downloader, ClipboardService, Database, DownloadService, DownloadStore
from .qml.icons import icons_rc  # noqa: F401  registers qrc resources on import

BASE_DIR = Path(__file__).resolve().parent
DOWNLOAD_DIR = Path(QStandardPaths.writableLocation(QStandardPaths.StandardLocation.DownloadLocation))
PLUGIN_DIRS = [
    BASE_DIR / "plugins",
    Path(QStandardPaths.writableLocation(QStandardPaths.StandardLocation.AppDataLocation)) / "plugins",
]

def downloader_service(engine: QQmlApplicationEngine) -> DownloadService:
    store = DownloadStore(Database())
    downloader = Aria2Downloader(downloadDir=DOWNLOAD_DIR)
    service = DownloadService(
        DOWNLOAD_DIR,
        store,
        downloader,
        downloader,
        PLUGIN_DIRS,
    )
    engine.rootContext().setContextProperty("DownloadService", service)
    return service


def clipboard_service(context: QQmlContext) -> ClipboardService:
    return ClipboardService()


def main() -> int:
    QCoreApplication.setApplicationName("rapid")
    app = QApplication(sys.argv)

    signal.signal(signal.SIGINT, lambda *_: app.quit())
    signal.signal(signal.SIGTERM, lambda *_: app.quit())
    # Qt's event loop is native code and never yields to Python, so signals
    # never get delivered without this: wake up periodically to run them.
    signal_pump = QTimer()
    signal_pump.timeout.connect(lambda: None)
    signal_pump.start(200)

    engine = QQmlApplicationEngine()
    downloader = downloader_service(engine)

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
    downloadDialog.setProperty("defaultDir", str(DOWNLOAD_DIR))
    engine.rootContext().setContextProperty("DownloadDialog", downloadDialog)

    engine.load(QUrl.fromLocalFile(str(BASE_DIR / "qml" / "Main.qml")))
    if not engine.rootObjects():
        return 1

    downloader.start()
    exit_code = app.exec()
    downloader.stop()

    del engine  # tear down QML before app to avoid shutdown warnings
    return exit_code


if __name__ == "__main__":
    sys.exit(main())
