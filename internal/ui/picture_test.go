package ui

import (
	"errors"
	"image"
	"testing"
)

func TestPictureViewRejectsStaleKittyFrameAfterResize(t *testing.T) {
	pv := newTestPictureViewWithImage("https://example.test/a.png")

	cmd := pv.Toggle()
	if cmd == nil {
		t.Fatal("expected Kitty toggle to render cached image")
	}
	msg := cmd()

	pv.SetSize(8, 4)
	if cmd := pv.Update(msg); cmd != nil {
		t.Fatal("expected stale frame to be ignored after resize")
	}
	if pv.kittyGrid != "" {
		t.Fatalf("expected stale frame not to update grid, got %q", pv.kittyGrid)
	}
}

func TestPictureViewRejectsKittyFrameAfterLeavingKittyMode(t *testing.T) {
	pv := newTestPictureViewWithImage("https://example.test/a.png")

	cmd := pv.Toggle()
	if cmd == nil {
		t.Fatal("expected Kitty toggle to render cached image")
	}
	msg := cmd()

	pv.Toggle()
	if pv.mode != PictureGlyph {
		t.Fatalf("expected glyph mode after second toggle, got %v", pv.mode)
	}
	if cmd := pv.Update(msg); cmd != nil {
		t.Fatal("expected stale Kitty frame to be ignored after leaving Kitty mode")
	}
	if pv.mode != PictureGlyph {
		t.Fatalf("expected stale frame not to re-enter Kitty mode, got %v", pv.mode)
	}
	if pv.kittyGrid != "" {
		t.Fatalf("expected stale frame not to update grid, got %q", pv.kittyGrid)
	}
}

func TestPictureViewIgnoresStaleFetchRequest(t *testing.T) {
	pv := newPictureView()

	pv.SetURL("https://example.test/a.png")
	oldSeq := pv.renderSeq
	pv.SetURL("https://example.test/b.png")

	cmd := pv.Update(productImageFetchRequestedMsg{url: "https://example.test/a.png", seq: oldSeq})
	if cmd != nil {
		t.Fatal("expected stale fetch request to be ignored")
	}
	if pv.loading["https://example.test/a.png"] {
		t.Fatal("expected stale URL not to be marked loading")
	}
}

func TestPictureViewFallsBackToGlyphOnKittyRenderError(t *testing.T) {
	pv := newTestPictureViewWithImage("https://example.test/a.png")

	if cmd := pv.Toggle(); cmd == nil {
		t.Fatal("expected Kitty toggle to render cached image")
	}
	msg := pictureKittyFrameMsg{
		url:  pv.currentURL,
		seq:  pv.renderSeq,
		cols: pv.cols,
		rows: pv.rows,
		err:  errors.New("encode failed"),
	}

	if cmd := pv.Update(msg); cmd != nil {
		t.Fatal("expected render error to be handled without an extra command")
	}
	if pv.mode != PictureGlyph {
		t.Fatalf("expected Kitty render error to fall back to glyph mode, got %v", pv.mode)
	}
}

func TestPictureViewBoundsImageCache(t *testing.T) {
	pv := newPictureView()

	for i := 0; i < productImageCacheLimit+6; i++ {
		url := string(rune('a' + i))
		pv.currentURL = url
		pv.rememberImage(url, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	}

	if len(pv.cache) > productImageCacheLimit {
		t.Fatalf("expected cache to be bounded to %d, got %d", productImageCacheLimit, len(pv.cache))
	}
	if _, ok := pv.cache[pv.currentURL]; !ok {
		t.Fatal("expected current image to remain cached")
	}
}

func newTestPictureViewWithImage(url string) pictureView {
	pv := newPictureView()
	pv.SetSize(10, 5)
	pv.rememberImage(url, image.NewRGBA(image.Rect(0, 0, 4, 4)))
	pv.SetURL(url)
	return pv
}
