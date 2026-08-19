from __future__ import annotations

import math
import random
import time

from ..download.models import Download, DownloadFile, FileUri, ResolvedUrl
from ..download.store import DownloadStore
from .seeder import Seeder

_DL_DIR = "/home/thoriqadillah/Downloads"


def _resolved(url: str, title: str, category: str, size: int) -> ResolvedUrl:
    return ResolvedUrl(url=url, title=title, filename=title, category=category, size=size)


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


def _speed_curve(base: int, samples: int = 60, seed: int = 0) -> list[int]:
    """A realistic speed history: warm-up ramp, bursts, stalls, and jitter."""
    rng = random.Random(seed)
    curve: list[int] = []
    for i in range(samples):
        t = i / samples
        # Slow ramp at the start, then settle near the baseline.
        ramp = min(1.0, t * 4)
        # Long-lived bursts (accentuated sine) + short jitter.
        burst = 0.25 * math.sin(i / 9) + 0.15 * math.sin(i / 3.7)
        # Occasional stalls that drop to near zero.
        stall = 0.9 if rng.random() > 0.05 else 0.15
        jitter = rng.uniform(0.8, 1.15)
        speed = max(0, int(base * ramp * stall * (1 + burst) * jitter))
        curve.append(speed)
    return curve


def _samples() -> tuple[Download, ...]:
    return (
        Download(
            gid="d6b8a91c2e5f4a7b",
            status="active",
            dir=_DL_DIR,
            category="video",
            resolved=_resolved(
                "https://example.com/movies/interstellar.mkv",
                "interstellar.mkv",
                "video",
                2_147_483_648,
            ),
            totalLength=2_147_483_648,
            completedLength=1_431_655_765,
            downloadSpeed=2_500_000,
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
            category="video",
            resolved=_resolved(
                "https://example.com/movies/the-dark-knight.mkv",
                "the-dark-knight.mkv",
                "video",
                894_784_853,
            ),
            totalLength=894_784_853,
            completedLength=894_784_853,
            downloadSpeed=0,
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
            category="audio",
            resolved=_resolved(
                "https://example.com/music/album.zip",
                "album.zip",
                "audio",
                104_857_600,
            ),
            totalLength=104_857_600,
            completedLength=31_457_280,
            downloadSpeed=0,
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
            category="compressed",
            resolved=_resolved(
                "https://example.com/archives/linux-kernel-6.12.tar.xz",
                "linux-kernel-6.12.tar.xz",
                "compressed",
                4_294_967_296,
            ),
            totalLength=4_294_967_296,
            completedLength=3_221_225_472,
            downloadSpeed=8_400_000,
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
            category="document",
            resolved=_resolved(
                "https://example.com/docs/manual.pdf",
                "manual.pdf",
                "document",
                5_242_880,
            ),
            totalLength=5_242_880,
            completedLength=1_048_576,
            downloadSpeed=0,
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
            category="application",
            resolved=_resolved(
                "https://example.com/software/setup.bin",
                "setup.bin",
                "application",
                734_003_200,
            ),
            totalLength=734_003_200,
            completedLength=734_003_200,
            downloadSpeed=0,
            connections=4,
            files=(
                _file(0, "software/setup.bin", 734_003_200, 734_003_200),
            ),
        ),
        Download(
            gid="f1e2d3c4b5a60719",
            status="waiting",
            dir=_DL_DIR,
            category="document",
            resolved=_resolved(
                "https://example.com/books/clean-code.pdf",
                "clean-code.pdf",
                "document",
                25_165_824,
            ),
            totalLength=25_165_824,
            completedLength=0,
            downloadSpeed=0,
            connections=0,
            files=(
                _file(0, "books/clean-code.pdf", 25_165_824),
            ),
        ),
    )


class DownloadSeeder(Seeder):
    def __init__(self, store: DownloadStore) -> None:
        self._store = store

    def seed(self) -> None:
        for i, download in enumerate(_samples()):
            self._store.upsert(download)
            if download.status == "active":
                self._seedSpeedHistory(
                    download.gid,
                    download.downloadSpeed or 1_000_000,
                    seed=i,
                )

    def _seedSpeedHistory(self, gid: str, base: int, seed: int) -> None:
        now_ms = int(time.time() * 1000)
        for i, speed in enumerate(_speed_curve(base, seed=seed)):
            ts = now_ms - 60_000 + i * 1_000
            self._store.addSpeedSample(gid, ts, speed)