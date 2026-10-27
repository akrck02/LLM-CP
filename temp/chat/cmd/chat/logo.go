package main

import (
	"bytes"
	_ "embed"
	"image/png"
	"log/slog"

	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"
)

// logoPicture is LLM-CP's logo (docs/images/logo.svg), drawn once into a PNG for the header.
//
//go:embed logo.png
var logoPicture []byte

// logoSize is how big the logo is beside the name, in pixels.
const logoSize = 34

// logo draws LLM-CP's logo; its picture is decoded the first time and kept.
func (chat *screen) logo(context *ui.Context) {
	if chat.logoImage == nil {
		picture, err := png.Decode(bytes.NewReader(logoPicture))
		if err != nil {
			slog.Warn("chat: the logo could not be read", "error", err)
			return
		}
		chat.logoImage = draw2d.NewImage(picture)
	}
	context.Image(chat.logoImage, ui.Layout{Width: ui.Fixed(logoSize), Height: ui.Fixed(logoSize)}, 0)
}
