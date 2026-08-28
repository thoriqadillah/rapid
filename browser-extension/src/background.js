const RAPID_URL = "http://127.0.0.1:17654";
const HEADER_TTL_MS = 10 * 60 * 1000;
const requestHeadersByUrl = new Map();
const responseMetadataByUrl = new Map();
const videoUrlsByTab = new Map();
const blockedHeaders = new Set([
    "connection",
    "content-length",
    "cookie",
    "host",
    "proxy-connection",
]);

function callChrome(fn, ...args) {
    return new Promise((resolve, reject) => {
        fn(...args, (result) => {
            const error = chrome.runtime.lastError;
            if (error) reject(new Error(error.message));
            else resolve(result);
        });
    });
}

function removeExpiredEntries(entries) {
    for (const [url, entry] of entries) {
        if (Date.now() - entry.timestamp > HEADER_TTL_MS) entries.delete(url);
    }
}

function cookiesFromHeader(value) {
    const cookies = {};
    for (const pair of String(value || "").split(";")) {
        const index = pair.indexOf("=");
        if (index > 0) cookies[pair.slice(0, index).trim()] = pair.slice(index + 1).trim();
    }
    return cookies;
}

function rememberHeaders(details) {
    if (!details.url || details.url.startsWith(RAPID_URL)) return;
    const headers = {};
    const cookies = {};
    for (const header of details.requestHeaders || []) {
        if (!header.name) continue;
        const name = header.name.toLowerCase();
        if (name === "cookie") {
            Object.assign(cookies, cookiesFromHeader(header.value));
            continue;
        }
        if (blockedHeaders.has(name)) continue;
        if (typeof header.value === "string") headers[header.name] = header.value;
    }

    requestHeadersByUrl.set(details.url, { headers, cookies, timestamp: Date.now() });
    removeExpiredEntries(requestHeadersByUrl);
}

function filenameFromDisposition(disposition) {
    if (!disposition) return "";
    const encoded = disposition.match(/filename\*\s*=\s*(?:UTF-8'')?([^;]+)/i);
    const regular = disposition.match(/filename\s*=\s*(?:"([^"]+)"|([^;]+))/i);
    const value = (encoded ? encoded[1] : regular ? regular[1] || regular[2] : "")
        .trim()
        .replace(/^"|"$/g, "");

    if (!value) return "";
    try {
        return decodeURIComponent(value).split(/[\\/]/).pop();
    } catch {
        return value.split(/[\\/]/).pop();
    }
}

function rememberResponseMetadata(details) {
    if (!details.url || details.url.startsWith(RAPID_URL)) return;
    const headers = {};
    for (const header of details.responseHeaders || []) {
        if (header.name && typeof header.value === "string") {
            headers[header.name.toLowerCase()] = header.value;
        }
    }

    const rawSize = Number.parseInt(headers["content-length"] || "", 10);
    responseMetadataByUrl.set(details.url, {
        filename: filenameFromDisposition(headers["content-disposition"]),
        mimeType: (headers["content-type"] || "").split(";", 1)[0].trim(),
        size: Number.isSafeInteger(rawSize) && rawSize >= 0 ? rawSize : undefined,
        timestamp: Date.now(),
    });
    removeExpiredEntries(responseMetadataByUrl);
}

function responseMetadataFor(...urls) {
    for (const url of urls) {
        const metadata = responseMetadataByUrl.get(url);
        if (metadata) return metadata;
    }
    return {};
}

const requestFilter = { urls: ["<all_urls>"] };
try {
    chrome.webRequest.onBeforeSendHeaders.addListener(
        rememberHeaders,
        requestFilter,
        ["requestHeaders", "extraHeaders"],
    );
} catch {
    chrome.webRequest.onBeforeSendHeaders.addListener(
        rememberHeaders,
        requestFilter,
        ["requestHeaders"],
    );
}

try {
    chrome.webRequest.onHeadersReceived.addListener(
        rememberResponseMetadata,
        requestFilter,
        ["responseHeaders", "extraHeaders"],
    );
} catch {
    chrome.webRequest.onHeadersReceived.addListener(
        rememberResponseMetadata,
        requestFilter,
        ["responseHeaders"],
    );
}

async function cookiesFor(url) {
    const cookies = await callChrome(chrome.cookies.getAll.bind(chrome.cookies), { url })
        .catch(() => []);

    const result = {};
    for (const cookie of cookies || []) result[cookie.name] = cookie.value;
    return result;
}

function filenameFromUrl(url) {
    try {
        const part = new URL(url).pathname.split("/").filter(Boolean).pop();
        return part ? decodeURIComponent(part) : "";
    } catch {
        return "";
    }
}

const IMAGES = [
    "png",
    "jpg",
    "jpeg",
    "gif",
    "bmp",
    "svg",
    "webp",
    "ico",
    "tiff",
    "avif",
];

const VIDEOS = [
    "mp4",
    "mkv",
    "webm",
    "avi",
    "mov",
    "m4v",
    "flv",
    "wmv",
    "mpg",
    "mpeg",
    "3gp",
];

const AUDIOS = [
    "mp3",
    "wav",
    "flac",
    "ogg",
    "m4a",
    "aac",
    "wma",
    "opus",
];


function categoryFor(mimeType, filename, url) {
    const prefix = (mimeType || "").split("/", 1)[0];
    if (["audio", "video", "image"].includes(prefix)) return prefix;
    const name = (filename || filenameFromUrl(url))
        .split(/[?#]/, 1)[0]
        .toLowerCase();

    const extension = name.includes(".") ? name.split(".").pop() : "";
    if (IMAGES.includes(extension)) return "image";
    if (VIDEOS.includes(extension)) return "video";
    if (AUDIOS.includes(extension)) return "audio";
    return "unknown";
}

async function activeTab() {
    const tabs = await callChrome(chrome.tabs.query.bind(chrome.tabs), {
        active: true,
        currentWindow: true,
    });

    return tabs && tabs[0] ? tabs[0] : {};
}

async function requestFor(candidate, tab) {
    const url = candidate.url;
    const cached = requestHeadersByUrl.get(url);
    const response = responseMetadataFor(url);
    const filename = response.filename || candidate.filename || filenameFromUrl(url);
    const mimeType = response.mimeType || candidate.mimeType || "";
    const size =
        Number.isInteger(response.size) && response.size >= 0
            ? response.size
            : Number.isInteger(candidate.size) && candidate.size >= 0
              ? candidate.size
              : undefined;
    const category =
        candidate.category && candidate.category !== "unknown"
            ? candidate.category
            : categoryFor(mimeType, filename, url);
    return {
        url,
        pageUrl: candidate.pageUrl || tab.url || "",
        referer: candidate.referer || candidate.pageUrl || tab.url || "",
        title: candidate.title || filename || tab.title || filenameFromUrl(url),
        filename,
        mimeType,
        size,
        category,
        browserResolved: candidate.browserResolved === true,
        source: candidate.source || "browser-extension",
        headers: cached ? cached.headers : {},
        cookies:
            cached && Object.keys(cached.cookies).length
                ? cached.cookies
                : await cookiesFor(url),
    };
}

async function sendToRapid(candidate, tab = {}) {
    let protocol = "";
    try {
        protocol = new URL(candidate && candidate.url).protocol;
    } catch { }

    if (!["http:", "https:", "ftp:", "ftps:"].includes(protocol)) {
        throw new Error(
            "Rapid can only receive HTTP, HTTPS, FTP, or FTPS URLs",
        );
    }

    const payload = await requestFor(candidate, tab);
    const response = await fetch(`${RAPID_URL}/downloads`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
            "X-Rapid-Extension": "1",
        },
        body: JSON.stringify(payload),
    });

    const result = await response.json().catch(() => ({}));
    if (!response.ok) {
        throw new Error(
            result.error || `Rapid returned HTTP ${response.status}`,
        );
    }

    return result;
}

async function rapidHealth() {
    const response = await fetch(`${RAPID_URL}/health`, {
        headers: { "X-Rapid-Extension": "1" },
    });
    return response.ok;
}

function updateVideoBadge(tabId) {
    const frames = videoUrlsByTab.get(tabId);
    const urls = new Set();
    if (frames) {
        for (const frameUrls of frames.values()) {
            frameUrls.forEach((url) => urls.add(url));
        }
    }

    const count = urls.size;
    const text = count > 99 ? "99+" : count > 0 ? String(count) : "";
    chrome.action.setBadgeBackgroundColor({ tabId, color: "#ee7b57" });
    chrome.action.setBadgeText({ tabId, text });
    chrome.action.setTitle({
        tabId,
        title:
            count > 0
                ? `Download with Rapid — ${count} video${count === 1 ? "" : "s"} found`
                : "Download with Rapid",
    });
}

function clearVideoBadge(tabId) {
    videoUrlsByTab.delete(tabId);
    updateVideoBadge(tabId);
}

chrome.tabs.onUpdated.addListener((tabId, changeInfo) => {
    if (changeInfo.status === "loading") clearVideoBadge(tabId);
});

chrome.tabs.onRemoved.addListener((tabId) => videoUrlsByTab.delete(tabId));

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (
        message &&
        message.type === "video-count" &&
        sender.tab &&
        sender.tab.id != null
    ) {
        const frames = videoUrlsByTab.get(sender.tab.id) || new Map();
        frames.set(
            sender.frameId || 0,
            new Set(Array.isArray(message.urls) ? message.urls : []),
        );

        videoUrlsByTab.set(sender.tab.id, frames);
        updateVideoBadge(sender.tab.id);
        return false;
    }

    if (!message || !["submit", "health"].includes(message.type)) return false;
    (async () => {
        if (message.type === "health") return { ok: await rapidHealth() };
        const tab = await activeTab();
        await sendToRapid(message.candidate, tab);
        return { ok: true };
    })()
        .then(sendResponse)
        .catch((error) => sendResponse({ ok: false, error: error.message }));

    return true;
});

chrome.downloads.onCreated.addListener(async (item) => {
    if (!item.url || item.id == null || !/^https?:/.test(item.url)) return;
    const settings = await callChrome(
        chrome.storage.local.get.bind(chrome.storage.local),
        { interceptDownloads: true },
    );
    if (!settings.interceptDownloads) return;
    if (!(await rapidHealth().catch(() => false))) return;

    await callChrome(chrome.downloads.pause.bind(chrome.downloads), item.id)
        .catch(() => undefined);

    await callChrome(chrome.downloads.erase.bind(chrome.downloads), { id: item.id })
        .catch(() => undefined);

    try {
        const url = item.finalUrl || item.url;
        const response = responseMetadataFor(url, item.url);
        const browserFilename = item.filename
            ? item.filename.split(/[\\/]/).pop()
            : "";
        const filename = response.filename || browserFilename || filenameFromUrl(url);
        const mimeType = response.mimeType || item.mime || "";
        const size =
            Number.isInteger(response.size) && response.size >= 0
                ? response.size
                : Number.isInteger(item.totalBytes) && item.totalBytes >= 0
                  ? item.totalBytes
                  : undefined;

        await sendToRapid({
            url,
            pageUrl: item.referrer || "",
            referer: item.referrer || "",
            filename,
            mimeType,
            size,
            category: categoryFor(mimeType, filename, url),
            browserResolved: true,
            source: "browser-download",
        });
    } catch (error) {
        console.error(error);
    }
});
