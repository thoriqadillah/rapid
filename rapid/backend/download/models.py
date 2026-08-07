from __future__ import annotations

from dataclasses import dataclass
from typing import Any


def _str(value: Any) -> str | None:
    return value if isinstance(value, str) else None


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


def _num(value: int | None) -> str | None:
    return None if value is None else str(value)


def _bool_str(value: bool | None) -> str | None:
    if value is None:
        return None
    return "true" if value else "false"


@dataclass(frozen=True)
class SpeedSample:
    ts: int
    speed: int

    def as_dict(self) -> dict[str, int]:
        return {"ts": self.ts, "speed": self.speed}


@dataclass(frozen=True)
class FileUri:
    uri: str | None = None
    status: str | None = None

    @classmethod
    def from_payload(cls, data: dict[str, Any]) -> FileUri:
        return cls(uri=_str(data.get("uri")), status=_str(data.get("status")))

    def as_dict(self) -> dict[str, Any]:
        return {"uri": self.uri, "status": self.status}


@dataclass(frozen=True)
class DownloadFile:
    index: int
    path: str | None = None
    length: int | None = None
    completed_length: int | None = None
    selected: bool | None = None
    uris: tuple[FileUri, ...] = ()

    @classmethod
    def from_payload(cls, data: dict[str, Any]) -> DownloadFile:
        index = _to_int(data.get("index")) or 0
        uris = data.get("uris")
        parsed_uris = (
            tuple(FileUri.from_payload(u) for u in uris if isinstance(u, dict))
            if isinstance(uris, list)
            else ()
        )
        return cls(
            index=index,
            path=_str(data.get("path")),
            length=_to_int(data.get("length")),
            completed_length=_to_int(data.get("completedLength")),
            selected=_to_bool(data.get("selected")),
            uris=parsed_uris,
        )

    def as_dict(self) -> dict[str, Any]:
        return {
            "index": str(self.index),
            "path": self.path,
            "length": _num(self.length),
            "completedLength": _num(self.completed_length),
            "selected": _bool_str(self.selected),
            "uris": [u.as_dict() for u in self.uris],
        }


@dataclass(frozen=True)
class Download:
    gid: str
    status: str | None = None
    dir: str | None = None
    total_length: int | None = None
    completed_length: int | None = None
    upload_length: int | None = None
    download_speed: int | None = None
    upload_speed: int | None = None
    connections: int | None = None
    num_pieces: int | None = None
    piece_length: int | None = None
    verified_length: int | None = None
    num_seeders: int | None = None
    seeder: bool | None = None
    info_hash: str | None = None
    bitfield: str | None = None
    error_code: int | None = None
    error_message: str | None = None
    bittorrent: dict[str, Any] | None = None
    files: tuple[DownloadFile, ...] = ()

    @property
    def progress(self) -> float | None:
        if not self.total_length:
            return None
        return (self.completed_length or 0) / self.total_length

    @classmethod
    def from_payload(cls, data: dict[str, Any]) -> Download:
        gid = data.get("gid")
        files = data.get("files")
        bittorrent = data.get("bittorrent")
        parsed_files = (
            tuple(DownloadFile.from_payload(f) for f in files if isinstance(f, dict))
            if isinstance(files, list)
            else ()
        )
        return cls(
            gid=str(gid) if gid is not None else "",
            status=_str(data.get("status")),
            dir=_str(data.get("dir")),
            total_length=_to_int(data.get("totalLength")),
            completed_length=_to_int(data.get("completedLength")),
            upload_length=_to_int(data.get("uploadLength")),
            download_speed=_to_int(data.get("downloadSpeed")),
            upload_speed=_to_int(data.get("uploadSpeed")),
            connections=_to_int(data.get("connections")),
            num_pieces=_to_int(data.get("numPieces")),
            piece_length=_to_int(data.get("pieceLength")),
            verified_length=_to_int(data.get("verifiedLength")),
            num_seeders=_to_int(data.get("numSeeders")),
            seeder=_to_bool(data.get("seeder")),
            info_hash=_str(data.get("infoHash")),
            bitfield=_str(data.get("bitfield")),
            error_code=_to_int(data.get("errorCode")),
            error_message=_str(data.get("errorMessage")),
            bittorrent=bittorrent if isinstance(bittorrent, dict) else None,
            files=parsed_files,
        )

    def as_dict(self) -> dict[str, Any]:
        payload: dict[str, Any] = {
            "gid": self.gid,
            "status": self.status,
            "dir": self.dir,
            "totalLength": _num(self.total_length),
            "completedLength": _num(self.completed_length),
            "uploadLength": _num(self.upload_length),
            "downloadSpeed": _num(self.download_speed),
            "uploadSpeed": _num(self.upload_speed),
            "connections": _num(self.connections),
            "numPieces": _num(self.num_pieces),
            "pieceLength": _num(self.piece_length),
            "verifiedLength": _num(self.verified_length),
            "numSeeders": _num(self.num_seeders),
            "seeder": _bool_str(self.seeder),
            "infoHash": self.info_hash,
            "bitfield": self.bitfield,
            "errorCode": _num(self.error_code),
            "errorMessage": self.error_message,
            "files": [f.as_dict() for f in self.files],
        }
        if self.bittorrent is not None:
            payload["bittorrent"] = self.bittorrent
        return payload
