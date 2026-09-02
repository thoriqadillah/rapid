from __future__ import annotations

from dataclasses import dataclass, asdict, field
from pathlib import Path
from typing import Any
from uuid import uuid4


def _str(value: Any) -> str | None:
    return value if isinstance(value, str) else None


def _toInt(value: Any) -> int | None:
    if value is None or isinstance(value, bool):
        return None
    if isinstance(value, int):
        return value
    try:
        return int(value)
    except (TypeError, ValueError):
        return None


def _toBool(value: Any) -> bool | None:
    if value is None:
        return None
    if isinstance(value, bool):
        return value
    if isinstance(value, str):
        return value == "true"
    return bool(value)


def _num(value: int | None) -> str | None:
    return None if value is None else str(value)


def _boolStr(value: bool | None) -> str | None:
    if value is None:
        return None
    return "true" if value else "false"


@dataclass(frozen=True)
class SpeedSample:
    ts: int
    speed: int

    def asDict(self) -> dict[str, int]:
        return {"ts": self.ts, "speed": self.speed}


@dataclass(frozen=True)
class FileUri:
    uri: str | None = None
    status: str | None = None

    @classmethod
    def fromPayload(cls, data: dict[str, Any]) -> FileUri:
        return cls(uri=_str(data.get("uri")), status=_str(data.get("status")))

    def asDict(self) -> dict[str, Any]:
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
    def fromPayload(cls, data: dict[str, Any]) -> DownloadFile:
        index = _toInt(data.get("index")) or 0
        uris = data.get("uris")
        parsed_uris = (
            tuple(FileUri.fromPayload(u) for u in uris if isinstance(u, dict))
            if isinstance(uris, list)
            else ()
        )
        return cls(
            index=index,
            path=_str(data.get("path")),
            length=_toInt(data.get("length")),
            completed_length=_toInt(data.get("completedLength")),
            selected=_toBool(data.get("selected")),
            uris=parsed_uris,
        )

    def asDict(self) -> dict[str, Any]:
        return {
            "index": str(self.index),
            "path": self.path,
            "length": _num(self.length),
            "completedLength": _num(self.completed_length),
            "selected": _boolStr(self.selected),
            "uris": [u.asDict() for u in self.uris],
        }


@dataclass(frozen=True)
class Download:
    gid: str
    status: str | None = None
    dir: str | None = None
    category: str | None = None
    totalLength: int | None = None
    completedLength: int | None = None
    downloadSpeed: int | None = None
    connections: int | None = None
    numPieces: int | None = None
    pieceLength: int | None = None
    verifiedLength: int | None = None
    errorCode: int | None = None
    errorMessage: str | None = None
    resolved: "ResolvedUrl | None" = None
    files: tuple[DownloadFile, ...] = ()

    @property
    def progress(self) -> float | None:
        if not self.totalLength:
            return None
        return (self.completedLength or 0) / self.totalLength

    @classmethod
    def fromPayload(cls, data: dict[str, Any]) -> Download:
        gid = data.get("gid")
        files = data.get("files")
        parsedFiles = (
            tuple(DownloadFile.fromPayload(f) for f in files if isinstance(f, dict))
            if isinstance(files, list)
            else ()
        )
        return cls(
            gid=str(gid) if gid is not None else "",
            status=_str(data.get("status")),
            dir=_str(data.get("dir")),
            category=_str(data.get("category")),
            totalLength=_toInt(data.get("totalLength")),
            completedLength=_toInt(data.get("completedLength")),
            downloadSpeed=_toInt(data.get("downloadSpeed")),
            connections=_toInt(data.get("connections")),
            numPieces=_toInt(data.get("numPieces")),
            pieceLength=_toInt(data.get("pieceLength")),
            verifiedLength=_toInt(data.get("verifiedLength")),
            errorCode=_toInt(data.get("errorCode")),
            errorMessage=_str(data.get("errorMessage")),
            files=parsedFiles,
        )

    def asDict(self) -> dict[str, Any]:
        return {
            "gid": self.gid,
            "status": self.status,
            "dir": self.dir,
            "category": self.category,
            "totalLength": _num(self.totalLength),
            "completedLength": _num(self.completedLength),
            "downloadSpeed": _num(self.downloadSpeed),
            "connections": _num(self.connections),
            "numPieces": _num(self.numPieces),
            "pieceLength": _num(self.pieceLength),
            "verifiedLength": _num(self.verifiedLength),
            "errorCode": _num(self.errorCode),
            "errorMessage": self.errorMessage,
            "resolved": self.resolved.asDict() if self.resolved else None,
            "files": [f.asDict() for f in self.files],
        }


@dataclass(frozen=True)
class ResolvedUrl:
    """
    A downloadable resource produced from a uri

    This contains both:
      - information about what will be downloaded
      - the request context required to download it
    """

    url: str

    # User-facing information.
    title: str | None = None
    filename: str | None = None
    dir: str | None = None

    # Resource metadata.
    mimeType: str | None = None
    size: int | None = None
    category: str = "unknown"

    # HTTP request context.
    headers: dict[str, str] | None = None
    cookies: dict[str, str] | None = None

    # Optional protocol/request information.
    referer: str | None = None
    resolverName: str | None = None

    def asDict(self) -> dict[str, Any]:
        return asdict(self)

    @classmethod
    def fromDict(cls, data: dict[str, Any]) -> ResolvedUrl:
        return cls(
            url=str(data["url"]),
            title=data.get("title"),
            filename=data.get("filename"),
            dir=data.get("dir"),
            mimeType=data.get("mimeType"),
            size=data.get("size"),
            category=data.get("category", "unknown"),
            headers=data.get("headers"),
            cookies=data.get("cookies"),
            referer=data.get("referer"),
            resolverName=data.get("resolverName"),
        )
