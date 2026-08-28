# Rapid browser extension

Manifest V3 extension for Firefox and Chromium-based browsers.

Shared extension files live in `src/`. Browser-specific manifests live in `manifests/`, and `build.py` generates isolated packages in `dist/`.

Build both extensions from the repository root:

```bash
poetry run poe build:browser-integration
```

Start the Rapid desktop app before using either generated extension. Rapid listens only on `127.0.0.1:17654`.

## Install on Chrome, Chromium, Edge, or Brave

1. Open the browser's extension page (`chrome://extensions`, `edge://extensions`, or `brave://extensions`).
2. Enable **Developer mode**.
3. Choose **Load unpacked** and select the generated `browser-extension/dist/chromium` directory.
4. Pin **Rapid Download Integration** to the toolbar.

## Install on Firefox for development

1. Open `about:debugging#/runtime/this-firefox`.
2. Click **Load Temporary Add-on…**.
3. Select `browser-extension/dist/firefox/manifest.json`.
4. Pin **Rapid Download Integration** to the toolbar if Firefox does not show it automatically.

Firefox removes temporary add-ons when the browser restarts. For permanent installation on standard Firefox, package the contents of `browser-extension/dist/firefox` as a ZIP/XPI and have it signed through [addons.mozilla.org](https://addons.mozilla.org/developers/). Firefox Developer Edition and Nightly can also run unsigned extensions when extension-signing checks are disabled for development.

The extension scans each page and its accessible frames for `<video>`/`<source>` URLs, direct video links, and video-like performance resources. The Rapid toolbar icon displays the number of unique videos found in the current tab, and the popup lists those resources for downloading.

**Intercept downloads** is enabled by default. A browser download is paused first, sent to Rapid, then removed from the browser only after Rapid accepts it. If Rapid is unavailable, the browser download is resumed.

Resources already requested by the browser—intercepted downloads, active media URLs, and performance-observed media—are marked as pre-resolved. Rapid uses the browser metadata directly and skips resolver plugins, its metadata HTTP fetch, and aria2's dry-run/HEAD probe. Unvisited links, page URLs, and streaming manifests still use the normal resolver path.

## Request context

For each URL, the extension sends Rapid:

- request headers observed by `webRequest`, including authorization headers when the browser exposes them;
- cookies applicable to the exact download URL, including HTTP-only cookies available through the extension API;
- referrer/page URL, title, filename, and MIME type.

Rapid carries this context through resolver plugins and into aria2. Browsers do not expose TLS session state, client certificates, response bodies, DRM keys, in-memory JavaScript tokens, or every browser-generated `Sec-*` header. DRM/blob/MSE-only video therefore cannot be downloaded merely by copying its URL.

The local bridge rejects normal web-page origins and requires the extension marker header. It is not exposed beyond loopback.
