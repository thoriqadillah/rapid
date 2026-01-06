package downloader

import (
	"testing"
)

func TestFiletypeDocument(t *testing.T) {
	testCases := []string{
		"document.doc",
		"document.docx",
		"document.pdf",
		"document.txt",
		"presentation.ppt",
		"presentation.pptx",
		"spreadsheet.xls",
		"spreadsheet.xlsx",
		"document.odt",
		"document.ods",
		"document.odp",
		"document.odg",
		"document.odf",
		"document.rtf",
		"document.tex",
		"document.texi",
		"document.texinfo",
		"document.wpd",
		"document.wps",
		"document.wpg",
		"document.wks",
		"document.wqd",
		"document.wqx",
		"document.w",
	}

	for _, name := range testCases {
		t.Run(name, func(t *testing.T) {
			result := filetype(name)
			if result != "Document" {
				t.Errorf("Expected filetype(%s) to be %v, but got %v", name, "Document", result)
			}
		})
	}
}

func TestFiletypeImage(t *testing.T) {
	testCases := []string{
		"image.jpg",
		"image.jpeg",
		"image.png",
		"image.gif",
		"image.svg",
		"image.bmp",
		"picture.JPG",  // Test case with different case
		"my_image.png", // Test case with a different name
	}

	for _, filename := range testCases {
		t.Run(filename, func(t *testing.T) {
			result := filetype(filename)
			if result != "Image" {
				t.Errorf("Expected filetype(%s) to be Image, but got %s", filename, result)
			}
		})
	}
}

func TestFiletypeVideo(t *testing.T) {
	testCases := []string{
		"video.mp4",
		"movie.mov",
		"clip.avi",
		"film.mkv",
		"music_video.flv",
		"webinar.webm",
		"mpeg_video.mpeg",
		"short_clip.mp4",
		"audio.m4a",
	}

	for _, filename := range testCases {
		t.Run(filename, func(t *testing.T) {
			result := filetype(filename)
			if result != "Video" {
				t.Errorf("Expected filetype(%s) to be Video, but got %s", filename, result)
			}
		})
	}
}

func TestFiletypeAudio(t *testing.T) {
	testCases := []string{
		"song.mp3",
		"audio.wav",
		"music.flac",
		"voice.aac",
		"podcast.ogg",
		"audio_book.opus",
	}

	for _, filename := range testCases {
		t.Run(filename, func(t *testing.T) {
			result := filetype(filename)
			if result != "Audio" {
				t.Errorf("Expected filetype(%s) to be Audio, but got %s", filename, result)
			}
		})
	}
}

func TestFiletypeCompressed(t *testing.T) {
	testCases := []string{
		"archive.zip",
		"compressed.rar",
		"compressed_file.7z",
		"tarball.tar",
		"gzip_file.gz",
		"bzip2_file.bz2",
		"tar_gzip_file.tgz",
		"tar_bzip2_file.tbz2",
		"xz_file.xz",
		"txz_file.txz",
		"zst_file.zst",
		"zstd_file.zstd",
	}

	for _, filename := range testCases {
		t.Run(filename, func(t *testing.T) {
			result := filetype(filename)
			if result != "Compressed" {
				t.Errorf("Expected filetype(%s) to be Compressed, but got %s", filename, result)
			}
		})
	}
}

func TestFiletypeOther(t *testing.T) {
	testCases := []string{
		"archive.psd",
		"compressed.xxx",
		"compressed_file.xyz",
		"tarball.aaa",
	}

	for _, filename := range testCases {
		t.Run(filename, func(t *testing.T) {
			result := filetype(filename)
			if result != "Other" {
				t.Errorf("Expected filetype(%s) to be Other, but got %s", filename, result)
			}
		})
	}
}
