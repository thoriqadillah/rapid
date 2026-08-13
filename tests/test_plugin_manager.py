from __future__ import annotations

import json
import sys
from pathlib import Path

from PySide6.QtGui import QGuiApplication

from rapid.backend.plugin import PluginManager
from rapid.backend.plugin.models import PluginSpec, StdIOTransportSpec

SAMPLE = Path(__file__).resolve().parent.parent / "rapid" / "plugins"


def _app() -> QGuiApplication:
    existing = QGuiApplication.instance()
    if existing is not None and isinstance(existing, QGuiApplication):
        return existing
    return QGuiApplication(sys.argv)


def _manager(dirs: list[Path] | None = None) -> PluginManager:
    _app()
    return PluginManager(dirs if dirs is not None else [SAMPLE])


def test_discovers_sample_resolver() -> None:
    assert _manager().resolverNames == ["sampledemo"]


def test_ignores_missing_dir() -> None:
    assert _manager([SAMPLE / "does-not-exist"]).resolverNames == []


def test_resolve_supported_url() -> None:
    results = _manager().resolve("https://www.youtube.com/watch?v=dQw4w9WgXcQ")
    assert len(results) == 2
    assert all(r.resolverName == "sampledemo" for r in results)
    assert all(r.url.startswith("https://example.com/video/") for r in results)


def test_resolve_unsupported_url_returns_empty() -> None:
    assert _manager().resolve("https://example.com/") == []


def test_should_resolve_matches_scheme() -> None:
    from rapid.backend.plugin.resolver import ResolverPlugin

    spec = _manager()._plugins["sampledemo"]
    resolver = ResolverPlugin(spec=spec)
    assert resolver.shouldResolve("https://www.youtube.com/watch?v=dQw4w9WgXcQ")
    assert resolver.shouldResolve("https://youtu.be/dQw4w9WgXcQ")
    assert not resolver.shouldResolve("https://example.com/")


def test_load_spec_skips_invalid_manifest(tmp_path: Path) -> None:
    bad_manifests = [
        "not json{",
        json.dumps({"name": "x", "type": "other", "command": "c"}),
        json.dumps({"name": "x", "type": "resolver"}),
        json.dumps({"name": "x", "type": "resolver", "transport": {"command": 5}}),
    ]
    for i, body in enumerate(bad_manifests):
        manifest = tmp_path / f"p{i}" / "plugin.json"
        manifest.parent.mkdir()
        manifest.write_text(body, encoding="utf-8")
        assert PluginManager._loadSpec(manifest) is None


def test_load_spec_resolves_relative_command(tmp_path: Path) -> None:
    manifest = tmp_path / "plugin.json"
    manifest.write_text(
        json.dumps({"name": "x", "type": "resolver", "transport": {"command": "run.sh"}}),
        encoding="utf-8",
    )
    spec = PluginManager._loadSpec(manifest)
    assert spec is not None
    assert spec.id == "x"
    assert spec.version == "0.0.0"
    if isinstance(spec.transport, StdIOTransportSpec):
        assert spec.transport.command == (tmp_path / "run.sh").as_posix()


def test_load_spec_reads_id_version_and_transport(tmp_path: Path) -> None:
    manifest = tmp_path / "plugin.json"
    manifest.write_text(
        json.dumps(
            {
                "id": "abc",
                "name": "x",
                "version": "1.2.3",
                "type": "resolver",
                "transport": {"command": "run.sh"},
            }
        ),
        encoding="utf-8",
    )
    spec = PluginManager._loadSpec(manifest)
    assert spec is not None
    assert spec == PluginSpec(
        id="abc",
        version="1.2.3",
        name="x",
        transport=StdIOTransportSpec(command=(tmp_path / "run.sh").as_posix()),
    )


def test_load_spec_filters_non_string_args_and_match(tmp_path: Path) -> None:
    manifest = tmp_path / "plugin.json"
    manifest.write_text(
        json.dumps(
            {
                "name": "x",
                "type": "resolver",
                "transport": {"command": "run.sh", "args": ["-v", 5, None]},
                "match": ["https:", 5, None],
            }
        ),
        encoding="utf-8",
    )
    spec = PluginManager._loadSpec(manifest)
    assert spec == PluginSpec(
        id="x",
        version="0.0.0",
        name="x",
        transport=StdIOTransportSpec(command=str(tmp_path / "run.sh"), args=("-v",)),
        match=("https:",),
    )
