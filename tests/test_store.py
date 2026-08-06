from __future__ import annotations

from pathlib import Path

from rapid.backend.download import DownloadStore

EMPTY_PAYLOAD = {
    "gid": "abc",
    "status": None,
    "dir": None,
    "totalLength": None,
    "completedLength": None,
    "uploadLength": None,
    "downloadSpeed": None,
    "uploadSpeed": None,
    "connections": None,
    "numPieces": None,
    "pieceLength": None,
    "verifiedLength": None,
    "numSeeders": None,
    "seeder": None,
    "infoHash": None,
    "bitfield": None,
    "errorCode": None,
    "errorMessage": None,
    "files": [],
}


def _store(tmp_path: Path) -> DownloadStore:
    return DownloadStore(path=tmp_path / "downloads.db")


def test_empty_store(tmp_path: Path) -> None:
    assert _store(tmp_path).all() == {}


def test_upsert_and_get(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert("abc", {"status": "active", "gid": "abc"})
    assert store.get("abc") == {**EMPTY_PAYLOAD, "status": "active"}


def test_upsert_merges_scalar_fields(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert("abc", {"status": "active", "gid": "abc"})
    store.upsert("abc", {"downloadSpeed": "1024", "connections": "4"})
    assert store.get("abc") == {
        **EMPTY_PAYLOAD,
        "status": "active",
        "downloadSpeed": "1024",
        "connections": "4",
    }


def test_upsert_zero_speed_is_kept(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert("abc", {"downloadSpeed": "1024"})
    store.upsert("abc", {"downloadSpeed": "0"})
    result = store.get("abc")
    assert result is not None
    assert result["downloadSpeed"] == "0"


def test_persists_files_and_uris(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert(
        "abc",
        {
            "gid": "abc",
            "status": "complete",
            "totalLength": "100",
            "files": [
                {
                    "index": "1",
                    "path": "/dl/a.bin",
                    "length": "60",
                    "completedLength": "60",
                    "selected": "true",
                    "uris": [
                        {"uri": "http://x/a.bin", "status": "used"},
                        {"uri": "http://x/a.bin", "status": "waiting"},
                    ],
                }
            ],
        },
    )
    payload = store.get("abc")
    assert payload is not None
    assert payload["status"] == "complete"
    assert payload["totalLength"] == "100"
    assert payload["files"] == [
        {
            "index": "1",
            "path": "/dl/a.bin",
            "length": "60",
            "completedLength": "60",
            "selected": "true",
            "uris": [
                {"uri": "http://x/a.bin", "status": "used"},
                {"uri": "http://x/a.bin", "status": "waiting"},
            ],
        }
    ]


def test_get_missing_returns_none(tmp_path: Path) -> None:
    assert _store(tmp_path).get("nope") is None


def test_persists_across_instances(tmp_path: Path) -> None:
    path = tmp_path / "downloads.db"
    DownloadStore(path=path).upsert("abc", {"status": "active", "gid": "abc"})
    reloaded = DownloadStore(path=path)
    assert reloaded.all() == {"abc": {**EMPTY_PAYLOAD, "status": "active"}}


def test_remove(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert("abc", {"status": "active", "gid": "abc"})
    store.remove("abc")
    assert store.all() == {}


def test_clear(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert("a", {"gid": "a"})
    store.upsert("b", {"gid": "b"})
    store.clear()
    assert store.all() == {}


def test_speed_samples_roundtrip(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert("abc", {"gid": "abc"})
    store.add_speed_sample("abc", 1000, 500)
    store.add_speed_sample("abc", 2000, 700)
    store.add_speed_sample("abc", 3000, 900)
    assert store.speed_history("abc") == [
        {"ts": 1000, "speed": 500},
        {"ts": 2000, "speed": 700},
        {"ts": 3000, "speed": 900},
    ]


def test_speed_samples_limit_and_order(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert("abc", {"gid": "abc"})
    for i in range(10):
        store.add_speed_sample("abc", i, i)
    history = store.speed_history("abc", limit=3)
    assert [s["ts"] for s in history] == [7, 8, 9]


def test_speed_samples_duplicate_ts_ignored(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert("abc", {"gid": "abc"})
    store.add_speed_sample("abc", 1000, 500)
    store.add_speed_sample("abc", 1000, 999)
    assert store.speed_history("abc") == [{"ts": 1000, "speed": 500}]


def test_prune_speeds(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert("abc", {"gid": "abc"})
    store.add_speed_sample("abc", 1000, 1)
    store.add_speed_sample("abc", 2000, 2)
    store.add_speed_sample("abc", 3000, 3)
    store.prune_speeds(2000)
    assert [s["ts"] for s in store.speed_history("abc")] == [2000, 3000]


def test_remove_cascades_speed_samples(tmp_path: Path) -> None:
    store = _store(tmp_path)
    store.upsert("abc", {"gid": "abc"})
    store.add_speed_sample("abc", 1000, 1)
    store.remove("abc")
    assert store.speed_history("abc") == []
