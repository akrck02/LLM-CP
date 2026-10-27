//go:build js && wasm

package main

import (
	"encoding/json"
	"log/slog"
	"net/url"
	"syscall/js"

	"github.com/akrck02/bubble"

	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"
	"github.com/akrck02/go-ui/window"

	"github.com/akrck02/LLM-CP/chat/conversation"
)

// link is what the page needs of its own to reach the server: nothing, the browser does it.
type link struct{}

// startLink starts listening as soon as the program starts: the page asked the name already.
func (chat *screen) startLink() {
	chat.me = pageName()
	chat.reduceMotion = js.Global().Call("matchMedia", "(prefers-reduced-motion: reduce)").Get("matches").Bool()
	chat.listenToSystemMode()
	listenToEvents("events", chat.receive, chat.receiveState, chat.connected)
	chat.recallSubscriptions()
}

// recallSubscriptions asks the server which topics the person follows, away from the frame.
// Until it answers, or if it cannot, every topic is listed.
func (chat *screen) recallSubscriptions() {
	if chat.me == "" {
		return
	}
	asked := js.Global().Call("fetch", "subscriptions?as="+url.QueryEscape(chat.me))
	asked.Call("then", js.FuncOf(func(_ js.Value, arguments []js.Value) any {
		return arguments[0].Call("text")
	})).Call("then", js.FuncOf(func(_ js.Value, arguments []js.Value) any {
		var topics []string
		if err := json.Unmarshal([]byte(arguments[0].String()), &topics); err != nil {
			slog.Warn("the server did not give the subscriptions; every topic is listed", "error", err)
			return nil
		}
		chat.receiveSubscriptions(topics)
		return nil
	}))
}

// listenToSystemMode follows the browser's night or day as it changes, while the person has not
// chosen one in the settings.
func (chat *screen) listenToSystemMode() {
	query := js.Global().Call("matchMedia", "(prefers-color-scheme: dark)")
	query.Call("addEventListener", "change", js.FuncOf(func(_ js.Value, arguments []js.Value) any {
		chat.mutex.Lock()
		chat.followSystemMode(arguments[0].Get("matches").Bool())
		chat.mutex.Unlock()
		window.Wake()
		return nil
	}))
}

// act asks the server for an action at a path (groups, archive, restore, subscriptions), away
// from the frame; the hub's next state shows what came of it.
func (chat *screen) act(path string, asked conversation.Action) {
	body, err := json.Marshal(asked)
	if err != nil {
		slog.Warn("an action that cannot be written", "error", err)
		return
	}
	options := map[string]any{"method": "POST", "body": string(body), "headers": map[string]any{"Content-Type": "application/json"}}
	js.Global().Call("fetch", path, options)
}

// drawSetup draws nothing in a page: there is nothing to set up.
func (chat *screen) drawSetup(*ui.Context) bool { return false }

// composerHeight is the room left at the foot of the open chat for the page's bar, in pixels.
const composerHeight = 64

// composerSlot names the room left for the page's bar, to find where it was laid out.
const composerSlot = "composer slot"

// composer leaves room at the foot of the open chat, where the page's own bar is laid over the
// canvas (see index.html): Go UI cannot open a phone's on-screen keyboard in a browser yet.
// Clicking the room gives the bar the keyboard.
func (chat *screen) composer(context *ui.Context) {
	flat := ui.Look{}
	layout := ui.Layout{Width: ui.Grow(), Height: ui.Fixed(composerHeight)}
	if context.Clickable(composerSlot, layout, ui.Looks{Normal: flat, Hovered: flat, Pressed: flat}, func() {}) {
		js.Global().Call("llmcpFocusBar")
	}
	chat.slot, chat.slotFound = context.Area("screen", composerSlot)
}

// afterFrame shows the page's bar over the room left for it when a chat is on screen, and tells
// it which chat it writes to.
func (chat *screen) afterFrame(narrow bool) {
	topic := chat.open
	if narrow && chat.showingList || !chat.slotFound {
		topic = ""
	}
	if topic != chat.writingTo || chat.slot != chat.barPlace {
		chat.writingTo, chat.barPlace = topic, chat.slot
		writeTo(topic, chat.slot)
	}
	chat.slotFound = false // found again in the next frame, or the bar hides
}

// listenToEvents opens the browser's EventSource on a path and hands each message to receive, and
// each of the hub's states to receiveState. The browser reconnects by itself when the server goes
// away; connected says when it is on or off.
func listenToEvents(path string, receive func(conversation.Message), receiveState func(conversation.State), connected func(bool)) {
	source := js.Global().Get("EventSource").New(path)
	source.Call("addEventListener", "state", js.FuncOf(func(_ js.Value, arguments []js.Value) any {
		var state conversation.State
		if err := json.Unmarshal([]byte(arguments[0].Get("data").String()), &state); err != nil {
			slog.Warn("a state event that is not a state", "error", err)
			return nil
		}
		receiveState(state)
		return nil
	}))
	source.Call("addEventListener", "message", js.FuncOf(func(_ js.Value, arguments []js.Value) any {
		var message conversation.Message
		if err := json.Unmarshal([]byte(arguments[0].Get("data").String()), &message); err != nil {
			slog.Warn("an event that is not a message", "error", err)
			return nil
		}
		receive(message)
		return nil
	}))
	source.Call("addEventListener", "open", js.FuncOf(func(js.Value, []js.Value) any {
		connected(true)
		return nil
	}))
	source.Call("addEventListener", "error", js.FuncOf(func(js.Value, []js.Value) any {
		slog.Warn("the server's events stopped; the browser tries again", "state", source.Get("readyState").Int())
		connected(false)
		return nil
	}))
}

// pageName is the name the page writes as, which it asked for before the program started.
func pageName() string {
	name := js.Global().Get("llmcpName")
	if name.Type() != js.TypeString {
		return ""
	}
	return name.String()
}

// writeTo shows the page's bar to write in the chat of a topic over a place of the canvas, or
// hides it for "".
func writeTo(topic string, place draw2d.Rectangle) {
	js.Global().Call("llmcpWriteTo", topic, place.X, place.Y, place.Width, place.Height)
}

// serverSettings says where the page comes from: the server that served it is the one it talks to.
func (chat *screen) serverSettings(context *ui.Context) {
	context.Text("This page talks to the server that served it, as "+chat.me+".", ui.TextStyle{Color: context.Theme.InkSoft})
}

// showMode tells the page's own parts (the bar written in HTML) whether it is night or day.
func showMode(mode bubble.Mode) {
	js.Global().Get("document").Get("body").Get("dataset").Set("mode", string(mode))
}
