from __future__ import annotations

from pathlib import Path
from typing import Any

from PySide6.QtCore import QStandardPaths
from sqlalchemy import create_engine, event
from sqlalchemy.orm import DeclarativeBase, sessionmaker


def default_db_path() -> Path:
    base = QStandardPaths.writableLocation(QStandardPaths.StandardLocation.AppDataLocation)
    return Path(base) / "database.db"


class Base(DeclarativeBase):
    pass


class Database:
    """Process-wide SQLite connection.

    ``Database()`` returns the shared instance. ``Database(path)`` builds a
    new one and installs it as the shared instance, so tests can point the
    whole app at a temp file.
    """

    _default: Database | None = None

    def __new__(cls, path: Path | None = None) -> Database:
        if path is None and cls._default is not None:
            return cls._default
        instance = super().__new__(cls)
        cls._default = instance
        return instance

    def __init__(self, path: Path | None = None) -> None:
        if getattr(self, "path", None) is not None:
            return
        self.path = path or default_db_path()
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self.engine = create_engine(f"sqlite:///{self.path}", future=True)
        event.listen(self.engine, "connect", self._enable_foreign_keys)
        self.session_factory = sessionmaker(bind=self.engine, expire_on_commit=False)
        Base.metadata.create_all(self.engine)

    @staticmethod
    def _enable_foreign_keys(dbapi_connection: Any, _record: Any) -> None:
        cursor = dbapi_connection.cursor()
        cursor.execute("PRAGMA foreign_keys=ON")
        cursor.close()
