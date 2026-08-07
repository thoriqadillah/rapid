from __future__ import annotations

import json
from typing import Any

JSONRPC_VERSION = "2.0"

# Plugin methods (line-delimited JSON-RPC over stdio).
PING = "rapid.ping"
MATCH = "rapid.match"
RESOLVE = "rapid.resolve"


class JsonRpcError(Exception):
    """Raised when a plugin replies with a JSON-RPC error or garbage."""

    def __init__(self, code: int, message: str) -> None:
        super().__init__(message)
        self.code = code
        self.message = message


def encode_request(rid: int, method: str, params: list[Any]) -> str:
    payload: dict[str, Any] = {
        "jsonrpc": JSONRPC_VERSION,
        "id": rid,
        "method": method,
        "params": params,
    }

    return json.dumps(payload, separators=(",", ":")) + "\n"


def decode_response(line: str, expected_id: int) -> Any:
    """Parse one response line; return the ``result`` or raise JsonRpcError."""
    try:
        msg = json.loads(line)
    except ValueError as exc:
        raise JsonRpcError(-32700, f"invalid json: {line!r}") from exc

    if not isinstance(msg, dict):
        raise JsonRpcError(-32600, "response not an object")

    if msg.get("id") != expected_id:
        raise JsonRpcError(-32600, "response id mismatch")

    err = msg.get("error")
    if isinstance(err, dict):
        raise JsonRpcError(int(err.get("code", -1)), str(err.get("message", err)))

    return msg.get("result")
