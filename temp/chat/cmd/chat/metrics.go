package main

import (
	"math"
	"strconv"
	"strings"

	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"

	"github.com/akrck02/LLM-CP/chat/conversation"
)

// metricsIcon marks a topic that carries numbers instead of a conversation.
const metricsIcon = "binary"

// Measures of a metrics topic's view, in pixels.
const (
	seriesChartHeight = 90
	seriesLineWidth   = 2
	seriesValueSize   = 26
	seriesCardWidth   = 300 // about: the cards share a row out between as many as fit
)

// metricsAvatar is a metrics topic's picture: its icon instead of initials.
func metricsAvatar(context *ui.Context, size float32) {
	theme := context.Theme
	layout := ui.Layout{Width: ui.Fixed(size), Height: ui.Fixed(size), AlignX: ui.Center, AlignY: ui.Center}
	context.Frame(layout, ui.Look{Fill: theme.PrimaryTint, Radius: size / 2}, func() {
		context.IconSized(metricsIcon, size*0.5, theme.Primary)
	})
}

// seriesCount is a metrics topic's line in the list: how many series it has.
func seriesCount(series []conversation.Series) string {
	switch len(series) {
	case 0:
		return "No points yet"
	case 1:
		return "1 series · " + series[0].Latest().Shown()
	}
	return strconv.Itoa(len(series)) + " series"
}

// metricsView is an open metrics topic: a card per series with its newest value and its line.
// Nobody writes on a metrics topic by hand, so it has no composer.
func (chat *screen) metricsView(context *ui.Context, topic string, narrow bool) {
	series := chat.log.SeriesIn(topic)
	width, _ := context.Size()
	columns := 1
	if !narrow {
		columns = max(1, int((width-chatListWidth-4*paneGap)/seriesCardWidth))
	}
	area := ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Height: ui.Grow(), Padding: ui.Pad(messagesSidePadding)}
	context.ScrollArea("metrics "+topic, area, func() {
		if len(series) == 0 {
			context.Text("No points yet: publish them with llmcp metric.", ui.TextStyle{Color: context.Theme.InkSoft})
			return
		}
		context.Grid(columns, func() {
			for _, each := range series {
				seriesCard(context, each)
			}
		})
	})
}

// seriesCard is one series: its name, its newest value with its unit, and its line over time.
func seriesCard(context *ui.Context, series conversation.Series) {
	theme := context.Theme
	latest := series.Latest()
	layout := ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Padding: ui.Pad(14), Gap: 6}
	context.Frame(layout, ui.Look{Fill: theme.Sunken, Radius: 14}, func() {
		context.Text(shortened(latest.Shown(), membersLetters), ui.TextStyle{Size: theme.SmallTextSize, Color: theme.InkSoft})
		context.Text(withUnit(latest.Value, latest.Unit), ui.TextStyle{Size: seriesValueSize, Weight: draw2d.Bold})
		context.Paint(ui.Layout{Width: ui.Grow(), Height: ui.Fixed(seriesChartHeight)}, func(area draw2d.Rectangle, drawing *draw2d.List) {
			drawSeries(drawing, area, series, colourOf(series.Name), theme.Line)
		})
	})
}

// drawSeries draws a series' points from its first to its newest, time across and value up, over
// a base line; a single point is a dot.
func drawSeries(drawing *draw2d.List, area draw2d.Rectangle, series conversation.Series, colour, baseColour draw2d.Color) {
	drawing.Rectangle(draw2d.Rectangle{X: area.X, Y: area.Y + area.Height - 1, Width: area.Width, Height: 1}, 0, baseColour, draw2d.Color{}, 0)
	first, last := series.Points[0].At, series.Latest().At
	lowest, highest := valueBounds(series)
	var path draw2d.Path
	for index, point := range series.Points {
		across := float32(0.5)
		if last > first {
			across = float32(point.At-first) / float32(last-first)
		}
		up := float32((point.Value - lowest) / (highest - lowest))
		x, y := area.Width*across, (area.Height-seriesLineWidth)*(1-up)
		if len(series.Points) == 1 {
			path.Circle(x, y, seriesLineWidth)
			break
		}
		if index == 0 {
			path.MoveTo(x, y)
			continue
		}
		path.LineTo(x, y)
	}
	drawing.Stroke(&path, area.X, area.Y, 1, seriesLineWidth, colour)
}

// valueBounds are a series' lowest and highest values, apart by at least one so a flat line sits
// in the middle rather than dividing by zero.
func valueBounds(series conversation.Series) (float64, float64) {
	lowest, highest := math.Inf(1), math.Inf(-1)
	for _, point := range series.Points {
		lowest, highest = min(lowest, point.Value), max(highest, point.Value)
	}
	if highest-lowest < 1 {
		middle := (lowest + highest) / 2
		return middle - 0.5, middle + 0.5
	}
	return lowest, highest
}

// withUnit writes a value with at most two decimals, and its unit.
func withUnit(value float64, unit string) string {
	written := strconv.FormatFloat(math.Round(value*100)/100, 'f', -1, 64)
	return strings.TrimSpace(written + " " + unit)
}
