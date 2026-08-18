from __future__ import annotations

from datetime import datetime, timezone
from typing import Any

from sqlalchemy import (
    BigInteger,
    Boolean,
    DateTime,
    ForeignKey,
    Index,
    Integer,
    JSON,
    String,
    UniqueConstraint,
    delete,
    select,
)
from sqlalchemy.dialects.sqlite import insert as sqlite_insert
from sqlalchemy.orm import (
    Mapped,
    Session,
    mapped_column,
    relationship,
    selectinload,
)

from ..database.database import Base, Database
from .models import DownloadFile, Download, FileUri, ResolvedUrl, SpeedSample


def _now() -> datetime:
    return datetime.now(timezone.utc)


class _UriRow(Base):
    __tablename__ = "file_uris"
    __table_args__ = (Index("ix_file_uris_file_id", "file_id"),)

    id: Mapped[int] = mapped_column(primary_key=True, autoincrement=True)
    file_id: Mapped[int] = mapped_column(
        ForeignKey("download_files.id", ondelete="CASCADE")
    )
    uri: Mapped[str | None] = mapped_column(String)
    status: Mapped[str | None] = mapped_column(String)

    file: Mapped["_FileRow"] = relationship(back_populates="uris")


class _FileRow(Base):
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


class _SpeedSample(Base):
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


class _Download(Base):
    __tablename__ = "downloads"

    gid: Mapped[str] = mapped_column(String, primary_key=True)
    status: Mapped[str | None] = mapped_column(String)
    dir: Mapped[str | None] = mapped_column("dir", String)
    category: Mapped[str | None] = mapped_column("category", String)
    total_length: Mapped[int | None] = mapped_column(BigInteger)
    completed_length: Mapped[int | None] = mapped_column(BigInteger)
    upload_length: Mapped[int | None] = mapped_column(BigInteger)
    download_speed: Mapped[int | None] = mapped_column(BigInteger)
    upload_speed: Mapped[int | None] = mapped_column(BigInteger)
    connections: Mapped[int | None] = mapped_column(Integer)
    num_pieces: Mapped[int | None] = mapped_column(Integer)
    piece_length: Mapped[int | None] = mapped_column(BigInteger)
    verified_length: Mapped[int | None] = mapped_column(BigInteger)
    error_code: Mapped[int | None] = mapped_column(Integer)
    error_message: Mapped[str | None] = mapped_column(String)
    resolved_url: Mapped[dict[str, Any] | None] = mapped_column(JSON)
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=_now)

    files: Mapped[list[_FileRow]] = relationship(
        back_populates="download", cascade="all, delete-orphan"
    )


def _toDownloadModel(row: _Download) -> Download:
    return Download(
        gid=row.gid,
        status=row.status,
        dir=row.dir,
        category=row.category,
        totalLength=row.total_length,
        completedLength=row.completed_length,
        uploadLength=row.upload_length,
        downloadSpeed=row.download_speed,
        uploadSpeed=row.upload_speed,
        connections=row.connections,
        numPieces=row.num_pieces,
        pieceLength=row.piece_length,
        verifiedLength=row.verified_length,
        errorCode=row.error_code,
        errorMessage=row.error_message,
        resolved=ResolvedUrl.fromDict(row.resolved_url) if row.resolved_url else None,
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

    def __init__(self, db: Database) -> None:
        self._session_factory = db.session_factory

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
            return {row.gid: _toDownloadModel(row) for row in rows}

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
            return _toDownloadModel(row) if row is not None else None

    def upsert(self, status: Download) -> None:
        gid = status.gid
        with self._session_factory() as session:
            row = session.get(_Download, gid)
            if row is None:
                row = _Download(gid=gid)
                session.add(row)

            self._applyDownload(session, row, status)
            row.updated_at = _now()
            session.commit()

    @staticmethod
    def _applyDownload(session: Session, row: _Download, status: Download) -> None:
        if status.status is not None:
            row.status = status.status
        if status.dir is not None:
            row.dir = status.dir
        if status.category is not None:
            row.category = status.category
        if status.totalLength is not None:
            row.total_length = status.totalLength
        if status.completedLength is not None:
            row.completed_length = status.completedLength
        if status.uploadLength is not None:
            row.upload_length = status.uploadLength
        if status.downloadSpeed is not None:
            row.download_speed = status.downloadSpeed
        if status.uploadSpeed is not None:
            row.upload_speed = status.uploadSpeed
        if status.connections is not None:
            row.connections = status.connections
        if status.numPieces is not None:
            row.num_pieces = status.numPieces
        if status.pieceLength is not None:
            row.piece_length = status.pieceLength
        if status.verifiedLength is not None:
            row.verified_length = status.verifiedLength
        if status.errorCode is not None:
            row.error_code = status.errorCode
        if status.errorMessage is not None:
            row.error_message = status.errorMessage
        if status.resolved is not None:
            row.resolved_url = status.resolved.asDict()

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

    def addSpeedSample(self, gid: str, ts: int, speed: int) -> None:
        with self._session_factory() as session:
            session.execute(
                sqlite_insert(_SpeedSample)
                .values(gid=gid, ts=ts, speed=speed)
                .on_conflict_do_nothing(index_elements=["gid", "ts"])
            )
            session.commit()

    def speedHistory(self, gid: str, limit: int = 60) -> list[SpeedSample]:
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
