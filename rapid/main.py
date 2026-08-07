import shutil
import subprocess
import signal
import sys
from pathlib import Path

from PySide6.QtCore import QObject, QStandardPaths, QTimer, QUrl, Slot
from PySide6.QtWidgets import QApplication, QFileDialog
from PySide6.QtQml import QQmlApplicationEngine

from .backend import Aria2Client, DownloadStore, PluginManager

BASE_DIR = Path(__file__).resolve().parent


class FileDialogs(QObject):
    def __init__(self, default_dir: str, parent: QObject | None = None) -> None:
        super().__init__(parent)
        self._default_dir = default_dir

    @Slot(str, result=str)
    def pickFolder(self, start_dir: str) -> str:
        """Open the OS-native directory chooser and return the picked path."""
        for tool, args in (
            ("zenity", ["--file-selection", "--directory"]),
            ("kdialog", ["--getexistingdirectory"]),
        ):
            program = shutil.which(tool)
            if program is None:
                continue

            result = subprocess.run([program, *args], capture_output=True, timeout=60)
            return result.stdout.decode().strip() if result.returncode == 0 else ""

        return QFileDialog.getExistingDirectory(
            None, "Select destination", start_dir or self._default_dir
        )


def _plugin_dirs() -> list[Path]:
    user = (
        Path(
            QStandardPaths.writableLocation(QStandardPaths.StandardLocation.AppDataLocation)
        )
        / "plugins"
    )
    dev = BASE_DIR / "plugins"
    bundled = Path(sys.executable).resolve().parent / "plugins"
    deduped = {p.resolve(): p for p in (user, dev, bundled)}
    return list(deduped.values())


def main() -> int:
    app = QApplication(sys.argv)

    signal.signal(signal.SIGINT, lambda *_: app.quit())
    signal.signal(signal.SIGTERM, lambda *_: app.quit())
    # Qt's event loop is native code and never yields to Python, so signals
    # never get delivered without this: wake up periodically to run them.
    signal_pump = QTimer()
    signal_pump.timeout.connect(lambda: None)
    signal_pump.start(200)

    store = DownloadStore()
    downloadDir = QStandardPaths.writableLocation(QStandardPaths.StandardLocation.DownloadLocation)
    aria2 = Aria2Client(store=store, download_dir=Path(downloadDir))
    plugins = PluginManager(_plugin_dirs())

    engine = QQmlApplicationEngine()
    engine.rootContext().setContextProperty("Aria2", aria2)
    engine.rootContext().setContextProperty("Plugins", plugins)
    dialogs = FileDialogs(downloadDir)
    engine.rootContext().setContextProperty("Dialogs", dialogs)

    engine.load(QUrl.fromLocalFile(str(BASE_DIR / "qml" / "components" / "download" / "DownloadDialog.qml")))
    if not engine.rootObjects():
        return 1
    downloadDialog = engine.rootObjects()[0]
    downloadDialog.setProperty("defaultDir", downloadDir)
    engine.rootContext().setContextProperty("NewDownloadDialog", downloadDialog)

    engine.load(QUrl.fromLocalFile(str(BASE_DIR / "qml" / "Main.qml")))
    if not engine.rootObjects():
        return 1

    aria2.start()

    exit_code = app.exec()
    aria2.stop()
    del engine  # tear down QML before app to avoid shutdown warnings
    return exit_code


if __name__ == "__main__":
    sys.exit(main())
