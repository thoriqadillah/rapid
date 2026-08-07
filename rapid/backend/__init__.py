from .download import Aria2Client, DownloadFile, Download, DownloadStore, FileUri, SpeedSample
from .plugin import PluginManager, PluginSpec, ResolvedUrl

__all__ = [
    "Aria2Client",
    "DownloadStore",
    "Download",
    "DownloadFile",
    "FileUri",
    "SpeedSample",
    "PluginManager",
    "PluginSpec",
    "ResolvedUrl",
]
