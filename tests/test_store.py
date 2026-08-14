from __future__ import annotations

from pathlib import Path

from rapid.backend import DownloadFile, Download, DownloadStore, FileUri, SpeedSample
from rapid.backend.database.database import Database


def _store(tmp_path: Path) -> DownloadStore:
    return DownloadStore(Database(path=tmp_path / "database.db"))


def _download(gid: str, **fields) -> Download:
    return Download(gid=gid, **fields)


def test_empty_store(tmp_path: Path) -> None:
    assert _store(tmp_path).all() == {}


def test_upsert_and_get(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert(_download("abc", status="active"))
    assert store.get("abc") == Download(gid="abc", status="active")


def test_upsert_merges_scalar_fields(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert(_download("abc", status="active"))
    store.upsert(_download("abc", downloadSpeed=1024, connections=4))
    assert store.get("abc") == Download(
        gid="abc", status="active", downloadSpeed=1024, connections=4
    )


def test_upsert_zero_speed_is_kept(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert(_download("abc", downloadSpeed=1024))
    store.upsert(_download("abc", downloadSpeed=0))
    result = store.get("abc")
    assert result is not None
    assert result.downloadSpeed == 0


def test_persists_category(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert(_download("abc", category="video"))
    result = store.get("abc")
    assert result is not None
    assert result.category == "video"
    assert result.asDict()["category"] == "video"


def test_persists_files_and_uris(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert(
        _download(
            "abc",
            status="complete",
            totalLength=100,
            files=(
                DownloadFile(
                    index=1,
                    path="/dl/a.bin",
                    length=60,
                    completed_length=60,
                    selected=True,
                    uris=(
                        FileUri(uri="http://x/a.bin", status="used"),
                        FileUri(uri="http://x/a.bin", status="waiting"),
                    ),
                ),
            ),
        )
    )
    row = store.get("abc")
    assert row == Download(
        gid="abc",
        status="complete",
        totalLength=100,
        files=(
            DownloadFile(
                index=1,
                path="/dl/a.bin",
                length=60,
                completed_length=60,
                selected=True,
                uris=(
                    FileUri(uri="http://x/a.bin", status="used"),
                    FileUri(uri="http://x/a.bin", status="waiting"),
                ),
            ),
        ),
    )
    assert row.files[0].uris[1].uri == "http://x/a.bin"


def test_get_missing_returns_none(tmp_path: Path) -> None:
    assert _store(tmp_path).get("nope") is None


def test_persists_across_instances(tmp_path: Path) -> None:
    path = tmp_path / "downloads.db"
    db = Database(path=path)
    DownloadStore(db).upsert(_download("abc", status="active"))
    reloaded = DownloadStore(db)
    assert reloaded.all() == {"abc": Download(gid="abc", status="active")}


def test_remove(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert(_download("abc", status="active"))
    store.remove("abc")
    assert store.all() == {}


def test_clear(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert(_download("a"))
    store.upsert(_download("b"))
    store.clear()
    assert store.all() == {}


def test_speed_samples_roundtrip(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert(_download("abc"))
    store.addSpeedSample("abc", 1000, 500)
    store.addSpeedSample("abc", 2000, 700)
    store.addSpeedSample("abc", 3000, 900)
    assert store.speedHistory("abc") == [
        SpeedSample(ts=1000, speed=500),
        SpeedSample(ts=2000, speed=700),
        SpeedSample(ts=3000, speed=900),
    ]


def test_speed_samples_limit_and_order(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert(_download("abc"))
    for i in range(10):
        store.addSpeedSample("abc", i, i)
    history = store.speedHistory("abc", limit=3)
    assert [s.ts for s in history] == [7, 8, 9]


def test_speed_samples_duplicate_ts_ignored(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert(_download("abc"))
    store.addSpeedSample("abc", 1000, 500)
    store.addSpeedSample("abc", 1000, 999)
    assert store.speedHistory("abc") == [SpeedSample(ts=1000, speed=500)]


def test_remove_cascades_speed_samples(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert(_download("abc"))
    store.addSpeedSample("abc", 1000, 1)
    store.remove("abc")
    assert store.speedHistory("abc") == []
