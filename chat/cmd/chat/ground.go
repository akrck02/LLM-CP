package main

import (
	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"
)

// groundTint is the pastel the page's ground leans to: a mint of LLM-CP's accent, so the ground
// belongs with the pastel pictures without competing with them.
var groundTint = draw2d.MustHex("#B2E2E0")

// groundTintShare is how much of the tint goes into the page's background: enough to tell, little
// enough to stay calm under the text, by night and by day.
const groundTintShare = 0.12

// paintGround fills the page with one flat colour, every frame: with no picture kept between
// frames, nothing is lost when Android puts the app in the background.
func (chat *screen) paintGround(area draw2d.Rectangle, drawing *draw2d.List) {
	background := chat.design.Colors.Background
	plain := draw2d.Color{float32(background.Red) / 255, float32(background.Green) / 255, float32(background.Blue) / 255, 1}
	var ground draw2d.Color
	for channel := range ground {
		ground[channel] = plain[channel] + (groundTint[channel]-plain[channel])*groundTintShare
	}
	ground[3] = 1
	drawing.Rectangle(area, 0, ground, draw2d.Color{}, 0)
}

// solidSurface is the theme's surface laid over its background: glass's surface lets what is under
// it show, which a dialog over the messages must not.
func solidSurface(theme ui.Theme) draw2d.Color {
	surface, ground := theme.Surface, theme.Background
	var solid draw2d.Color
	for channel := range 3 {
		solid[channel] = ground[channel] + (surface[channel]-ground[channel])*surface[3]
	}
	solid[3] = 1
	return solid
}
