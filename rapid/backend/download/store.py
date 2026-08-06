from __future__ import annotations

import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from PySide6.QtCore import QStandardPaths
from sqlalchemy import (
    BigInteger,
    Boolean,
    DateTime,
    ForeignKey,
    Integer,
    String,
    UniqueConstraint,
    create_engine,
    delete,
    event,
    select,
)
from sqlalchemy.dialects.sqlite import insert as sqlite_insert
from sqlalchemy.engine import Engine
from sqlalchemy.orm import (
    DeclarativeBase,
    Mapped,
    Session,
    mapped_column,
    relationship,
    selectinload,
    sessionmaker,
)


def _default_path() -> Path:
    base = QStandardPaths.writableLocation(QStandardPaths.StandardLocation.AppDataLocation)
    return Path(base) / "downloads.db"


def _now() -> datetime:
    return datetime.now(timezone.utc)


class _Base(DeclarativeBase):
    pass


class _UriRow(_Base):
    __tablename__ = "file_uris"

    id: Mapped[int] = mapped_column(primary_key=True, autoincrement=True)
    file_id: Mapped[int] = mapped_column(
        ForeignKey("download_files.id", ondelete="CASCADE")
    )
    uri: Mapped[str | None] = mapped_column(String)
    status: Mapped[str | None] = mapped_column(String)

    file: Mapped["_FileRow"] = relationship(back_populates="uris")


class _FileRow(_Base):
    __tablename__ = "download_files"
    __table_args__ = (UniqueConstraint("gid", "index", name="uq_download_file"),)

    id: Mapped[int] = mapped_column(primary_key=True, autoincrement=True)
    gid: Mapped[str] = mapped_column(ForeignKey("downloads.gid", ondelete="CASCADE"))
    index: Mapped[int] = mapped_column(Integer)
    path: Mapped[str | None] = mapped_column(String)
    length: Mapped[int | None] = mapped_column(BigInteger)
    completed_length: Mapped[int | None] = mapped_column(BigInteger)
    selected: Mapped[bool | None] = mapped_column(Boolean)

    download: Mapped["_DownloadRow"] = relationship(back_populates="files")
    uris: Mapped[list[_UriRow]] = relationship(
        back_populates="file", cascade="all, delete-orphan"
    )


class _SpeedSample(_Base):
    __tablename__ = "speed_samples"
    __table_args__ = {"sqlite_with_rowid": False}

    gid: Mapped[str] = mapped_column(
        ForeignKey("downloads.gid", ondelete="CASCADE"), primary_key=True
    )
    ts: Mapped[int] = mapped_column(BigInteger, primary_key=True)
    speed: Mapped[int] = mapped_column(BigInteger)


class _DownloadRow(_Base):
    __tablename__ = "downloads"

    gid: Mapped[str] = mapped_column(String, primary_key=True)
    status: Mapped[str | None] = mapped_column(String)
    dir: Mapped[str | None] = mapped_column("dir", String)
    total_length: Mapped[int | None] = mapped_column(BigInteger)
    completed_length: Mapped[int | None] = mapped_column(BigInteger)
    upload_length: Mapped[int | None] = mapped_column(BigInteger)
    download_speed: Mapped[int | None] = mapped_column(BigInteger)
    upload_speed: Mapped[int | None] = mapped_column(BigInteger)
    connections: Mapped[int | None] = mapped_column(Integer)
    num_pieces: Mapped[int | None] = mapped_column(Integer)
    piece_length: Mapped[int | None] = mapped_column(BigInteger)
    verified_length: Mapped[int | None] = mapped_column(BigInteger)
    num_seeders: Mapped[int | None] = mapped_column(Integer)
    seeder: Mapped[bool | None] = mapped_column(Boolean)
    info_hash: Mapped[str | None] = mapped_column(String)
    bitfield: Mapped[str | None] = mapped_column(String)
    error_code: Mapped[int | None] = mapped_column(Integer)
    error_message: Mapped[str | None] = mapped_column(String)
    bittorrent: Mapped[str | None] = mapped_column(String)
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=_now)

    files: Mapped[list[_FileRow]] = relationship(
        back_populates="download", cascade="all, delete-orphan"
    )


def _to_int(value: Any) -> int | None:
    if value is None or isinstance(value, bool):
        return None
    if isinstance(value, int):
        return value
    try:
        return int(value)
    except (TypeError, ValueError):
        return None


def _to_bool(value: Any) -> bool | None:
    if value is None:
        return None
    if isinstance(value, bool):
        return value
    if isinstance(value, str):
        return value == "true"
    return bool(value)


def _num_str(value: int | None) -> str | None:
    return None if value is None else str(value)


def _bool_str(value: bool | None) -> str | None:
    if value is None:
        return None
    return "true" if value else "false"


def _row_to_payload(row: _DownloadRow) -> dict[str, Any]:
    payload: dict[str, Any] = {
        "gid": row.gid,
        "status": row.status,
        "dir": row.dir,
        "totalLength": _num_str(row.total_length),
        "completedLength": _num_str(row.completed_length),
        "uploadLength": _num_str(row.upload_length),
        "downloadSpeed": _num_str(row.download_speed),
        "uploadSpeed": _num_str(row.upload_speed),
        "connections": _num_str(row.connections),
        "numPieces": _num_str(row.num_pieces),
        "pieceLength": _num_str(row.piece_length),
        "verifiedLength": _num_str(row.verified_length),
        "numSeeders": _num_str(row.num_seeders),
        "seeder": _bool_str(row.seeder),
        "infoHash": row.info_hash,
        "bitfield": row.bitfield,
        "errorCode": _num_str(row.error_code),
        "errorMessage": row.error_message,
    }
    if row.bittorrent is not None:
        payload["bittorrent"] = json.loads(row.bittorrent)
    payload["files"] = [
        {
            "index": _num_str(f.index),
            "path": f.path,
            "length": _num_str(f.length),
            "completedLength": _num_str(f.completed_length),
            "selected": _bool_str(f.selected),
            "uris": [{"uri": u.uri, "status": u.status} for u in sorted(f.uris, key=lambda u: u.id)],
        }
        for f in sorted(row.files, key=lambda f: f.index)
    ]
    return payload


class DownloadStore:
    """Persist downloads to SQLite via SQLAlchemy using a normalized schema.

    Flat aria2 ``tellStatus`` fields map to typed columns on ``downloads``,
    with ``download_files`` and ``file_uris`` holding the nested lists.
    Reads reconstruct the original aria2-shaped payload dict.
    """

    def __init__(self, path: Path | None = None) -> None:
        self._path = path or _default_path()
        self._path.parent.mkdir(parents=True, exist_ok=True)
        self._engine = create_engine(f"sqlite:///{self._path}", future=True)
        self._enable_foreign_keys()
        _Base.metadata.create_all(self._engine)
        self._session_factory = sessionmaker(bind=self._engine, expire_on_commit=False)

    def _enable_foreign_keys(self) -> None:
        @event.listens_for(self._engine, "connect")
        def _on_connect(dbapi_connection: Any, _record: Any) -> None:
            cursor = dbapi_connection.cursor()
            cursor.execute("PRAGMA foreign_keys=ON")
            cursor.close()

    def all(self) -> dict[str, dict[str, Any]]:
        with self._session_factory() as session:
            rows = (
                session.execute(
                    select(_DownloadRow).options(
                        selectinload(_DownloadRow.files).selectinload(_FileRow.uris)
                    )
                )
                .scalars()
                .all()
            )
            return {row.gid: _row_to_payload(row) for row in rows}

    def get(self, gid: str) -> dict[str, Any] | None:
        with self._session_factory() as session:
            row = (
                session.execute(
                    select(_DownloadRow)
                    .where(_DownloadRow.gid == gid)
                    .options(selectinload(_DownloadRow.files).selectinload(_FileRow.uris))
                )
                .scalar_one_or_none()
            )
            return _row_to_payload(row) if row is not None else None

    def upsert(self, gid: str, data: dict[str, Any]) -> None:
        with self._session_factory() as session:
            row = session.get(_DownloadRow, gid)
            if row is None:
                row = _DownloadRow(gid=gid)
                session.add(row)
            self._apply_payload(session, row, data)
            row.updated_at = _now()
            session.commit()

    @staticmethod
    def _apply_payload(session: Session, row: _DownloadRow, data: dict[str, Any]) -> None:
        _set_str(row, "status", data)
        _set_str(row, "dir", data)
        _set_int(row, "total_length", "totalLength", data)
        _set_int(row, "completed_length", "completedLength", data)
        _set_int(row, "upload_length", "uploadLength", data)
        _set_int(row, "download_speed", "downloadSpeed", data)
        _set_int(row, "upload_speed", "uploadSpeed", data)
        _set_int(row, "connections", "connections", data)
        _set_int(row, "num_pieces", "numPieces", data)
        _set_int(row, "piece_length", "pieceLength", data)
        _set_int(row, "verified_length", "verifiedLength", data)
        _set_int(row, "num_seeders", "numSeeders", data)
        if "seeder" in data:
            seeder = _to_bool(data.get("seeder"))
            if seeder is not None:
                row.seeder = seeder
        _set_str(row, "info_hash", data)
        _set_str(row, "bitfield", data)
        _set_int(row, "error_code", "errorCode", data)
        _set_str(row, "error_message", data)
        bt = data.get("bittorrent")
        if isinstance(bt, dict):
            row.bittorrent = json.dumps(bt)

        files = data.get("files")
        if not isinstance(files, list):
            return
        existing = {f.index: f for f in row.files}
        seen: set[int] = set()
        for item in files:
            if not isinstance(item, dict):
                continue
            index = _to_int(item.get("index"))
            if index is None:
                continue
            seen.add(index)
            file_row = existing.get(index)
            if file_row is None:
                file_row = _FileRow(gid=row.gid, index=index)
                row.files.append(file_row)
            if isinstance(item.get("path"), str):
                file_row.path = item.get("path")
            _assign_int(file_row, "length", item.get("length"))
            _assign_int(file_row, "completed_length", item.get("completedLength"))
            if "selected" in item:
                selected = _to_bool(item.get("selected"))
                if selected is not None:
                    file_row.selected = selected

            file_row.uris.clear()
            uris = item.get("uris")
            if isinstance(uris, list):
                for u in uris:
                    if not isinstance(u, dict):
                        continue
                    file_row.uris.append(
                        _UriRow(
                            uri=u.get("uri") if isinstance(u.get("uri"), str) else None,
                            status=u.get("status") if isinstance(u.get("status"), str) else None,
                        )
                    )
        for index, file_row in existing.items():
            if index not in seen:
                row.files.remove(file_row)

    def add_speed_sample(self, gid: str, ts: int, speed: int) -> None:
        with self._session_factory() as session:
            session.execute(
                sqlite_insert(_SpeedSample)
                .values(gid=gid, ts=ts, speed=speed)
                .on_conflict_do_nothing(index_elements=["gid", "ts"])
            )
            session.commit()

    def speed_history(
        self, gid: str, limit: int = 60
    ) -> list[dict[str, int]]:
        with self._session_factory() as session:
            rows = (
                session.execute(
                    select(_SpeedSample)
                    .where(_SpeedSample.gid == gid)
                    .order_by(_SpeedSample.ts.desc())
                    .limit(limit)
                )
                .scalars()
                .all()
            )
            return [
                {"ts": row.ts, "speed": row.speed} for row in reversed(rows)
            ]

    def prune_speeds(self, before_ts: int) -> None:
        with self._session_factory() as session:
            session.execute(delete(_SpeedSample).where(_SpeedSample.ts < before_ts))
            session.commit()

    def remove(self, gid: str) -> None:
        with self._session_factory() as session:
            row = session.get(_DownloadRow, gid)
            if row is not None:
                session.delete(row)
                session.commit()

    def clear(self) -> None:
        with self._session_factory() as session:
            session.execute(delete(_DownloadRow))
            session.commit()

    def save(self) -> None:
        pass


def _set_str(row: _DownloadRow, attr: str, data: dict[str, Any]) -> None:
    value = data.get(attr)
    if isinstance(value, str):
        setattr(row, attr, value)


def _set_int(row: _DownloadRow, attr: str, key: str, data: dict[str, Any]) -> None:
    value = _to_int(data.get(key))
    if value is not None:
        setattr(row, attr, value)


def _assign_int(file_row: _FileRow, attr: str, value: Any) -> None:
    parsed = _to_int(value)
    if parsed is not None:
        setattr(file_row, attr, parsed)