// Command kitty-probe emits Kitty graphics APCs straight to stdout and
// exits. Two modes:
//
//	go run ./cmd/kitty-probe -red             # direct placement (APC+put)
//	go run ./cmd/kitty-probe -placeholder     # transmit + Unicode placeholders
//
// The direct-placement mode is what the old mapview renderer did. The
// placeholder mode is what we switched to, since it survives bubbletea's
// cell-based renderer.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"strings"

	"github.com/charmbracelet/x/ansi/kitty"
	sm "github.com/flopp/go-staticmaps"
	"github.com/golang/geo/s2"
)

func main() {
	red := flag.Bool("red", false, "emit a solid-red test image instead of a map")
	cols := flag.Int("cols", 40, "display width in terminal cells")
	rows := flag.Int("rows", 20, "display height in terminal cells")
	chunk := flag.Bool("chunk", true, "chunk the APC payload (kitty protocol >4KB requires this)")
	imageID := flag.Int("id", 42, "kitty image id")
	placeholder := flag.Bool("placeholder", false, "use Unicode placeholder protocol instead of direct placement")
	flag.Parse()

	var img image.Image
	if *red {
		ri := image.NewRGBA(image.Rect(0, 0, 64, 64))
		draw.Draw(ri, ri.Bounds(), &image.Uniform{C: color.RGBA{0xff, 0x00, 0x00, 0xff}}, image.Point{}, draw.Src)
		img = ri
	} else {
		ctx := sm.NewContext()
		ctx.SetSize(400, 400)
		ctx.SetTileProvider(sm.NewTileProviderOpenStreetMaps())
		ctx.SetCenter(s2.LatLngFromDegrees(41.76, -72.68))
		ctx.SetZoom(12)
		m, err := ctx.Render()
		if err != nil {
			fmt.Fprintln(os.Stderr, "map render:", err)
			os.Exit(1)
		}
		img = m
	}

	var buf bytes.Buffer
	opts := &kitty.Options{
		Action:          kitty.TransmitAndPut,
		Transmission:    kitty.Direct,
		Format:          kitty.PNG,
		ID:              *imageID,
		Columns:         *cols,
		Rows:            *rows,
		DoNotMoveCursor: true,
		Quite:           2,
		Chunk:           *chunk,
	}
	if *placeholder {
		opts.VirtualPlacement = true
		opts.Columns = 0
		opts.Rows = 0
	}
	if err := kitty.EncodeGraphics(&buf, img, opts); err != nil {
		fmt.Fprintln(os.Stderr, "encode graphics:", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "---- diagnostics ----")
	fmt.Fprintf(os.Stderr, "mode         : %s\n", map[bool]string{true: "unicode placeholder", false: "direct placement"}[*placeholder])
	fmt.Fprintf(os.Stderr, "options      : %s\n", opts)
	fmt.Fprintf(os.Stderr, "apc bytes    : %d\n", buf.Len())
	fmt.Fprintf(os.Stderr, "chunks       : %d\n", strings.Count(buf.String(), "\x1b_G"))
	fmt.Fprintf(os.Stderr, "display size : %d cols x %d rows\n", *cols, *rows)
	fmt.Fprintln(os.Stderr, "---------------------")
	fmt.Fprintln(os.Stderr)

	// Emit APC first.
	os.Stdout.Write(buf.Bytes())

	if *placeholder {
		// Then emit placeholder grid referencing the image id.
		r := (*imageID >> 16) & 0xff
		g := (*imageID >> 8) & 0xff
		b := *imageID & 0xff
		sgr := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
		reset := "\x1b[39m"
		for y := 0; y < *rows; y++ {
			var line strings.Builder
			line.WriteString(sgr)
			for x := 0; x < *cols; x++ {
				line.WriteRune(kitty.Placeholder)
				line.WriteRune(kitty.Diacritic(y))
				line.WriteRune(kitty.Diacritic(x))
			}
			line.WriteString(reset)
			fmt.Println(line.String())
		}
	} else {
		// Direct placement: just a cols×rows whitespace region under the image.
		rowLine := strings.Repeat(" ", *cols)
		for i := 0; i < *rows; i++ {
			fmt.Println(rowLine)
		}
	}
}
