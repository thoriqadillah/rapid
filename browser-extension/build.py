from __future__ import annotations

import json
import shutil
from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent
SOURCE_DIR = BASE_DIR / "src"
MANIFEST_DIR = BASE_DIR / "manifests"
DIST_DIR = BASE_DIR / "dist"
TARGETS = ("chromium", "firefox")


def build() -> None:
    shutil.rmtree(DIST_DIR, ignore_errors=True)
    DIST_DIR.mkdir(parents=True)

    for target in TARGETS:
        manifest = MANIFEST_DIR / f"{target}.json"
        with manifest.open(encoding="utf-8") as file:
            json.load(file)

        output = DIST_DIR / target
        shutil.copytree(SOURCE_DIR, output)
        shutil.copy2(manifest, output / "manifest.json")
        print(f"Built {output}")


if __name__ == "__main__":
    build()
