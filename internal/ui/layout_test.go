package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/AgentDank/dank-bubbler/internal/models"
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

	// Test cases for different window sizes
	testSizes := []struct{ w, h int }{
		{80, 24},
		{100, 40},
		{120, 50},
	}

	for _, sz := range testSizes {
		// Update dimensions via Update (simulating window resize)
		pb.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})

		view := pb.View()

		actualHeight := lipgloss.Height(view.Content)
		actualWidth := lipgloss.Width(view.Content)

		// Check Width
		if actualWidth > sz.w {
			t.Logf("Full View:\n%s", view.Content)
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
