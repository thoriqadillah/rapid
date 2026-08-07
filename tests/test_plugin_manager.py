from __future__ import annotations
from pprint import pprint

import sys
from pathlib import Path

from PySide6.QtGui import QGuiApplication

from rapid.backend.plugin import PluginManager, PluginError

SAMPLE = Path(__file__).resolve().parent.parent / "rapid" / "plugins"


def _app() -> QGuiApplication:
    existing = QGuiApplication.instance()
    if existing is not None and isinstance(existing, QGuiApplication):
        return existing
    return QGuiApplication(sys.argv)


def _manager() -> PluginManager:
    _app()
    return PluginManager([SAMPLE])


def test_discovers_sample_resolver() -> None:
    assert _manager().resolver_names == ["sampledemo"]


def test_available_for_matching_url() -> None:
    manager = _manager()
    assert manager.available_for("https://www.youtube.com/watch?v=abc123") == ["sampledemo"]


def test_available_for_nonmatching_url_is_empty() -> None:
    assert _manager().available_for("https://www.tiktok.com/@user/video/1") == []


def test_resolve_returns_items() -> None:
    manager = _manager()
    items = manager.resolve("https://www.youtube.com/watch?v=xyz789", "sampledemo")
    assert isinstance(items, list) and len(items) >= 2
    assert items[0]["url"].endswith(".mp4")
    assert items[0]["kind"] == "video"


def test_wrong_resolver_is_refused() -> None:
    manager = _manager()
    try:
        manager.resolve("https://www.tiktok.com/@user/video/1", "sampledemo")
        raise AssertionError("expected PluginError for non-matching resolver")
    except PluginError:
        pass
