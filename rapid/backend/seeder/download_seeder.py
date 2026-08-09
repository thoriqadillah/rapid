from __future__ import annotations

import math
import time

from ..download.models import Download, DownloadFile, FileUri
from ..download.store import DownloadStore
from .seeder import Seeder

_DL_DIR = "/home/thoriqadillah/Downloads"


def _file(
    index: int,
    path: str,
    length: int,
    completed: int | None = None,
    selected: bool = True,
) -> DownloadFile:
    return DownloadFile(
        index=index,
        path=path,
        length=length,
        completed_length=completed,
        selected=selected,
        uris=(FileUri(uri=f"https://mirror.example.com/{path}", status="used"),),
    )


def _samples() -> tuple[Download, ...]:
    return (
        Download(
            gid="d6b8a91c2e5f4a7b",
            status="active",
            dir=_DL_DIR,
            kind="video",
            totalLength=2_147_483_648,
            completedLength=1_431_655_765,
            downloadSpeed=2_500_000,
            uploadSpeed=1_024,
            connections=8,
            numPieces=2048,
            pieceLength=1_048_576,
            verifiedLength=1_431_655_765,
            files=(
                _file(0, "movies/interstellar.mkv", 2_147_483_648, 1_431_655_765),
                _file(1, "movies/interstellar.srt", 45_678, 45_678),
            ),
        ),
        Download(
            gid="9c3d7e1a4b5f6c2d",
            status="complete",
            dir=_DL_DIR,
            kind="video",
            totalLength=894_784_853,
            completedLength=894_784_853,
            downloadSpeed=0,
            uploadSpeed=0,
            connections=0,
            numPieces=853,
            pieceLength=1_048_576,
            verifiedLength=894_784_853,
            files=(
                _file(0, "movies/the-dark-knight.mkv", 894_784_853, 894_784_853),
            ),
        ),
        Download(
            gid="5f0a2b8c3d1e4f6a",
            status="paused",
            dir=_DL_DIR,
            kind="audio",
            totalLength=104_857_600,
            completedLength=31_457_280,
            downloadSpeed=0,
            uploadSpeed=0,
            connections=4,
            numPieces=100,
            pieceLength=1_048_576,
            verifiedLength=31_457_280,
            files=(
                _file(0, "music/album.zip", 104_857_600, 31_457_280),
            ),
        ),
        Download(
            gid="a1b2c3d4e5f60718",
            status="active",
            dir=_DL_DIR,
            kind="compressed",
            totalLength=4_294_967_296,
            completedLength=3_221_225_472,
            downloadSpeed=8_400_000,
            uploadSpeed=2_100_000,
            connections=24,
            numPieces=4096,
            pieceLength=1_048_576,
            verifiedLength=3_221_225_472,
            files=(
                _file(0, "archives/linux-kernel-6.12.tar.xz", 4_294_967_296, 3_221_225_472),
            ),
        ),
        Download(
            gid="e7f8091a2b3c4d5e",
            status="error",
            dir=_DL_DIR,
            kind="other",
            totalLength=5_242_880,
            completedLength=1_048_576,
            downloadSpeed=0,
            uploadSpeed=0,
            connections=0,
            errorCode=1,
            errorMessage="URI not found",
            files=(
                _file(0, "docs/manual.pdf", 5_242_880, 1_048_576),
            ),
        ),
        Download(
            gid="1122334455667788",
            status="complete",
            dir=_DL_DIR,
            kind="application",
            totalLength=734_003_200,
            completedLength=734_003_200,
            downloadSpeed=0,
            uploadSpeed=524_288,
            connections=4,
            files=(
                _file(0, "software/setup.bin", 734_003_200, 734_003_200),
            ),
        ),
    )


class DownloadSeeder(Seeder):
    def __init__(self, store: DownloadStore) -> None:
        self._store = store

    def seed(self) -> None:
        for download in _samples():
            self._store.upsert(download)
            if download.status == "active":
                self._seedSpeedHistory(download.gid, download.downloadSpeed or 1_000_000)

    def _seedSpeedHistory(self, gid: str, base: int) -> None:
        now = int(time.time())
        for i in range(60):
            ts = now - 60 + i
            speed = max(0, int(base * (0.8 + 0.4 * math.sin(i / 8))))
            self._store.addSpeedSample(gid, ts, speed)
