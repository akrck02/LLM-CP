//go:build !js

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/akrck02/bubble"

	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"
	"github.com/akrck02/go-ui/window"

	"github.com/akrck02/LLM-CP/chat/conversation"
)

// reconnectDelay is how long the app waits before asking the server again after losing it.
const reconnectDelay = 2 * time.Second

// The starts of the lines of a server-sent event: its name, when it has one, and its data.
const (
	namePrefix = "event: "
	dataPrefix = "data: "
)

// link is what an app needs to reach llmcp-chat: its address, the person's name, and what they
// are writing.
type link struct {
	server string // like http://192.168.1.10:8090
	name   string
	ready  bool
	draft  string
	failed string // why the last message could not be sent, shown until the next one
	// recalled is whether the last server and name were looked for: in the first frame, once
	// Android has given the app its folder
	recalled bool
	// following counts the connections made: a stream that belongs to an older one stops
	following int
}

// The preferences the app keeps the last server and name under, to connect on its own next time.
const (
	serverPreference = "server"
	namePreference   = "name"
)

// startLink waits for the first frame: recallLink needs the app's folder.
func (chat *screen) startLink() {}

// recallLink connects to the server and with the name of the last time, when there was one;
// otherwise drawSetup asks for them.
func (chat *screen) recallLink() {
	chat.link.recalled = true
	server, knownServer := window.Preference(serverPreference)
	name, knownName := window.Preference(namePreference)
	if !knownServer || !knownName {
		return
	}
	chat.link.server, chat.link.name = server, name
	chat.connect()
}

// drawSetup asks for the server's address and the person's name until both are given, and then
// starts listening. It tells whether it drew, so nothing else is drawn under it.
func (chat *screen) drawSetup(context *ui.Context) bool {
	if !chat.link.recalled {
		chat.recallLink()
	}
	if chat.link.ready {
		return false
	}
	theme := context.Theme
	pane := chat.paneLook(theme, false)
	context.Box(ui.Layout{Width: ui.Grow(), Height: ui.Grow(), AlignX: ui.Center, AlignY: ui.Center}, func() {
		context.Frame(ui.Layout{Direction: ui.Vertical, Width: ui.Fixed(setupWidth), Padding: ui.Pad(24), Gap: 12}, pane, func() {
			context.Text("LLM-CP", ui.TextStyle{Size: theme.HeadingSize, Weight: draw2d.Bold})
			context.Text("Where llmcp-chat runs, and your name.", ui.TextStyle{Color: theme.InkSoft})
			context.TextFieldWith("server", &chat.link.server, ui.TextFieldOptions{Placeholder: "http://192.168.1.10:8090"})
			submitted := context.EntryField("name", &chat.link.name, ui.TextFieldOptions{Placeholder: "Your name"})
			if (context.PrimaryButton("Connect") || submitted) && chat.link.server != "" && chat.link.name != "" {
				chat.connect()
			}
		})
	})
	return true
}

// connect starts listening to the server the person gave.
func (chat *screen) connect() {
	chat.link.server = strings.TrimRight(strings.TrimSpace(chat.link.server), "/")
	if !strings.Contains(chat.link.server, "://") {
		chat.link.server = "http://" + chat.link.server
	}
	chat.me, chat.link.ready = strings.TrimSpace(chat.link.name), true
	chat.rememberLink()
	chat.link.following++
	go chat.followServer(chat.link.server+"/events", chat.link.following)
	go chat.recallSubscriptions(chat.link.server + "/subscriptions?as=" + url.QueryEscape(chat.me))
}

// recallSubscriptions asks the server at an address which topics the person follows. Until it
// answers, or if it cannot, every topic is listed.
func (chat *screen) recallSubscriptions(address string) {
	response, err := http.Get(address)
	if err != nil {
		slog.Warn("the server did not give the subscriptions; every topic is listed", "error", err)
		return
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		slog.Warn("the server did not give the subscriptions; every topic is listed", "status", response.Status)
		return
	}
	var topics []string
	if err := json.NewDecoder(response.Body).Decode(&topics); err != nil {
		slog.Warn("the server did not give the subscriptions; every topic is listed", "error", err)
		return
	}
	chat.receiveSubscriptions(topics)
}

// composer is the bar to write in the open chat, under its messages.
func (chat *screen) composer(context *ui.Context) {
	theme := context.Theme
	context.Box(ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Padding: ui.PadXY(12, 10), Gap: 6}, func() {
		if chat.link.failed != "" {
			context.Text(chat.link.failed, ui.TextStyle{Size: theme.SmallTextSize, Color: theme.Danger})
		}
		chat.composerPill(context)
	})
}

// Measures of the bar to write, in pixels: a pill with the send button inside it.
const (
	pillHeight = 52
	sendSize   = 40
)

// composerPill is the field to write in and the round send button, inside one pill whose edge
// takes the accent while the field has the keyboard. The button lights up once there is
// something to send.
func (chat *screen) composerPill(context *ui.Context) {
	theme := context.Theme
	pill := ui.Look{Fill: theme.Sunken, Radius: pillHeight / 2, Border: theme.Line, BorderWidth: 1}
	if context.FieldFocused("draft") {
		pill.Border = theme.Primary
	}
	layout := ui.Layout{Width: ui.Grow(), Height: ui.Fixed(pillHeight), Padding: ui.Padding{Left: 18, Right: 6}, Gap: 6, AlignY: ui.Center}
	context.Frame(layout, pill, func() {
		submitted := context.EntryField("draft", &chat.link.draft, ui.TextFieldOptions{Placeholder: "Message " + conversation.Title(chat.open), Plain: true})
		ready := strings.TrimSpace(chat.link.draft) != ""
		button, ink := ui.Look{Fill: theme.Line, Radius: sendSize / 2}, theme.InkQuiet
		if ready {
			button.Fill, ink = theme.Primary, theme.OnPrimary
		}
		grown := context.Animate("send ready", boolToFloat(ready), chat.motion(tabMove))
		size := sendSize * (0.9 + 0.1*grown)
		sendLayout := ui.Layout{Width: ui.Fixed(size), Height: ui.Fixed(size), AlignX: ui.Center, AlignY: ui.Center}
		pressed := context.Clickable("send", sendLayout, ui.Looks{Normal: button, Hovered: button, Pressed: button}, func() {
			context.IconSized("send-horizontal", theme.IconSize-2, ink)
		})
		if (pressed || submitted) && ready {
			chat.send()
		}
	})
}

// send publishes the draft to the open chat, away from the frame, and empties it.
func (chat *screen) send() {
	text := strings.TrimSpace(chat.link.draft)
	if text == "" {
		return
	}
	chat.link.draft, chat.link.failed = "", ""
	body, err := json.Marshal(map[string]string{"as": chat.me, "topic": chat.open, "text": text})
	if err != nil {
		chat.link.failed = err.Error()
		return
	}
	address := chat.link.server + "/publish"
	go func() {
		failed := post(address, body)
		chat.mutex.Lock()
		chat.link.failed = failed
		chat.mutex.Unlock()
		window.Wake()
	}()
}

// act asks the server for an action at a path (groups, archive, restore, subscriptions), away
// from the frame; the hub's next state shows what came of it, and a failure shows under the bar
// to write.
func (chat *screen) act(path string, asked conversation.Action) {
	body, err := json.Marshal(asked)
	if err != nil {
		chat.link.failed = err.Error()
		return
	}
	address := chat.link.server + "/" + path
	go func() {
		failed := post(address, body)
		chat.mutex.Lock()
		chat.link.failed = failed
		chat.mutex.Unlock()
		window.Wake()
	}()
}

// post sends a message to the server; it gives why it failed, or "".
func post(address string, body []byte) string {
	response, err := http.Post(address, "application/json", bytes.NewReader(body))
	if err != nil {
		return "Not sent: " + err.Error()
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		answer, _ := io.ReadAll(response.Body) // the status alone says enough if this fails
		return "Not sent: " + strings.TrimSpace(string(answer))
	}
	return ""
}

// afterFrame has nothing to do in an app: its bar is drawn with the chat.
func (chat *screen) afterFrame(bool) {}

// followServer reads the server's events for as long as the app runs, coming back when it is lost.
func (chat *screen) followServer(address string, connection int) {
	for chat.isFollowing(connection) {
		err := chat.readEvents(address, connection)
		chat.connected(false)
		slog.Warn("lost the chat server; trying again", "address", address, "error", err)
		time.Sleep(reconnectDelay)
	}
}

// readEvents reads one stream of server-sent events: a message per "data:" line, or the hub's
// state when the line before named the event "state".
func (chat *screen) readEvents(address string, connection int) error {
	response, err := http.Get(address)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	chat.connected(true)
	lines := bufio.NewScanner(response.Body)
	lines.Buffer(make([]byte, 0, 64<<10), 1<<20)
	name := ""
	for lines.Scan() && chat.isFollowing(connection) {
		if named, isName := strings.CutPrefix(lines.Text(), namePrefix); isName {
			name = named
			continue
		}
		data, isData := strings.CutPrefix(lines.Text(), dataPrefix)
		if !isData {
			continue
		}
		chat.receiveEvent(name, []byte(data))
		name = ""
	}
	return lines.Err()
}

// receiveEvent reads one event's data: the hub's state, or a message.
func (chat *screen) receiveEvent(name string, data []byte) {
	if name == "state" {
		var state conversation.State
		if err := json.Unmarshal(data, &state); err != nil {
			slog.Warn("a state event that is not a state", "error", err)
			return
		}
		chat.receiveState(state)
		return
	}
	var message conversation.Message
	if err := json.Unmarshal(data, &message); err != nil {
		slog.Warn("an event that is not a message", "error", err)
		return
	}
	chat.receive(message)
}

// rememberLink keeps the server and the name for the next start. A phone that cannot keep them
// only asks again next time, so a failure is logged and the connection goes on.
func (chat *screen) rememberLink() {
	err := window.KeepPreference(serverPreference, chat.link.server)
	if err == nil {
		err = window.KeepPreference(namePreference, chat.me)
	}
	if err != nil {
		slog.Warn("chat: the server and name will be asked again next time", "error", err)
	}
}

// isFollowing tells whether a connection is still the app's: one left by signing out is not.
func (chat *screen) isFollowing(connection int) bool {
	chat.mutex.Lock()
	defer chat.mutex.Unlock()
	return chat.link.following == connection
}

// serverSettings shows the server and the name the app uses, and lets the person change them.
func (chat *screen) serverSettings(context *ui.Context) {
	theme := context.Theme
	context.Text(chat.link.server, ui.TextStyle{Color: theme.InkSoft})
	context.Text("as "+chat.me, ui.TextStyle{Color: theme.InkSoft})
	if context.Button("Change server") {
		chat.signOut()
	}
}

// signOut forgets the server and the name, drops what came from that server, and asks again.
// The fields keep what was typed, so a small change is quick.
func (chat *screen) signOut() {
	for _, name := range []string{serverPreference, namePreference} {
		if err := window.ForgetPreference(name); err != nil {
			slog.Warn("chat: the server may be used again next time", "error", err)
		}
	}
	chat.link.following++ // the old stream stops at its next line
	chat.link.ready, chat.online = false, false
	chat.log, chat.open, chat.showingList, chat.section = conversation.Log{}, "", true, sessionsSection
}

// showMode has nothing to tell in the app: everything on screen is drawn with the theme.
func showMode(bubble.Mode) {}
