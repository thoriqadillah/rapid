(() => {
    const VIDEO_URL = /\.(?:mp4|m4v|webm|mov|mkv|avi|m3u8|mpd)(?:$|[?#])/i;
    let previous = "";
    let timer = null;

    function detectedUrls() {
        const urls = new Set();
        const add = (rawUrl) => {
            if (!rawUrl) return;
            try {
                const url = new URL(rawUrl, document.baseURI).href;
                if (/^https?:/.test(url)) urls.add(url);
            } catch {}
        };

        for (const video of document.querySelectorAll("video")) {
            add(video.currentSrc || video.src);
            for (const source of video.querySelectorAll("source"))
                add(source.src);
        }

        for (const source of document.querySelectorAll("source")) {
            if ((source.type || "").startsWith("video/")) add(source.src);
        }

        for (const anchor of document.querySelectorAll("a[href]")) {
            if (VIDEO_URL.test(anchor.href)) add(anchor.href);
        }

        for (const entry of performance.getEntriesByType("resource")) {
            if (VIDEO_URL.test(entry.name)) add(entry.name);
        }

        return [...urls].sort();
    }

    function report() {
        timer = null;
        const urls = detectedUrls();
        const serialized = JSON.stringify(urls);
        if (serialized === previous) return;
        previous = serialized;
        try {
            chrome.runtime.sendMessage(
                { type: "video-count", urls },
                () => void chrome.runtime.lastError,
            );
        } catch {}
    }

    function schedule() {
        if (timer !== null) clearTimeout(timer);
        timer = setTimeout(report, 200);
    }

    const observer = new MutationObserver(schedule);
    observer.observe(document.documentElement, {
        childList: true,
        subtree: true,
        attributes: true,
        attributeFilter: ["src", "href"],
    });
    setInterval(schedule, 2000);
    schedule();
})();
