import signal
import sys
from pathlib import Path

from PySide6.QtCore import QTimer, QUrl
from PySide6.QtGui import QGuiApplication
from PySide6.QtQml import QQmlApplicationEngine

from .backend import Aria2Client, DownloadStore

BASE_DIR = Path(__file__).resolve().parent


def main() -> int:
    app = QGuiApplication(sys.argv)

    signal.signal(signal.SIGINT, lambda *_: app.quit())
    signal.signal(signal.SIGTERM, lambda *_: app.quit())
    # Qt's event loop is native code and never yields to Python, so signals
    # never get delivered without this: wake up periodically to run them.
    signal_pump = QTimer()
    signal_pump.timeout.connect(lambda: None)
    signal_pump.start(200)

    store = DownloadStore()
    aria2 = Aria2Client(store=store)

    engine = QQmlApplicationEngine()
    engine.rootContext().setContextProperty("Aria2", aria2)
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
