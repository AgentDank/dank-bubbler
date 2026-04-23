package mapview

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/eliukblau/pixterm/pkg/ansimage"
	sm "github.com/flopp/go-staticmaps"
	"github.com/golang/geo/s2"
)

// RenderMode selects how the map is rendered. RenderGlyph uses the default
// half-block ANSI renderer (works on any terminal). RenderKitty emits the
// image via the Kitty graphics protocol and only works on terminals that
// support it (Kitty, Ghostty, WezTerm, etc).
type RenderMode int8

const (
	RenderGlyph RenderMode = iota
	RenderKitty
)

type Style int8

const (
	Wikimedia Style = iota
	OpenStreetMaps
	OpenTopoMap
	OpenCycleMap
	CartoLight
	CartoDark
	StamenToner
	StamenTerrain
	ThunderforestLandscape
	ThunderforestOutdoors
	ThunderforestTransport
	ArcgisWorldImagery
)

type MapRender string
type MapCoordinates struct {
	Lat float64
	Lng float64
	Err error
}

// IsMapUpdate reports whether msg is a message the mapview's Update method
// needs to see. Parent components (e.g. a page that contains a mapview
// alongside other focusable widgets) should forward matching messages to the
// mapview regardless of which sub-widget currently has focus — otherwise
// async render results get routed to the wrong widget and lost.
func IsMapUpdate(msg tea.Msg) bool {
	switch msg.(type) {
	case MapRender, MapCoordinates, kittyFrameMsg:
		return true
	}
	return false
}

type NominatimResponse []struct {
	PlaceID     int    `json:"place_id"`
	License     string `json:"license"`
	OSMType     string `json:"osm_type"`
	OSMID       int    `json:"osm_id"`
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
}

type KeyMap struct {
	Up      key.Binding
	Right   key.Binding
	Down    key.Binding
	Left    key.Binding
	ZoomIn  key.Binding
	ZoomOut key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("↑/l", "right"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("↑/h", "left"),
		),
		ZoomIn: key.NewBinding(
			key.WithKeys("+", "="),
			key.WithHelp("+", "plus"),
		),
		ZoomOut: key.NewBinding(
			key.WithKeys("-", "_"),
			key.WithHelp("-", "minus"),
		),
	}
}

// Marker is a point to draw on the map. Color and Size are optional and fall
// back to sensible defaults (red, size 16) when left zero.
type Marker struct {
	Lat, Lng float64
	Color    color.Color
	Size     float64
}

type Model struct {
	Width  int
	Height int
	KeyMap KeyMap

	Style lipgloss.Style

	initialized bool

	osm          *sm.Context
	tileProvider *sm.TileProvider
	lat          float64
	lng          float64
	loc          string
	zoom         int
	maprender    string
	markers      []Marker
	renderMode   RenderMode
	tileStyle    Style
}

func New(width, height int) (m Model) {
	m.Width = width
	m.Height = height
	m.setInitialValues()
	return m
}

func (m *Model) setInitialValues() {
	m.KeyMap = DefaultKeyMap()
	m.osm = sm.NewContext()
	m.osm.SetSize(400, 400)
	m.tileProvider = sm.NewTileProviderOpenStreetMaps()
	m.tileStyle = OpenStreetMaps
	m.zoom = 15
	m.lat = 25.0782266
	m.lng = -77.3383438
	m.loc = ""
	m.applyToOSM()
	m.applyMarkersToOSM()
	m.initialized = true
}

func (m *Model) applyToOSM() {
	m.osm.SetTileProvider(m.tileProvider)
	m.osm.SetCenter(s2.LatLngFromDegrees(m.lat, m.lng))
	m.osm.SetZoom(m.zoom)
}

// Center returns the current map center.
func (m Model) Center() (lat, lng float64) { return m.lat, m.lng }

// SetMarkers replaces all currently-drawn markers on the map. Pass an empty
// slice (or call ClearMarkers) to remove them.
func (m *Model) SetMarkers(markers []Marker) {
	m.markers = markers
	m.applyMarkersToOSM()
}

// ClearMarkers removes all markers from the map.
func (m *Model) ClearMarkers() {
	m.markers = nil
	if m.osm != nil {
		m.osm.ClearObjects()
	}
}

// applyMarkersToOSM clears the underlying context's objects and re-adds the
// current marker set. Markers are drawn in slice order, so callers that want
// a particular marker drawn on top should place it last.
func (m *Model) applyMarkersToOSM() {
	if m.osm == nil {
		return
	}
	m.osm.ClearObjects()
	for _, mk := range m.markers {
		col := mk.Color
		if col == nil {
			col = color.RGBA{0xff, 0x00, 0x00, 0xff}
		}
		size := mk.Size
		if size == 0 {
			size = 16
		}
		m.osm.AddObject(sm.NewMarker(s2.LatLngFromDegrees(mk.Lat, mk.Lng), col, size))
	}
}

// Zoom returns the current zoom level.
func (m Model) Zoom() int {
	return m.zoom
}

func (m *Model) SetLatLng(lat float64, lng float64, zoom int) {
	m.lat = lat
	m.lng = lng
	m.zoom = zoom
	m.applyToOSM()
}

func (m *Model) SetLocation(loc string, zoom int) {
	m.loc = loc
	m.zoom = zoom
	m.applyToOSM()
}

func getThunderforestAPIKey() string {
	// In a real application, you would want to load this from an environment variable or configuration file
	return "YOUR_THUNDERFOREST_API_KEY"
}

// RenderMode returns the current render mode.
func (m Model) RenderMode() RenderMode { return m.renderMode }

// SetRenderMode sets the render mode and returns a tea.Cmd that re-renders
// the map at the new mode. Use the returned cmd so callers don't have to
// manually poke the render loop. When leaving Kitty mode, a delete-image APC
// is also emitted so the terminal drops the uploaded bitmap.
func (m *Model) SetRenderMode(mode RenderMode) tea.Cmd {
	prev := m.renderMode
	m.renderMode = mode
	cmd := m.render(m.Width, m.Height)
	if prev == RenderKitty && mode != RenderKitty {
		return tea.Batch(tea.Raw(kittyDeleteImage(kittyMapImageID)), cmd)
	}
	return cmd
}

// TileStyle returns the currently-selected tile style.
func (m Model) TileStyle() Style { return m.tileStyle }

// SetStyle switches the tile provider and returns a tea.Cmd that re-renders
// the map at the new style.
func (m *Model) SetStyle(style Style) tea.Cmd {
	switch style {
	case Wikimedia:
		m.tileProvider = sm.NewTileProviderWikimedia()
	case OpenStreetMaps:
		m.tileProvider = sm.NewTileProviderOpenStreetMaps()
	case OpenTopoMap:
		m.tileProvider = sm.NewTileProviderOpenTopoMap()
	case OpenCycleMap:
		m.tileProvider = sm.NewTileProviderOpenCycleMap()
	case CartoLight:
		m.tileProvider = sm.NewTileProviderCartoLight()
	case CartoDark:
		m.tileProvider = sm.NewTileProviderCartoDark()
	case StamenToner:
		m.tileProvider = sm.NewTileProviderStamenToner()
	case StamenTerrain:
		m.tileProvider = sm.NewTileProviderStamenTerrain()
	case ThunderforestLandscape:
		m.tileProvider = sm.NewTileProviderThunderforestLandscape(getThunderforestAPIKey())
	case ThunderforestOutdoors:
		m.tileProvider = sm.NewTileProviderThunderforestOutdoors(getThunderforestAPIKey())
	case ThunderforestTransport:
		m.tileProvider = sm.NewTileProviderThunderforestTransport(getThunderforestAPIKey())
	case ArcgisWorldImagery:
		m.tileProvider = sm.NewTileProviderArcgisWorldImagery()
	}
	m.tileStyle = style
	m.applyToOSM()
	return m.render(m.Width, m.Height)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	if !m.initialized {
		m.setInitialValues()
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		var hit = false
		movement := (1000 / math.Pow(2, float64(m.zoom))) / 3

		switch {

		case key.Matches(msg, m.KeyMap.Up):
			m.lat += movement
			if m.lat > 90.0 {
				m.lat = -90.0
			}
			hit = true

		case key.Matches(msg, m.KeyMap.Right):
			m.lng += movement
			if m.lng > 180.0 {
				m.lng = -180.0
			}
			hit = true

		case key.Matches(msg, m.KeyMap.Down):
			m.lat -= movement
			if m.lat < -90.0 {
				m.lat = 90.0
			}
			hit = true

		case key.Matches(msg, m.KeyMap.Left):
			m.lng -= movement
			if m.lng < -180.0 {
				m.lng = 180.0
			}
			hit = true

		case key.Matches(msg, m.KeyMap.ZoomIn):
			if m.zoom < 16 {
				m.zoom += 1
			}
			hit = true

		case key.Matches(msg, m.KeyMap.ZoomOut):
			if m.zoom > 2 {
				m.zoom -= 1
			}
			hit = true

		}
		if hit {
			m.applyToOSM()
			cmds = append(cmds, m.render(m.Width, m.Height))
			return m, tea.Batch(cmds...)
		}

	case MapRender:
		m.maprender = string(msg)
		return m, nil

	case kittyFrameMsg:
		// Split a Kitty render: upload the image via tea.Raw (side channel,
		// bypasses the cell renderer) and set the view to the placeholder
		// grid. The terminal ties them together via image ID + diacritics.
		m.maprender = msg.grid
		return m, tea.Raw(msg.apc)

	case MapCoordinates:
		m.loc = ""
		if msg.Err != nil {
			m.maprender = msg.Err.Error()
		} else {
			m.lat = msg.Lat
			m.lng = msg.Lng
			m.applyToOSM()
		}
		return m, m.render(m.Width, m.Height)

	}

	if m.initialized && m.loc != "" {
		cmds = append(cmds, m.lookup(m.loc))
		return m, tea.Batch(cmds...)
	}

	if m.initialized && m.loc == "" && m.maprender == "" {
		cmds = append(cmds, m.render(m.Width, m.Height))
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) render(width, height int) tea.Cmd {
	if m.renderMode == RenderKitty {
		return m.renderKitty(width, height)
	}
	return m.renderGlyph(width, height)
}

func (m *Model) renderGlyph(width, height int) tea.Cmd {
	return func() tea.Msg {
		img, err := m.osm.Render()
		if err != nil {
			return MapRender(err.Error())
		}

		ascii, err := ansimage.NewScaledFromImage(
			img,
			(height * 2),
			width,
			color.Transparent,
			ansimage.ScaleModeFill,
			ansimage.NoDithering,
		)
		if err != nil {
			return MapRender(err.Error())
		}

		return MapRender(ascii.RenderExt(false, false))
	}
}

func (m *Model) lookup(address string) tea.Cmd {
	return func() tea.Msg {
		u := fmt.Sprintf(
			"https://nominatim.openstreetmap.org/search?q=%s&format=json&polygon=1&addressdetails=1",
			url.QueryEscape(address),
		)

		resp, err := http.Get(u)
		if err != nil {
			return MapCoordinates{Err: err}
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return MapCoordinates{Err: errors.New(string(body))}
		}

		var data NominatimResponse
		if err := json.Unmarshal(body, &data); err != nil {
			return MapCoordinates{Err: err}
		}

		if len(data) == 0 {
			return MapCoordinates{Err: errors.New("Location not found")}
		}

		lat, err := strconv.ParseFloat(data[0].Lat, 64)
		if err != nil {
			return MapCoordinates{Err: err}
		}
		lon, err := strconv.ParseFloat(data[0].Lon, 64)
		if err != nil {
			return MapCoordinates{Err: err}
		}

		return MapCoordinates{
			Lat: lat,
			Lng: lon,
		}
	}
}

func (m Model) View() tea.View {
	return tea.NewView(m.maprender)
}
