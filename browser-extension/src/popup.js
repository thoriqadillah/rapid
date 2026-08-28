function callChrome(fn, ...args) {
    return new Promise((resolve, reject) => {
        fn(...args, (result) => {
            const error = chrome.runtime.lastError;
            if (error) reject(new Error(error.message));
            else resolve(result);
        });
    });
}

function scanPageForVideos() {
    const found = [];
    const add = (rawUrl, data = {}) => {
        if (!rawUrl) return;
        try {
            const url = new URL(rawUrl, document.baseURI).href;
            if (!/^https?:/.test(url)) return;
            const isManifest = /\.(?:m3u8|mpd)(?:$|[?#])/i.test(url);
            found.push({
                url,
                pageUrl: location.href,
                title: data.title || document.title,
                mimeType: data.mimeType || "",
                category: "video",
                browserResolved: data.browserResolved === true && !isManifest,
                source: data.source || "page-scan",
            });
        } catch {}
    };

    for (const video of document.querySelectorAll("video")) {
        add(video.currentSrc || video.src, {
            title:
                video.title ||
                video.getAttribute("aria-label") ||
                document.title,
            mimeType: video.type,
            browserResolved: true,
            source: "video-element",
        });

        for (const source of video.querySelectorAll("source"))
            add(source.src, {
                title: video.title || document.title,
                mimeType: source.type,
                browserResolved: source.src === video.currentSrc,
                source: "video-source",
            });
    }

    for (const source of document.querySelectorAll("source")) {
        if ((source.type || "").startsWith("video/"))
            add(source.src, {
                mimeType: source.type,
                source: "source-element",
            });
    }

    for (const anchor of document.querySelectorAll("a[href]")) {
        if (
            /\.(?:mp4|m4v|webm|mov|mkv|avi|m3u8|mpd)(?:$|[?#])/i.test(
                anchor.href,
            )
        )
            add(anchor.href, {
                title: anchor.textContent.trim() || document.title,
                source: "video-link",
            });
    }

    for (const entry of performance.getEntriesByType("resource")) {
        if (
            /\.(?:mp4|m4v|webm|mov|mkv|avi|m3u8|mpd)(?:$|[?#])/i.test(
                entry.name,
            )
        )
            add(entry.name, {
                browserResolved: true,
                source: "performance-resource",
            });
    }

    return found;
}

const status = document.querySelector("#status");
const list = document.querySelector("#videos");
const connection = document.querySelector("#connection");
const intercept = document.querySelector("#intercept");

async function message(payload) {
    return callChrome(chrome.runtime.sendMessage.bind(chrome.runtime), payload);
}

function render(videos) {
    list.textContent = "";
    status.hidden = videos.length > 0;
    status.textContent = videos.length
        ? ""
        : "No downloadable videos found on this page.";

    for (const video of videos) {
        const item = document.createElement("li");
        const details = document.createElement("div");
        const title = document.createElement("div");
        const url = document.createElement("div");
        const button = document.createElement("button");
        title.className = "video-title";
        title.textContent =
            video.title || video.url.split("/").pop() || "Video";
        url.className = "video-url";
        url.textContent = video.url;
        details.append(title, url);
        button.className = "download";
        button.type = "button";
        button.textContent = "Download";
        button.addEventListener("click", async () => {
            button.disabled = true;
            button.textContent = "Sending…";
            const result = await message({
                type: "submit",
                candidate: video,
            }).catch((error) => ({ ok: false, error: error.message }));
            button.textContent = result.ok ? "Sent" : "Retry";
            button.disabled = result.ok;
            if (!result.ok) {
                status.hidden = false;
                status.textContent = result.error || "Rapid is not available.";
            }
        });
        item.append(details, button);
        list.append(item);
    }
}

async function scan() {
    status.hidden = false;
    status.textContent = "Scanning…";
    list.textContent = "";
    const tabs = await callChrome(chrome.tabs.query.bind(chrome.tabs), {
        active: true,
        currentWindow: true,
    });

    if (!tabs[0] || !tabs[0].id) return render([]);
    try {
        const frames = await callChrome(
            chrome.scripting.executeScript.bind(chrome.scripting),
            {
                target: { tabId: tabs[0].id, allFrames: true },
                func: scanPageForVideos,
            },
        );
        const unique = new Map();
        for (const frame of frames || []) {
            for (const video of frame.result || []) {
                unique.set(video.url, video);
            }
        }

        render([...unique.values()]);
    } catch {
        status.textContent = "This page cannot be scanned.";
    }
}

async function initialize() {
    const settings = await callChrome(
        chrome.storage.local.get.bind(chrome.storage.local),
        { interceptDownloads: true },
    );

    intercept.checked = settings.interceptDownloads;
    intercept.addEventListener("change", () =>
        chrome.storage.local.set({ interceptDownloads: intercept.checked }),
    );

    const health = await message({ type: "health" }).catch(() => ({
        ok: false,
    }));

    connection.textContent = health.ok
        ? "Desktop app connected"
        : "Open the Rapid desktop app";
    connection.className = health.ok ? "online" : "offline";
    await scan();
}

document.querySelector("#refresh").addEventListener("click", scan);
initialize();
