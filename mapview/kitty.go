package mapview

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi/kitty"
)

// kittyDumpEnv, when set to a writable path, causes each Kitty-mode render to
// write its transmit APC + placeholder grid to that path. Cat the file in a
// Kitty-capable terminal to compare against the output of cmd/kitty-probe.
const kittyDumpEnv = "DANK_KITTY_DUMP"

// kittyMapImageID is the Kitty protocol image ID used for the map placement.
// Re-uploading under the same ID replaces the prior image on compliant
// terminals, so the map occupies one slot regardless of how many times it
// is re-rendered.
const kittyMapImageID = 42

// kittyFrameMsg carries the results of a Kitty render. The Update loop splits
// it: apc goes to tea.Raw (side-channel write, bypasses the cell renderer),
// grid becomes the view content.
//
// We need the side channel because bubbletea's cell-based renderer rewrites
// cell.Content whenever a printable character follows an escape sequence (see
// ultraviolet/styled.go printString), so an APC embedded in the view string
// gets silently discarded before it reaches the terminal.
type kittyFrameMsg struct {
	apc  string
	grid string
}

// renderKitty renders the map and returns a cmd that produces a kittyFrameMsg.
// The mapview Update loop turns that into tea.Raw(apc) + MapRender(grid).
func (m *Model) renderKitty(width, height int) tea.Cmd {
	return func() tea.Msg {
		img, err := m.osm.Render()
		if err != nil {
			return MapRender(err.Error())
		}
		msg := kittyFrameMsg{
			apc:  kittyBuildTransmitAPC(img, kittyMapImageID, width, height),
			grid: kittyBuildPlaceholderGrid(width, height, kittyMapImageID),
		}
		if path := os.Getenv(kittyDumpEnv); path != "" {
			if f, ferr := os.Create(path); ferr == nil {
				_, _ = f.WriteString(msg.apc)
				_, _ = f.WriteString(msg.grid)
				_ = f.Close()
			}
		}
		return msg
	}
}

// kittyBuildTransmitAPC builds a Transmit-and-virtual-display APC for img.
// Action=T with VirtualPlacement=1 tells the terminal "this image is ready
// for Unicode placeholder cells to reference." The c/r options declare the
// cell extent so the terminal scales the image to fit the placeholder grid
// — without them it defaults to image_pixels / cell_pixels and diacritics
// beyond that extent render blank.
func kittyBuildTransmitAPC(img image.Image, id, cols, rows int) string {
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
		return ""
	}
	return buf.String()
}

// kittyBuildPlaceholderGrid returns a cols×rows grid of Kitty Unicode
// placeholder cells referencing imageID. Each cell carries the id in the
// foreground SGR (truecolor) and row/column diacritics identifying which
// part of the image belongs there.
func kittyBuildPlaceholderGrid(cols, rows, imageID int) string {
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

// kittyDeleteImage returns a Kitty APC that deletes the image with the given
// ID and its cached data. Emitted via tea.Raw when leaving Kitty mode.
func kittyDeleteImage(id int) string {
	return fmt.Sprintf("\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", id)
}
