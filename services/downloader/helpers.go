package downloader

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rapid-downloader/rapid/api"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func randomId(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)

	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}

func handleDuplicate(filename string) string {
	name := filename
	if file, _ := os.Stat(filename); file == nil {
		return name
	}

	regex, err := regexp.Compile(`\((.*?)\)`)
	if err != nil { // if there is no number prefix
		return name
	}

	ext := filepath.Ext(name)
	prefix := regex.FindStringSubmatch(name)
	if len(prefix) == 0 {
		// add number before ext of a file if there is none
		name = strings.ReplaceAll(name, ext, fmt.Sprint(" (1)", ext))

		// re-check if the current name has duplication
		name = handleDuplicate(name)
		return name
	}

	// if it's still has, add the number
	name = strings.ReplaceAll(name, " "+prefix[0]+ext, "")
	number, err := strconv.Atoi(prefix[1])
	if err != nil {
		return name
	}

	name = fmt.Sprintf("%s (%d)%s", name, number+1, ext)

	// re-check if the current name has duplication
	name = handleDuplicate(name)

	return name
}

func resumable(r *http.Response) bool {
	acceptRanges := r.Header.Get("Accept-Ranges")
	return acceptRanges != "" || acceptRanges == "bytes"
}

func filename(r *http.Response) string {
	disposition := r.Header.Get("Content-Disposition")
	_, params, _ := mime.ParseMediaType(disposition)

	filename, ok := params["filename"]
	if ok {
		return filename
	}

	urlPath := r.Request.URL.Path
	if i := strings.LastIndex(urlPath, "/"); i != -1 {
		return urlPath[i+1:]
	}

	return "file"
}

// calculatePartition calculates how many chunks will be for certain size
func calculatePartition(size int64, setting api.Setting) int {
	if size < setting.MinChunkSize {
		return 1
	}

	total := math.Log10(float64(size / (1024 * 1024)))
	partsize := setting.MinChunkSize

	// dampening the total partition based on digit figures, e.g 100 -> 3 digits
	for i := 0; i < int(total); i++ {
		partsize *= int64(total)
	}

	return int(size / partsize)
}

func createRequest(ctx context.Context, urlFile string, headers http.Header, cookies []*http.Cookie) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlFile, nil)
	if err != nil {
		return nil, err
	}

	url, err := url.Parse(urlFile)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", fmt.Sprintf("%s://%s", url.Scheme, url.Host))
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	for k, h := range headers {
		for _, v := range h {
			req.Header.Add(k, v)
		}
	}

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	return req, nil
}
