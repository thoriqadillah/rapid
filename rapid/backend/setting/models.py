from dataclasses import dataclass, field
from pathlib import Path

from PySide6.QtCore import QStandardPaths

@dataclass(frozen=True)
class Settings:
    dataDir: Path
    downloadDir: Path
    pluginDirs: list[Path]
    baseDir: Path
    aria2Host: str = "127.0.0.1"
    aria2Port: int = 6800
    aria2Token: str = ""
    aria2SessionFile: str = ""
    aria2SaveSessionInterval: int = 1
    pollIntervalMs: int = 1000

    @staticmethod
    def default(baseDir: Path) -> "Settings":
        appData = Path(QStandardPaths.writableLocation(QStandardPaths.StandardLocation.AppDataLocation))
        appData.mkdir(parents=True, exist_ok=True)

        downloadDir = Path(QStandardPaths.writableLocation(QStandardPaths.StandardLocation.DownloadLocation))
        sessionFile = appData / "aria2.session"
        sessionFile.touch(exist_ok=True)

        return Settings(
            dataDir=appData,
            downloadDir=downloadDir,
            pluginDirs=[baseDir / "plugins", appData / "plugins"],
            baseDir=baseDir,
            aria2Host="127.0.0.1",
            aria2Port=6800,
            aria2Token="",
            aria2SessionFile=str(sessionFile),
            aria2SaveSessionInterval=1,
            pollIntervalMs=1000,
        )
