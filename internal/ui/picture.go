package ui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"  // decoder registration
	_ "image/jpeg" // decoder registration
	_ "image/png"  // decoder registration
	"io"
	"net/http"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi/kitty"
	"github.com/eliukblau/pixterm/pkg/ansimage"
)

// PictureMode selects how product images are rendered. PictureGlyph uses the
// half-block ANSI renderer (works anywhere). PictureKitty emits the bitmap
// via the Kitty graphics protocol (Kitty, Ghostty, WezTerm, etc).
type PictureMode int8

const (
	PictureGlyph PictureMode = iota
	PictureKitty
)

const (
	// productKittyImageID is the Kitty image ID used for the product picture
	// placement. Distinct from mapview's 42 so both can coexist.
	productKittyImageID = 43

	productImageFetchDelay = 150 * time.Millisecond
	productImageCacheLimit = 24
)

// productImageFetchRequestedMsg starts the actual network fetch if the
// requested URL is still current after the debounce window.
type productImageFetchRequestedMsg struct {
	url string
	seq uint64
}

// productImageLoadedMsg delivers a fetched image (or fetch error) for a URL.
type productImageLoadedMsg struct {
	url string
	img image.Image
	err error
}

// pictureKittyFrameMsg carries a Kitty render result for the product picture.
// Update routes apc via tea.Raw (the cell renderer would silently strip an
// APC embedded in the view string) and stashes grid as the view content.
type pictureKittyFrameMsg struct {
	url        string
	seq        uint64
	cols, rows int
	apc        string
	grid       string
	err        error
}

// isPictureMsg reports whether msg is a pictureView-owned async update.
// app.go uses this to forward picture results regardless of active page.
func isPictureMsg(msg tea.Msg) bool {
	switch msg.(type) {
	case productImageFetchRequestedMsg, productImageLoadedMsg, pictureKittyFrameMsg:
		return true
	}
	return false
}

// pictureView owns the product picture pane: async fetch, cache, render mode,
// and Kitty placeholder grid. ProductBrowser holds one instance and forwards
// messages/size changes/URL changes into it.
type pictureView struct {
	mode       PictureMode
	cols, rows int
	currentURL string
	cache      map[string]image.Image
	cacheOrder []string
	errs       map[string]error
	loading    map[string]bool
	renderSeq  uint64

	// kittyGrid is the most recent Kitty placeholder grid for currentURL.
	// Cleared whenever the upstream state (url, mode, size) invalidates it.
	kittyGrid string

	// glyphCache caches the rendered half-block ANSI for a given (url, cols,
	// rows). Recomputing on every View() is expensive enough to matter when
	// the user holds down arrow keys.
	glyphCache string
	glyphKey   string
}

func newPictureView() pictureView {
	return pictureView{
		mode:    PictureGlyph,
		cache:   make(map[string]image.Image),
		errs:    make(map[string]error),
		loading: make(map[string]bool),
	}
}

func (pv *pictureView) invalidateRenderedFrame() {
	pv.renderSeq++
	pv.glyphCache = ""
	pv.glyphKey = ""
	pv.kittyGrid = ""
}

// SetURL records url as the current picture. Returns a Cmd that (a) fetches
// the image if it's new, or (b) re-renders the Kitty frame if it's already
// cached. Repeat calls with the same URL are noops. New URLs are debounced so
// holding an arrow key does not start a network request for every row passed.
func (pv *pictureView) SetURL(url string) tea.Cmd {
	if url == pv.currentURL {
		return nil
	}
	pv.currentURL = url
	pv.invalidateRenderedFrame()
	if url == "" {
		return nil
	}
	if _, ok := pv.cache[url]; ok {
		return pv.renderCmd()
	}
	if _, ok := pv.errs[url]; ok {
		return nil
	}
	if pv.loading[url] {
		return nil
	}
	return requestProductImageFetch(url, pv.renderSeq)
}

// SetSize updates the target inner cell dimensions (inside the pane border).
// Returns a re-render cmd in Kitty mode.
func (pv *pictureView) SetSize(cols, rows int) tea.Cmd {
	if cols == pv.cols && rows == pv.rows {
		return nil
	}
	pv.cols = cols
	pv.rows = rows
	pv.invalidateRenderedFrame()
	return pv.renderCmd()
}

// Toggle flips the render mode. When leaving Kitty, a delete-image APC is
// emitted so the terminal drops our bitmap slot.
func (pv *pictureView) Toggle() tea.Cmd {
	prev := pv.mode
	if pv.mode == PictureGlyph {
		pv.mode = PictureKitty
	} else {
		pv.mode = PictureGlyph
	}
	pv.invalidateRenderedFrame()
	if prev == PictureKitty && pv.mode == PictureGlyph {
		return tea.Raw(kittyDeletePictureImage())
	}
	return pv.renderCmd()
}

// Mode returns the current render mode (for footer/help display).
func (pv *pictureView) Mode() PictureMode { return pv.mode }

// renderCmd returns a Cmd that builds a Kitty APC+grid frame for the current
// image. Returns nil in glyph mode or when there's nothing to render.
func (pv *pictureView) renderCmd() tea.Cmd {
	if pv.mode != PictureKitty {
		return nil
	}
	if pv.currentURL == "" || pv.cols <= 0 || pv.rows <= 0 {
		return nil
	}
	img, ok := pv.cache[pv.currentURL]
	if !ok {
		return nil
	}
	url, seq, cols, rows := pv.currentURL, pv.renderSeq, pv.cols, pv.rows
	return func() tea.Msg {
		apc, err := buildPictureKittyAPC(img, productKittyImageID, cols, rows)
		if err != nil {
			return pictureKittyFrameMsg{url: url, seq: seq, cols: cols, rows: rows, err: err}
		}
		grid := buildPictureKittyGrid(cols, rows, productKittyImageID)
		return pictureKittyFrameMsg{url: url, seq: seq, cols: cols, rows: rows, apc: apc, grid: grid}
	}
}

// Update handles the pictureView's async messages.
func (pv *pictureView) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case productImageFetchRequestedMsg:
		if msg.url != pv.currentURL || msg.seq != pv.renderSeq {
			return nil
		}
		if _, ok := pv.cache[msg.url]; ok {
			return pv.renderCmd()
		}
		if _, ok := pv.errs[msg.url]; ok {
			return nil
		}
		if pv.loading[msg.url] {
			return nil
		}
		pv.loading[msg.url] = true
		return fetchProductImage(msg.url)

	case productImageLoadedMsg:
		delete(pv.loading, msg.url)
		if msg.err != nil {
			pv.errs[msg.url] = msg.err
			return nil
		}
		pv.rememberImage(msg.url, msg.img)
		if msg.url == pv.currentURL {
			pv.glyphCache = ""
			pv.glyphKey = ""
			return pv.renderCmd()
		}
	case pictureKittyFrameMsg:
		// Discard stale frames from a URL, size, mode, or render generation
		// we've already moved past.
		if msg.url != pv.currentURL || msg.seq != pv.renderSeq || msg.cols != pv.cols || msg.rows != pv.rows || pv.mode != PictureKitty {
			return nil
		}
		if msg.err != nil {
			pv.mode = PictureGlyph
			pv.invalidateRenderedFrame()
			return nil
		}
		pv.kittyGrid = msg.grid
		return tea.Raw(msg.apc)
	}
	return nil
}

func (pv *pictureView) rememberImage(url string, img image.Image) {
	if _, ok := pv.cache[url]; !ok {
		pv.cacheOrder = append(pv.cacheOrder, url)
	}
	pv.cache[url] = img
	pv.trimImageCache()
}

func (pv *pictureView) trimImageCache() {
	for len(pv.cacheOrder) > productImageCacheLimit {
		evict := pv.cacheOrder[0]
		pv.cacheOrder = pv.cacheOrder[1:]
		if evict == pv.currentURL {
			pv.cacheOrder = append(pv.cacheOrder, evict)
			continue
		}
		delete(pv.cache, evict)
	}
}

// View renders the pane's inner content (caller wraps with border/padding).
func (pv *pictureView) View() string {
	if pv.currentURL == "" {
		return "No image"
	}
	if err, ok := pv.errs[pv.currentURL]; ok {
		return "Image error:\n" + err.Error()
	}
	if _, ok := pv.cache[pv.currentURL]; !ok {
		return "Loading image…"
	}
	if pv.cols <= 0 || pv.rows <= 0 {
		return ""
	}
	if pv.mode == PictureKitty {
		if pv.kittyGrid != "" {
			return pv.kittyGrid
		}
		return "Rendering…"
	}
	key := fmt.Sprintf("%s|%d|%d", pv.currentURL, pv.cols, pv.rows)
	if pv.glyphKey == key && pv.glyphCache != "" {
		return pv.glyphCache
	}
	img := pv.cache[pv.currentURL]
	ascii, err := ansimage.NewScaledFromImage(
		img,
		pv.rows*2, pv.cols,
		color.Transparent,
		ansimage.ScaleModeFit,
		ansimage.NoDithering,
	)
	if err != nil {
		return "glyph render error: " + err.Error()
	}
	out := ascii.RenderExt(false, false)
	pv.glyphCache = out
	pv.glyphKey = key
	return out
}

// fetchProductImage returns a Cmd that GETs url, decodes it, and emits a
// productImageLoadedMsg. A 15MB/15s limit keeps pathological URLs from
// hanging the UI.
func fetchProductImage(url string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			return productImageLoadedMsg{url: url, err: err}
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return productImageLoadedMsg{url: url, err: errors.New(resp.Status)}
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 15*1024*1024))
		if err != nil {
			return productImageLoadedMsg{url: url, err: err}
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return productImageLoadedMsg{url: url, err: err}
		}
		return productImageLoadedMsg{url: url, img: img}
	}
}

func requestProductImageFetch(url string, seq uint64) tea.Cmd {
	return tea.Tick(productImageFetchDelay, func(time.Time) tea.Msg {
		return productImageFetchRequestedMsg{url: url, seq: seq}
	})
}

// buildPictureKittyAPC mirrors mapview's transmit APC — scaled to the pane's
// cell grid, and using a distinct image ID so it doesn't collide with the
// map placement.
func buildPictureKittyAPC(img image.Image, id, cols, rows int) (string, error) {
	var buf bytes.Buffer
	opts := &kitty.Options{
		Action:           kitty.TransmitAndPut,
		Transmission:     kitty.Direct,
		Format:           kitty.PNG,
		ID:               id,
		Columns:          cols,
		Rows:             rows,
		VirtualPlacement: true,
		Quite:            2,
		Chunk:            true,
	}
	if err := kitty.EncodeGraphics(&buf, img, opts); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// buildPictureKittyGrid builds the cols×rows Unicode-placeholder grid that
// references imageID — one printable cell per grid position, each carrying
// the image ID in its SGR truecolor triple and row/column diacritics.
func buildPictureKittyGrid(cols, rows, imageID int) string {
	r := (imageID >> 16) & 0xff
	g := (imageID >> 8) & 0xff
	b := imageID & 0xff
	sgr := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
	reset := "\x1b[39m"
	var sb strings.Builder
	sb.Grow((cols*4 + len(sgr) + len(reset) + 1) * rows)
	for y := 0; y < rows; y++ {
		sb.WriteString(sgr)
		rowDia := kitty.Diacritic(y)
		for x := 0; x < cols; x++ {
			sb.WriteRune(kitty.Placeholder)
			sb.WriteRune(rowDia)
			sb.WriteRune(kitty.Diacritic(x))
		}
		sb.WriteString(reset)
		if y < rows-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// kittyDeletePictureImage returns an APC that tells the terminal to drop our
// uploaded bitmap. Emitted when leaving Kitty mode.
func kittyDeletePictureImage() string {
	return fmt.Sprintf("\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", productKittyImageID)
}
