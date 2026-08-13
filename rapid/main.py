from rapid.backend.download.service import registerService
import signal
import sys
from pathlib import Path

from PySide6.QtCore import QStandardPaths, QTimer, QUrl, QCoreApplication
from PySide6.QtWidgets import QApplication
from PySide6.QtQml import QQmlApplicationEngine

from .backend import DownloadService
from .qml.icons import icons_rc  # noqa: F401  registers qrc resources on import

BASE_DIR = Path(__file__).resolve().parent

def plugin_dirs() -> list[Path]:
    return [
        BASE_DIR / "plugins",
        Path(QStandardPaths.writableLocation(QStandardPaths.StandardLocation.AppDataLocation)) / "plugins",
    ]


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

    downloadDir = QStandardPaths.writableLocation(QStandardPaths.StandardLocation.DownloadLocation)

    engine = QQmlApplicationEngine()
    downloads = registerService(engine, Path(downloadDir), plugin_dirs())

    engine.load(QUrl.fromLocalFile(str(BASE_DIR / "qml" / "components" / "download" / "DownloadDialog.qml")))
    if not engine.rootObjects():
        return 1
    downloadDialog = engine.rootObjects()[0]
    downloadDialog.setProperty("defaultDir", downloadDir)
    engine.rootContext().setContextProperty("DownloadDialog", downloadDialog)

    engine.load(QUrl.fromLocalFile(str(BASE_DIR / "qml" / "Main.qml")))
    if not engine.rootObjects():
        return 1

    downloads.start()
    exit_code = app.exec()
    downloads.stop()

    del engine  # tear down QML before app to avoid shutdown warnings
    return exit_code


if __name__ == "__main__":
    sys.exit(main())
