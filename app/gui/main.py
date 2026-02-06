import os
import sys

from PySide6.QtCore import QUrl
from PySide6.QtWidgets import QApplication, QSystemTrayIcon
from PySide6.QtQml import QQmlApplicationEngine


def main():
    app = QApplication(sys.argv)
    engine = QQmlApplicationEngine()

    base_path = getattr(sys, "_MEIPASS", os.path.abspath(os.path.dirname(__file__)))
    qml_path = os.path.join(base_path, "qml", "Main.qml")
    engine.load(QUrl.fromLocalFile(qml_path))

    if not engine.rootObjects():
        sys.exit(-1)

    sys.exit(app.exec())


if __name__ == "__main__":
    main()
