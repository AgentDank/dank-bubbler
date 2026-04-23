package mapview

import (
	"image/color"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNewInitializesDefaultState(t *testing.T) {
	m := New(80, 24)

	if m.Width != 80 {
		t.Fatalf("expected width 80, got %d", m.Width)
	}

	if m.Height != 24 {
		t.Fatalf("expected height 24, got %d", m.Height)
	}

	if !m.initialized {
		t.Fatal("expected model to be initialized")
	}

	if m.zoom != 15 {
		t.Fatalf("expected default zoom 15, got %d", m.zoom)
	}

	if m.lat != 25.0782266 {
		t.Fatalf("expected default lat 25.0782266, got %f", m.lat)
	}

	if m.lng != -77.3383438 {
		t.Fatalf("expected default lng -77.3383438, got %f", m.lng)
	}

	if m.loc != "" {
		t.Fatalf("expected empty location, got %q", m.loc)
	}

	if m.osm == nil {
		t.Fatal("expected static map context to be initialized")
	}

	if m.tileProvider == nil {
		t.Fatal("expected tile provider to be initialized")
	}

	if m.View().Content != "" {
		t.Fatalf("expected empty initial view, got %q", m.View().Content)
	}
}

func TestUpdateHandlesCoordinatesAndRenderMessage(t *testing.T) {
	m := New(80, 24)

	updated, cmd := m.Update(MapCoordinates{Lat: 41.5, Lng: -72.7})
	if cmd == nil {
		t.Fatal("expected render command after coordinate update")
	}

	if updated.lat != 41.5 {
		t.Fatalf("expected lat 41.5, got %f", updated.lat)
	}

	if updated.lng != -72.7 {
		t.Fatalf("expected lng -72.7, got %f", updated.lng)
	}

	if updated.loc != "" {
		t.Fatalf("expected location to remain empty, got %q", updated.loc)
	}

	updated, cmd = updated.Update(MapRender("rendered map"))
	if cmd != nil {
		t.Fatal("expected no command after receiving rendered map")
	}

	if updated.View().Content != "rendered map" {
		t.Fatalf("expected view to return rendered map, got %q", updated.View().Content)
	}
}

func TestSetMarkersStoresAndClears(t *testing.T) {
	m := New(80, 24)

	m.SetMarkers([]Marker{
		{Lat: 41.5, Lng: -72.7},
		{Lat: 41.6, Lng: -72.8, Color: color.RGBA{0x00, 0xff, 0x00, 0xff}, Size: 20},
	})

	if len(m.markers) != 2 {
		t.Fatalf("expected 2 markers, got %d", len(m.markers))
	}
	if m.markers[0].Color != nil {
		t.Errorf("expected first marker to keep nil color for defaulting, got %#v", m.markers[0].Color)
	}
	if m.markers[1].Size != 20 {
		t.Errorf("expected second marker size 20, got %v", m.markers[1].Size)
	}

	m.ClearMarkers()
	if len(m.markers) != 0 {
		t.Fatalf("expected markers to be cleared, got %d", len(m.markers))
	}
}

func TestSetRenderModeTogglesAndReRenders(t *testing.T) {
	m := New(80, 24)

	if m.RenderMode() != RenderGlyph {
		t.Fatalf("expected default render mode RenderGlyph, got %v", m.RenderMode())
	}

	cmd := m.SetRenderMode(RenderKitty)
	if cmd == nil {
		t.Fatal("expected SetRenderMode to return a render cmd")
	}
	if m.RenderMode() != RenderKitty {
		t.Fatalf("expected RenderKitty after set, got %v", m.RenderMode())
	}

	cmd = m.SetRenderMode(RenderGlyph)
	if cmd == nil {
		t.Fatal("expected SetRenderMode to return a render cmd when going back to glyph")
	}
	if m.RenderMode() != RenderGlyph {
		t.Fatalf("expected RenderGlyph after toggle back, got %v", m.RenderMode())
	}
}

func TestUpdateZoomInRespectsUpperBound(t *testing.T) {
	m := New(80, 24)
	m.zoom = 16

	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Text: "+", Code: '+'}))
	if cmd == nil {
		t.Fatal("expected render command after zoom-in keypress")
	}

	if updated.zoom != 16 {
		t.Fatalf("expected zoom to stay capped at 16, got %d", updated.zoom)
	}
}
