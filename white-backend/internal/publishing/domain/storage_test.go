package domain

import "testing"

// Smallest real files of each kind: headers only, enough for the sniffer.
var (
	pngBytes  = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0x0d, 'I', 'H', 'D', 'R'}
	jpegBytes = []byte{0xff, 0xd8, 0xff, 0xe0, 0, 0x10, 'J', 'F', 'I', 'F', 0}
	gifBytes  = []byte("GIF89a\x01\x00\x01\x00")
	webpBytes = []byte("RIFF\x24\x00\x00\x00WEBPVP8 ")
	mp4Bytes  = []byte{0, 0, 0, 0x18, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0, 0, 0, 0}
	movBytes  = []byte{0, 0, 0, 0x14, 'f', 't', 'y', 'p', 'q', 't', ' ', ' ', 0, 0, 0, 0}
)

func TestSniffContentType(t *testing.T) {
	cases := map[string][]byte{
		"image/png":       pngBytes,
		"image/jpeg":      jpegBytes,
		"image/gif":       gifBytes,
		"image/webp":      webpBytes,
		"video/mp4":       mp4Bytes,
		"video/quicktime": movBytes,
	}
	for want, b := range cases {
		if got := SniffContentType(b); got != want {
			t.Errorf("%s: got %q", want, got)
		}
	}
	for name, b := range map[string][]byte{
		"html as png": []byte("<!doctype html><html><script>alert(1)</script></html>"),
		"svg as jpg":  []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"empty":       {},
		"short":       {0xff, 0xd8},
		"pdf":         []byte("%PDF-1.7"),
	} {
		if got := SniffContentType(b); got != "" {
			t.Errorf("%s: accepted as %q", name, got)
		}
	}
}

func TestKeysAndExtensions(t *testing.T) {
	if !IsSafeKey("client_1/0199-abc.png") || IsSafeKey("../etc/passwd") || IsSafeKey("a/b/c.png") || IsSafeKey("a/b c.png") || IsSafeKey("a") {
		t.Fatal("safe key")
	}
	if ExtensionFor("image/jpeg") != "jpg" || ExtensionFor("image/svg+xml") != "" || ContentTypeForExtension("mov") != "video/quicktime" || ContentTypeForExtension("bin") != "application/octet-stream" {
		t.Fatal("extensions")
	}
	url := PublicMediaURL("https://app.mixist.id/", "c1/x.png")
	if url != "https://app.mixist.id/api/media/c1/x.png" || KeyFromURL(url) != "c1/x.png" || KeyFromURL("https://cdn.example.com/a.png") != "" {
		t.Fatal(url)
	}
	if KindForMime("video/mp4") != MediaVideo || KindForMime("image/png") != MediaImage {
		t.Fatal("kind")
	}
}

func TestExternalMedia(t *testing.T) {
	if ExternalFilename("/img/Kopi%20Promo.PNG") != "Kopi%20Promo.PNG" || ExternalFilename("/") != "media" || ExternalFilename("") != "media" {
		t.Fatal(ExternalFilename("/img/Kopi%20Promo.PNG"))
	}
	if ExternalMime(MediaVideo, "clip.MOV") != "video/quicktime" || ExternalMime(MediaVideo, "clip") != "video/mp4" {
		t.Fatal("video mime")
	}
	if ExternalMime(MediaImage, "a.png") != "image/png" || ExternalMime(MediaImage, "a.webp") != "image/webp" || ExternalMime(MediaImage, "a.gif") != "image/gif" || ExternalMime(MediaImage, "a.bmp") != "image/jpeg" {
		t.Fatal("image mime")
	}
}
