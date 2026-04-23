package ui

import (
	"math/rand/v2"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// appTitle is the static decorative title that renders in the top-right
// corner of every page's header, immediately after the animated face.
const appTitle = " AgentDank dank-bubbler-ct 𖠞༄"

// currentFace is the animated Egyptian-eye face currently displayed in the
// header. Mutated in the tea.Model Update loop on each faceTickMsg and read
// by renderAppHeader during View. Writes and reads always happen on the main
// bubbletea goroutine so no locking is needed.
var currentFace = faceFrames[0]

// faceFrames drives the header-face animation cycle. Each frame is held for
// faceFrameInterval; the table is mostly "open eyes" with occasional look-
// around and blink frames so the face feels alive without being distracting.
var faceFrames = []string{
	"𓁹‿𓁹", "𓁹‿𓁹", "𓁹‿𓁹", "𓁹‿𓁹", "𓁹‿𓁹", "𓁹‿𓁹", "𓁹‿𓁹", "𓁹‿𓁹",
	"𓂀‿𓁹",               // left glance
	"𓁹‿𓁹", "𓁹‿𓁹", "𓁹‿𓁹", // hold
	"𓁹‿𓂀", // right glance
	"𓁹‿𓁹", "𓁹‿𓁹", "𓁹‿𓁹", "𓁹‿𓁹", "𓁹‿𓁹", "𓁹‿𓁹", "𓁹‿𓁹", "𓁹‿𓁹",
	"-‿-", // blink
}

// faceFrameInterval is the hold time per frame. ~250ms gives a visible blink
// that doesn't feel manic.
const faceFrameInterval = 250 * time.Millisecond

// faceTickMsg is the internal tick that advances the face animation.
type faceTickMsg struct{}

// faceTickCmd schedules the next face-animation tick.
func faceTickCmd() tea.Cmd {
	return tea.Tick(faceFrameInterval, func(_ time.Time) tea.Msg { return faceTickMsg{} })
}

// maxFaceOffset bounds how far the face can drift horizontally inside its
// reserved slot. The slot width is maxFaceOffset + face width + appTitle, so
// leading+trailing padding always sums to maxFaceOffset — the title stays
// pinned and the face slides inside that zone.
const maxFaceOffset = 2

// faceOffset is the current leading-space count in [0, maxFaceOffset]. It
// changes ~every faceSlideInterval via a small random walk for a subtle,
// organic drift. Like currentFace it's read/written only on the main Update
// goroutine.
var faceOffset int

// faceSlideInterval is how often the random-walk step fires.
const faceSlideInterval = 2 * time.Second

// faceSlideMsg is the internal tick that nudges faceOffset by ±1 (or 0).
type faceSlideMsg struct{}

// faceSlideCmd schedules the next slide step.
func faceSlideCmd() tea.Cmd {
	return tea.Tick(faceSlideInterval, func(_ time.Time) tea.Msg { return faceSlideMsg{} })
}

// stepFaceOffset performs the random walk one step: 50% chance to stay put,
// otherwise ±1 with equal probability. Clamps to [0, maxFaceOffset].
func stepFaceOffset() {
	if rand.IntN(2) == 0 {
		return // 50%: stay
	}
	delta := 1
	if rand.IntN(2) == 0 {
		delta = -1
	}
	next := faceOffset + delta
	if next < 0 {
		next = 0
	}
	if next > maxFaceOffset {
		next = maxFaceOffset
	}
	faceOffset = next
}

// renderFaceSlot returns the face prefixed/suffixed with spaces so that the
// slot is always the same total width (currentFace + maxFaceOffset padding).
// Keeping the slot width constant is what guarantees the rest of the header
// never shifts as the face drifts.
func renderFaceSlot() string {
	return strings.Repeat(" ", faceOffset) + currentFace + strings.Repeat(" ", maxFaceOffset-faceOffset)
}
