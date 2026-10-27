package main

import (
	"math"
	"time"

	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"
)

// How long things take to move, before reduce motion is asked for.
const (
	messageRise   = 280 * time.Millisecond // a new message rising into place
	paneSlide     = 300 * time.Millisecond // the open chat sliding in on a phone
	rowLeave      = 280 * time.Millisecond // an archived chat sliding out of the list
	dialogRise    = 260 * time.Millisecond
	tabMove       = 200 * time.Millisecond // the line under the chosen tab
	riseDistance  = 12                     // pixels a message or dialog rises as it appears
	slideDistance = 48                     // pixels a pane slides as it appears
	leaveDistance = 40                     // pixels an archived row slides out
)

// The typing dots: three that rise and fall one after the other, as in a messaging app.
const (
	dotCount     = 3
	dotGap       = 4
	dotHop       = 4                       // pixels a dot rises at the top of its hop
	dotCycle     = 1200 * time.Millisecond // one rise and fall of all three
	dotDelay     = 0.125                   // of a cycle, between one dot and the next
	dotFrameTime = 33 * time.Millisecond   // frames while dots move: about 30 a second is smooth enough
	dotQuiet     = 0.45                    // how see-through a dot is when resting
)

// motion is how long a movement takes: none at all when the person asked their system for
// less motion.
func (chat *screen) motion(duration time.Duration) time.Duration {
	if chat.reduceMotion {
		return 0
	}
	return duration
}

// rising is the look of something appearing: it fades in and rises the last pixels into place.
func rising(appeared float32, look ui.Look) ui.Look {
	look.Fade = 1 - appeared
	look.ShiftY = (1 - appeared) * riseDistance
	return look
}

// typingDots draws three dots that hop one after the other, and asks for the frames that move
// them. With reduce motion they stay still.
func (chat *screen) typingDots(context *ui.Context, size float32, colour draw2d.Color, now time.Time) {
	if !chat.reduceMotion {
		context.RequestFrame(dotFrameTime)
	}
	cycle := float64(now.UnixMilli()%dotCycle.Milliseconds()) / float64(dotCycle.Milliseconds())
	context.Box(ui.Layout{Gap: dotGap, AlignY: ui.Center, Height: ui.Fixed(size + dotHop)}, func() {
		for index := range dotCount {
			height := hopHeight(cycle - float64(index)*dotDelay)
			if chat.reduceMotion {
				height = 0
			}
			look := ui.Look{Fill: colour, Radius: size / 2, ShiftY: -height * dotHop, Fade: dotQuiet * (1 - height)}
			context.Frame(ui.Layout{Width: ui.Fixed(size), Height: ui.Fixed(size)}, look, func() {})
		}
	})
}

// hopHeight is how high a dot is at a point of its cycle, from 0 (resting) to 1: it hops in the
// first part of the cycle and rests for the rest, so the three hop as a wave.
func hopHeight(cycle float64) float32 {
	const hopShare = 0.6 // of the cycle a dot spends hopping
	cycle -= math.Floor(cycle)
	if cycle > hopShare {
		return 0
	}
	return float32(math.Sin(cycle / hopShare * math.Pi))
}
