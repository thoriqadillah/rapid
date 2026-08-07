#!/usr/bin/env python3
"""Demo resolver plugin: line-delimited JSON-RPC over stdio.

Speaks the rapid resolver protocol:
    rapid.ping    -> {"name": ..., "version": ..., "type": "resolver", "schemes": [...]}
    rapid.match   -> {"supported": bool}
    rapid.resolve -> {"items": [{"title", "url", "kind"}]}
"""
import json
import sys

NAME = "sampledemo"
SCHEMES = ["https://www.youtube.com/", "https://youtu.be/"]


def supported(url: str) -> bool:
    return url.startswith(tuple(SCHEMES))


def resolve(url: str) -> list[dict[str, str]]:
    video_id = url.split("v=")[1][:11] if "v=" in url else "dQw4w9WgXcQ"
    return [
        {
            "title": f"Sample video {video_id} (720p)",
            "url": f"https://example.com/video/{video_id}/720.mp4",
            "kind": "video",
        },
        {
            "title": f"Sample video {video_id} (audio)",
            "url": f"https://example.com/video/{video_id}/audio.m4a",
            "kind": "audio",
        },
    ]


def handle(method: str, params: list[object]) -> object:
    url = str(params[0]) if params else ""
    if method == "rapid.ping":
        return {"name": NAME, "version": "0.1.0", "type": "resolver", "schemes": SCHEMES}
    if method == "rapid.match":
        return {"supported": supported(url)}
    if method == "rapid.resolve":
        return {"items": resolve(url)}
    return None


def main() -> None:
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            msg = json.loads(line)
            rid = msg.get("id")
            result = handle(str(msg.get("method", "")), msg.get("params") or [])
            print(json.dumps({"jsonrpc": "2.0", "id": rid, "result": result}, separators=(",", ":")), flush=True)
        except Exception as exc:  # noqa: BLE001 - respond with an error, keep serving
            rid = None
            try:
                rid = json.loads(line).get("id")
            except Exception:  # noqa: BLE001
                pass
            print(json.dumps({"jsonrpc": "2.0", "id": rid, "error": {"code": -1, "message": str(exc)}}), flush=True)


if __name__ == "__main__":
    main()
