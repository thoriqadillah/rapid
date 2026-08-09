from __future__ import annotations

from dataclasses import dataclass, asdict
from pathlib import Path
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


_VIDEO = {"mp4", "mkv", "webm", "avi", "mov", "flv", "m4v", "ts", "3gp"}
_AUDIO = {"mp3", "m4a", "aac", "flac", "wav", "ogg", "opus", "wma"}
_IMAGE = {"jpg", "jpeg", "png", "gif", "webp", "bmp", "svg", "heic", "avif"}


def _infer_kind(files: tuple[DownloadFile, ...]) -> str:
    for file in files:
        if file.path:
            ext = Path(file.path).suffix.lower().lstrip(".")
            if ext in _VIDEO:
                return "video"
            if ext in _AUDIO:
                return "audio"
            if ext in _IMAGE:
                return "image"
    return "other"


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
    kind: str | None = None
    totalLength: int | None = None
    completedLength: int | None = None
    uploadLength: int | None = None
    downloadSpeed: int | None = None
    uploadSpeed: int | None = None
    connections: int | None = None
    numPieces: int | None = None
    pieceLength: int | None = None
    verifiedLength: int | None = None
    errorCode: int | None = None
    errorMessage: str | None = None
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
            tuple(DownloadFile.from_payload(f) for f in files if isinstance(f, dict))
            if isinstance(files, list)
            else ()
        )
        return cls(
            gid=str(gid) if gid is not None else "",
            status=_str(data.get("status")),
            dir=_str(data.get("dir")),
            kind=_infer_kind(parsedFiles),
            totalLength=_to_int(data.get("totalLength")),
            completedLength=_to_int(data.get("completedLength")),
            uploadLength=_to_int(data.get("uploadLength")),
            downloadSpeed=_to_int(data.get("downloadSpeed")),
            uploadSpeed=_to_int(data.get("uploadSpeed")),
            connections=_to_int(data.get("connections")),
            numPieces=_to_int(data.get("numPieces")),
            pieceLength=_to_int(data.get("pieceLength")),
            verifiedLength=_to_int(data.get("verifiedLength")),
            errorCode=_to_int(data.get("errorCode")),
            errorMessage=_str(data.get("errorMessage")),
            files=parsedFiles,
        )

    def asDict(self) -> dict[str, Any]:
        payload: dict[str, Any] = {
            "gid": self.gid,
            "status": self.status,
            "dir": self.dir,
            "kind": self.kind,
            "totalLength": _num(self.totalLength),
            "completedLength": _num(self.completedLength),
            "uploadLength": _num(self.uploadLength),
            "downloadSpeed": _num(self.downloadSpeed),
            "uploadSpeed": _num(self.uploadSpeed),
            "connections": _num(self.connections),
            "numPieces": _num(self.numPieces),
            "pieceLength": _num(self.pieceLength),
            "verifiedLength": _num(self.verifiedLength),
            "errorCode": _num(self.errorCode),
            "errorMessage": self.errorMessage,
            "files": [f.as_dict() for f in self.files],
        }
        return payload


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

    # Resource metadata.
    mimeType: str | None = None
    size: int | None = None
    kind: str = "unknown"

    # HTTP request context.
    headers: dict[str, str] | None = None
    cookies: dict[str, str] | None = None

    # Optional protocol/request information.
    referer: str | None = None

    def asDict(self) -> dict[str, Any]:
        return asdict(self)

    @classmethod
    def fromDict(
        cls,
        data: dict[str, Any],
    ) -> ResolvedUrl:
        return cls(
            url=str(data["url"]),
            title=data.get("title"),
            filename=data.get("filename"),
            mimeType=data.get("mime_type"),
            size=data.get("size"),
            kind=data.get("kind", "unknown"),
            headers=dict(
                data.get("headers") or {},
            ),
            cookies=dict(
                data.get("cookies") or {},
            ),
            referer=data.get("referer"),
        )
