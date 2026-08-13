from PySide6.QtCore import QProcess
from abc import ABC, abstractmethod

class TransportError(Exception):
    """Raised when a transport cannot send or receive a request."""

class Transport(ABC):
    @abstractmethod
    def request(self, payload: str) -> str:
        pass

class ProcessTransport(Transport):
    """
    Runs a resolver plugin as a short-lived process.

    The process is started for every request and terminated afterwards.
    Communication happens over stdin/stdout using line-delimited payloads.
    """

    def __init__(
        self,
        *,
        command: str,
        args: list[str] | tuple[str, ...] = (),
        startTimeoutMs: int = 3000,
        requestTimeoutMs: int = 8000,
        writeTimeoutMs: int = 2000,
        terminateTimeoutMs: int = 2000,
    ) -> None:
        self._command = command
        self._args = list(args)
        self._startTimeoutMs = startTimeoutMs
        self._requestTimeoutMs = requestTimeoutMs
        self._writeTimeoutMs = writeTimeoutMs
        self._terminateTimeoutMs = terminateTimeoutMs

    def request(self, payload: str) -> str:
        proc = QProcess()
        proc.setProcessChannelMode(
            QProcess.ProcessChannelMode.SeparateChannels
        )

        proc.start(self._command, self._args)
        if not proc.waitForStarted(self._startTimeoutMs):
            raise TransportError(f"Plugin did not start: {self._command}")

        try:
            self._write(proc, payload)
            return self._readLine(proc)
        finally:
            self._teardown(proc)

    def _write(self, proc: QProcess, payload: str) -> None:
        data = payload.encode("utf-8")

        if proc.write(data) == -1:
            raise TransportError("Failed to write to plugin stdin")

        if not proc.waitForBytesWritten(self._writeTimeoutMs):
            raise TransportError("Plugin stdin write timeout")

    def _readLine(self, proc: QProcess) -> str:
        buffer = bytearray()
        deadline = self._deadline(self._requestTimeoutMs)

        while self._remainingMs(deadline) > 0:
            if proc.waitForReadyRead(50):
                buffer += bytes(proc.readAllStandardOutput())

                if b"\n" in buffer:
                    raw, _ = buffer.split(b"\n", 1)

                    return raw.decode("utf-8", "replace")

            elif proc.state() == QProcess.ProcessState.NotRunning:
                # The process exited before producing a complete response.
                break

        if not buffer:
            raise TransportError("Plugin returned nothing")

        raise TransportError("Plugin response timeout")

    @staticmethod
    def _deadline(timeoutMs: int) -> float:
        import time

        return time.monotonic() + timeoutMs / 1000

    @staticmethod
    def _remainingMs(deadline: float) -> int:
        import time

        remaining = deadline - time.monotonic()
        return max(0, int(remaining * 1000))

    def _teardown(self, proc: QProcess) -> None:
        if proc.state() == QProcess.ProcessState.NotRunning:
            return

        proc.terminate()
        if proc.waitForFinished(self._terminateTimeoutMs):
            return

        proc.kill()
        proc.waitForFinished(self._terminateTimeoutMs)
