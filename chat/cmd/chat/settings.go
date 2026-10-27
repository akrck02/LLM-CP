package main

import (
	"log/slog"

	"github.com/akrck02/bubble"
	"github.com/akrck02/bubble/goui"
	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"
	"github.com/akrck02/go-ui/window"
)

// modePreference is where the person's choice of night or day is kept between runs.
const (
	modePreference = "mode"
	lightMode      = "light"
)

// settings is the settings section: the appearance, and the server the app is linked to.
func (chat *screen) settings(context *ui.Context) {
	theme := context.Theme
	context.Box(ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Padding: ui.PadXY(12, 10), Gap: 12}, func() {
		context.Text("Appearance", ui.TextStyle{Weight: draw2d.Bold, Color: theme.InkSoft})
		dark := chat.mode == bubble.ModeDark
		if context.Toggle("Dark mode", &dark) {
			chat.chooseMode(dark)
		}
		context.Text("Server", ui.TextStyle{Weight: draw2d.Bold, Color: theme.InkSoft})
		chat.serverSettings(context)
	})
}

// chooseMode switches between night and day, and keeps the choice for the next run.
func (chat *screen) chooseMode(dark bool) {
	mode, kept := bubble.ModeDark, ""
	if !dark {
		mode, kept = bubble.ModeLight, lightMode
	}
	if err := chat.useMode(mode); err != nil {
		slog.Warn("chat: the mode could not change", "error", err)
		return
	}
	if err := window.KeepPreference(modePreference, kept); err != nil {
		slog.Warn("chat: the mode will be dark again next time", "error", err)
	}
}

// recallMode uses the mode chosen last time, once, in the first frame.
func (chat *screen) recallMode() {
	if chat.modeRecalled {
		return
	}
	chat.modeRecalled = true
	if kept, found := window.Preference(modePreference); found && kept == lightMode {
		if err := chat.useMode(bubble.ModeLight); err != nil {
			slog.Warn("chat: the light mode could not be used", "error", err)
		}
	}
}

// useMode makes the page's colours for night or day.
func (chat *screen) useMode(mode bubble.Mode) error {
	design, err := chatDesign(mode)
	if err != nil {
		return err
	}
	theme := goui.Theme(design)
	// the ground behind the panes is nearly even, so blurring it changes nothing to the eye, and
	// blurring it every frame made phones stutter
	theme.BackdropBlur = 0
	chat.mode, chat.design, chat.theme = mode, design, theme
	showMode(mode)
	return nil
}
