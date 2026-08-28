from __future__ import annotations

import json
from http.client import HTTPConnection

import pytest
from PySide6.QtGui import QGuiApplication

from rapid.backend.browser_integration import (
    BrowserIntegration,
    BrowserRequestError,
    normalizeBrowserRequest,
)


def test_normalize_browser_request_preserves_download_context() -> None:
    request = normalizeBrowserRequest({
        "url": "https://cdn.example/video.mp4",
        "pageUrl": "https://example/watch/1",
        "referer": "https://example/watch/1",
        "title": "Example video",
        "headers": {
            "Authorization": "Bearer token",
            "Origin": "https://example",
            "Host": "forged.example",
            "Content-Length": "42",
        },
        "cookies": {"session": "secret"},
    })

    assert request == {
        "url": "https://cdn.example/video.mp4",
        "pageUrl": "https://example/watch/1",
        "referer": "https://example/watch/1",
        "title": "Example video",
        "headers": {
            "Authorization": "Bearer token",
            "Origin": "https://example",
        },
        "cookies": {"session": "secret"},
    }


@pytest.mark.parametrize(
    "origin",
    ["chrome-extension://extension-id", "moz-extension://extension-id"],
)
def test_bridge_accepts_extension_preflight_and_dispatches_request(origin: str) -> None:
    app = QGuiApplication.instance() or QGuiApplication([])
    plugin = BrowserIntegration(port=0)
    received: list[dict[str, object]] = []
    plugin.downloadRequested.connect(received.append)
    plugin.start()
    host, port = plugin.address
    connection = HTTPConnection(host, port, timeout=2)
    headers = {
        "Origin": origin,
        "Access-Control-Request-Headers": "content-type, x-rapid-extension",
        "Access-Control-Request-Method": "POST",
    }
    try:
        connection.request("OPTIONS", "/downloads", headers=headers)
        assert connection.getresponse().status == 204
        connection.request(
            "POST",
            "/downloads",
            body=json.dumps({"url": "https://example.com/video.mp4"}),
            headers={
                "Origin": origin,
                "Content-Type": "application/json",
                "X-Rapid-Extension": "1",
            },
        )
        assert connection.getresponse().status == 202
        app.processEvents()
    finally:
        connection.close()
        plugin.close()

    assert received[0]["url"] == "https://example.com/video.mp4"


def test_normalize_browser_request_accepts_preresolved_metadata() -> None:
    request = normalizeBrowserRequest({
        "url": "https://example.com/video.mp4",
        "browserResolved": True,
        "size": 2048,
        "category": "video",
    })

    assert request["browserResolved"] is True
    assert request["size"] == 2048
    assert request["category"] == "video"


def test_normalize_browser_request_rejects_non_downloadable_urls() -> None:
    with pytest.raises(BrowserRequestError, match="downloadable"):
        normalizeBrowserRequest({"url": "blob:https://example/id"})
