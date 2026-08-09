from __future__ import annotations

from PySide6.QtCore import QCoreApplication

from ..database.database import Database
from ..download.store import DownloadStore
from .download_seeder import DownloadSeeder
from .seeder import SeederService


def main() -> None:
    QCoreApplication.setApplicationName("rapid")
    QCoreApplication([])
    db = Database()
    seeder = SeederService([
        DownloadSeeder(DownloadStore(db)),
    ])

    seeder.seed()


if __name__ == "__main__":
    main()
