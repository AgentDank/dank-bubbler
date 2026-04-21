package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/AgentDank/dank-bubbler/internal/models"
	"github.com/charmbracelet/x/ansi"
)

func TestLayoutDimensions(t *testing.T) {
	products := []models.Product{
		{
			BrandName:          "Test Brand",
			DosageForm:         "Flower",
			RegistrationNumber: "123",
		},
	}
	brands := []models.Brand{{Name: "Test Brand"}}

	// Create browser
	pb := NewProductBrowser(products, brands, nil)

	wideSize := struct{ w, h int }{80, 24}
	pb.Update(tea.WindowSizeMsg{Width: wideSize.w, Height: wideSize.h})

	wideView := pb.View()
	wideContent := wideView.Content
	wideText := ansi.Strip(wideContent)

	if !strings.Contains(wideText, appHeader) {
		t.Fatalf("expected header bar to contain app title")
	}

	if !strings.Contains(wideText, "q quit") {
		t.Fatalf("expected footer bar to contain help text")
	}

	// Test cases for different window sizes, including narrower terminals.
	testSizes := []struct{ w, h int }{
		{60, 20},
		{72, 20},
		{80, 24},
		{100, 40},
		{120, 50},
	}

	for _, sz := range testSizes {
		// Update dimensions via Update (simulating window resize)
		pb.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})

		view := pb.View()
		content := view.Content

		actualHeight := lipgloss.Height(content)
		actualWidth := lipgloss.Width(content)
		lines := strings.Split(ansi.Strip(content), "\n")
		middleMaxWidth := 0
		if len(lines) > 2 {
			for _, line := range lines[1 : len(lines)-1] {
				middleMaxWidth = max(middleMaxWidth, lipgloss.Width(line))
			}
		}

		// Check Width
		if actualWidth > sz.w {
			t.Logf("Full View:\n%s", content)
			t.Errorf("Window Size: %dx%d. Generated View Width: %d. Overflow: %d", sz.w, sz.h, actualWidth, actualWidth-sz.w)

			// Analyze components to see where the overflow is
			// We can inspect internal state usage if needed, but the error confirms the issue.
			// Let's print component widths calculation based on our analysis

			// Left List
			// width = pb.width / 3
			// style = border (+2) + padding(0,1) -> (+2) = +4
			// total = w/3 + 4

			leftW := sz.w/3 + 4

			// Right Panes
			// width = (sz.w * 2) / 3
			// Info Pane: style = border(+2) + padding(1,2) -> (+4h) = +6
			// Chart Pane: style = border(+2) + padding(0,1) -> (+2h) = +4

			rightInfoW := (sz.w*2)/3 + 6
			// rightChartW := (sz.w*2)/3 + 4 // This one is smaller horizontally? View joins vertically.

			// JoinHorizontal(Left, Right)
			// Total width = leftW + max(rightInfoW, rightChartW)

			calcW := leftW + rightInfoW // since rightInfoW > rightChartW
			t.Logf("Calculated Expected Width: %d (Left: %d + RightInfo: %d)", calcW, leftW, rightInfoW)
		}

		if middleMaxWidth != sz.w {
			t.Errorf("Window Size: %dx%d. Middle content width: %d. Expected: %d", sz.w, sz.h, middleMaxWidth, sz.w)
		}

		// Check Height
		if actualHeight > sz.h {
			t.Errorf("Window Size: %dx%d. Generated View Height: %d. Overflow: %d", sz.w, sz.h, actualHeight, actualHeight-sz.h)

			// Left List Height:
			// height = sz.h - 3
			// style = border(+2) + padding(0,1) -> (+0v) = +2
			// total = (h-3) + 2 = h-1

			// Right Info:
			// height = (h-3)/2
			// style = border(+2) + padding(1,2) -> (+2v) = +4
			// total = (h-3)/2 + 4

			// Right Chart:
			// height = (h-3)/2
			// style = border(+2) + padding(0,1) -> (+0v) = +2
			// total = (h-3)/2 + 2

			// Right Total = RightInfo + RightChart
			// = (h-3)/2 + 4 + (h-3)/2 + 2 = 2*((h-3)/2) + 6
			// ~= h-3 + 6 = h+3

			// Help text height = 1

			// Total View Height = max(Left, Right) + Help
			// Left = h-1
			// Right = h+3
			// Total = h+3 + 1 = h+4

			rightH := ((sz.h-3)/2)*2 + 6
			t.Logf("Calculated Expected Right Pane Height: %d", rightH)
		}
	}
}

func TestZoningLayoutFitsWindow(t *testing.T) {
	z := NewZoningBrowser(nil)
	sizes := []struct{ w, h int }{{80, 24}, {100, 40}, {120, 50}}
	for _, sz := range sizes {
		z.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
		v := z.View()
		if w := lipgloss.Width(v.Content); w > sz.w {
			t.Errorf("zoning: size %dx%d overflow width %d", sz.w, sz.h, w)
		}
		if h := lipgloss.Height(v.Content); h > sz.h {
			t.Errorf("zoning: size %dx%d overflow height %d", sz.w, sz.h, h)
		}
	}
}

func TestRetailLayoutFitsWindow(t *testing.T) {
	r := NewRetailBrowser(nil)
	sizes := []struct{ w, h int }{{80, 24}, {100, 40}, {120, 50}}
	for _, sz := range sizes {
		r.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
		v := r.View()
		if w := lipgloss.Width(v.Content); w > sz.w {
			t.Errorf("retail: size %dx%d overflow width %d", sz.w, sz.h, w)
		}
		if h := lipgloss.Height(v.Content); h > sz.h {
			t.Errorf("retail: size %dx%d overflow height %d", sz.w, sz.h, h)
		}
	}
}
