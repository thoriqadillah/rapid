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
    Index,
    Integer,
    String,
    UniqueConstraint,
    create_engine,
    delete,
    event,
    select,
)
from sqlalchemy.dialects.sqlite import insert as sqlite_insert
from sqlalchemy.orm import (
    DeclarativeBase,
    Mapped,
    Session,
    mapped_column,
    relationship,
    selectinload,
    sessionmaker,
)

from .models import DownloadFile, Download, FileUri, SpeedSample


def _default_path() -> Path:
    base = QStandardPaths.writableLocation(QStandardPaths.StandardLocation.AppDataLocation)
    return Path(base) / "downloads.db"


def _now() -> datetime:
    return datetime.now(timezone.utc)


class _Base(DeclarativeBase):
    pass


class _UriRow(_Base):
    __tablename__ = "file_uris"
    __table_args__ = (Index("ix_file_uris_file_id", "file_id"),)

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

    download: Mapped["_Download"] = relationship(back_populates="files")
    uris: Mapped[list[_UriRow]] = relationship(
        back_populates="file", cascade="all, delete-orphan"
    )


class _SpeedSample(_Base):
    __tablename__ = "speed_samples"
    __table_args__ = (
        Index("ix_speed_samples_ts", "ts"),
        {"sqlite_with_rowid": False},
    )

    gid: Mapped[str] = mapped_column(
        ForeignKey("downloads.gid", ondelete="CASCADE"), primary_key=True
    )
    ts: Mapped[int] = mapped_column(BigInteger, primary_key=True)
    speed: Mapped[int] = mapped_column(BigInteger)


class _Download(_Base):
    __tablename__ = "downloads"

    gid: Mapped[str] = mapped_column(String, primary_key=True)
    status: Mapped[str | None] = mapped_column(String)
    dir: Mapped[str | None] = mapped_column("dir", String)
    kind: Mapped[str | None] = mapped_column(String)
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


def _to_download_model(row: _Download) -> Download:
    return Download(
        gid=row.gid,
        status=row.status,
        dir=row.dir,
        kind=row.kind,
        total_length=row.total_length,
        completed_length=row.completed_length,
        upload_length=row.upload_length,
        download_speed=row.download_speed,
        upload_speed=row.upload_speed,
        connections=row.connections,
        num_pieces=row.num_pieces,
        piece_length=row.piece_length,
        verified_length=row.verified_length,
        num_seeders=row.num_seeders,
        seeder=row.seeder,
        info_hash=row.info_hash,
        bitfield=row.bitfield,
        error_code=row.error_code,
        error_message=row.error_message,
        bittorrent=json.loads(row.bittorrent) if row.bittorrent is not None else None,
        files=tuple(
            DownloadFile(
                index=f.index,
                path=f.path,
                length=f.length,
                completed_length=f.completed_length,
                selected=f.selected,
                uris=tuple(
                    FileUri(uri=u.uri, status=u.status)
                    for u in sorted(f.uris, key=lambda u: u.id)
                ),
            )
            for f in sorted(row.files, key=lambda f: f.index)
        ),
    )


class DownloadStore:
    """Persist downloads to SQLite via SQLAlchemy using a normalized schema.

    Flat aria2 ``tellStatus`` fields map to typed columns on ``downloads``,
    with ``download_files`` and ``file_uris`` holding the nested lists, and
    ``speed_samples`` holding the per-second throughput history. Reads return
    the typed :class:`DownloadRow` / :class:`SpeedSample` models.
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

    def all(self) -> dict[str, Download]:
        with self._session_factory() as session:
            rows = (
                session.execute(
                    select(_Download).options(
                        selectinload(_Download.files).selectinload(_FileRow.uris)
                    )
                )
                .scalars()
                .all()
            )
            return {row.gid: _to_download_model(row) for row in rows}

    def get(self, gid: str) -> Download | None:
        with self._session_factory() as session:
            row = (
                session.execute(
                    select(_Download)
                    .where(_Download.gid == gid)
                    .options(selectinload(_Download.files).selectinload(_FileRow.uris))
                )
                .scalar_one_or_none()
            )
            return _to_download_model(row) if row is not None else None

    def upsert(self, status: Download) -> None:
        gid = status.gid
        with self._session_factory() as session:
            row = session.get(_Download, gid)
            if row is None:
                row = _Download(gid=gid)
                session.add(row)

            self._apply_download(session, row, status)
            row.updated_at = _now()
            session.commit()

    @staticmethod
    def _apply_download(session: Session, row: _Download, status: Download) -> None:
        if status.status is not None:
            row.status = status.status
        if status.dir is not None:
            row.dir = status.dir
        if status.kind is not None:
            row.kind = status.kind
        if status.total_length is not None:
            row.total_length = status.total_length
        if status.completed_length is not None:
            row.completed_length = status.completed_length
        if status.upload_length is not None:
            row.upload_length = status.upload_length
        if status.download_speed is not None:
            row.download_speed = status.download_speed
        if status.upload_speed is not None:
            row.upload_speed = status.upload_speed
        if status.connections is not None:
            row.connections = status.connections
        if status.num_pieces is not None:
            row.num_pieces = status.num_pieces
        if status.piece_length is not None:
            row.piece_length = status.piece_length
        if status.verified_length is not None:
            row.verified_length = status.verified_length
        if status.num_seeders is not None:
            row.num_seeders = status.num_seeders
        if status.seeder is not None:
            row.seeder = status.seeder
        if status.info_hash is not None:
            row.info_hash = status.info_hash
        if status.bitfield is not None:
            row.bitfield = status.bitfield
        if status.error_code is not None:
            row.error_code = status.error_code
        if status.error_message is not None:
            row.error_message = status.error_message
        if status.bittorrent is not None:
            row.bittorrent = json.dumps(status.bittorrent)

        existing = {f.index: f for f in row.files}
        seen: set[int] = set()
        for file in status.files:
            seen.add(file.index)
            file_row = existing.get(file.index)
            if file_row is None:
                file_row = _FileRow(gid=row.gid, index=file.index)
                row.files.append(file_row)
            if file.path is not None:
                file_row.path = file.path
            if file.length is not None:
                file_row.length = file.length
            if file.completed_length is not None:
                file_row.completed_length = file.completed_length
            if file.selected is not None:
                file_row.selected = file.selected
            file_row.uris.clear()
            for uri in file.uris:
                file_row.uris.append(_UriRow(uri=uri.uri, status=uri.status))
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

    def speed_history(self, gid: str, limit: int = 60) -> list[SpeedSample]:
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
            return [SpeedSample(ts=row.ts, speed=row.speed) for row in reversed(rows)]

    def prune_speeds(self, before_ts: int) -> None:
        with self._session_factory() as session:
            session.execute(delete(_SpeedSample).where(_SpeedSample.ts < before_ts))
            session.commit()

    def remove(self, gid: str) -> None:
        with self._session_factory() as session:
            row = session.get(_Download, gid)
            if row is not None:
                session.delete(row)
                session.commit()

    def clear(self) -> None:
        with self._session_factory() as session:
            session.execute(delete(_Download))
            session.commit()

    def save(self) -> None:
        pass
