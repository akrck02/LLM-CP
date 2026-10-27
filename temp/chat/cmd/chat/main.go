// Command chat shows the hub's topics as the chats of a messaging app, drawn with Go UI in
// Bubble 4's style and filled live from llmcp-chat's events, and lets a person write to them.
// The same program is the page llmcp-chat serves (built for WebAssembly) and a phone's app
// (built for Android, which asks for the server's address); chat/build.sh builds both.
//
// The person's messages are on the right. On a narrow screen, the list and the open chat take turns.
package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/akrck02/bubble"
	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"
	"github.com/akrck02/go-ui/window"

	"github.com/akrck02/LLM-CP/chat/conversation"
)

// accent is LLM-CP's colour, the one its docs are published with.
const accent = "#00897B"

// screen is what the page shows: the chats, the one open, and whose messages are one's own.
type screen struct {
	mutex  sync.Mutex // the events arrive on another goroutine than the frames
	log    conversation.Log
	open   string
	me     string
	online bool

	narrow      bool    // a phone's screen, or a window as narrow: one pane at a time, filling it
	showingList bool    // on a narrow screen, the list of chats instead of the open chat
	section     section // what the side pane shows
	// archivedFrom is the tab the archived chats were opened from, to go back to
	archivedFrom section
	creating     newGroup
	leaving      string // the chat sliding out of the list to be archived, "" when none
	// reduceMotion is the person's wish, told by their system, for things not to move
	reduceMotion bool
	writingTo    string // the topic the page's bar writes to, "" when it is hidden
	// where the page's bar goes: the room left for it in the last frame, and where it was put
	slot, barPlace draw2d.Rectangle
	slotFound      bool
	bubbleWidth    float32 // the widest a message's bubble gets on this screen

	link link // what this platform needs to reach the server
	// shownCount is how many messages each chat draws, once the person asked for earlier ones
	shownCount map[string]int

	design bubble.Theme // glass's colours, the ground's among them
	theme  ui.Theme     // the same, as Go UI draws it
	mode   bubble.Mode  // night or day

	logoImage *draw2d.Image // the header's logo, decoded once

	readingOf   string            // the name whose reading was recalled
	readingKept map[string]uint64 // the read marks as last kept
	// modeRecalled is whether the mode the person chose last time was looked for: in the first
	// frame, once the platform has given the program its storage
	modeRecalled bool
	// modeChosen is whether the person chose night or day; until then the chat follows the system
	modeChosen bool
}

func main() {
	chat := &screen{showingList: true, shownCount: map[string]int{}}
	if err := chat.useMode(bubble.ModeDark); err != nil {
		fmt.Fprintln(os.Stderr, "chat:", err)
		os.Exit(1)
	}
	chat.startLink()
	options := window.Options{Title: "LLM-CP chat", Canvas: "view"}
	err := window.Run(options, func(context *ui.Context) {
		chat.recallMode()
		context.Theme = chat.theme
		chat.draw(context)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "chat:", err)
		os.Exit(1)
	}
}

// chatDesign is Bubble 4's paper style, by night or by day, with the glass finish and LLM-CP's accent.
func chatDesign(mode bubble.Mode) (bubble.Theme, error) {
	colour, err := bubble.ParseColor(accent)
	if err != nil {
		return bubble.Theme{}, err
	}
	return bubble.NewTheme(bubble.Options{
		Style: bubble.StylePaper, Mode: mode, Density: bubble.DensityComfortable, Accent: colour, Glass: true,
	})
}

// receive adds a message from the server and asks for a frame to show it.
func (chat *screen) receive(message conversation.Message) {
	chat.mutex.Lock()
	chat.online = true // a message that arrives is proof enough, whatever the browser said
	chat.log.Add(message)
	chat.openFirstChat(message.Topic)
	chat.mutex.Unlock()
	window.Wake()
}

// openFirstChat opens the topic of the first message that arrives, unless a chat is open or the
// topic carries metrics: a conversation is a better first sight than numbers.
func (chat *screen) openFirstChat(topic string) {
	if chat.open == "" && !chat.log.IsMetrics(topic) {
		chat.open = topic
	}
}

// receiveState takes the hub's newest state and asks for a frame to show it.
func (chat *screen) receiveState(state conversation.State) {
	chat.mutex.Lock()
	chat.log.SetState(state)
	chat.mutex.Unlock()
	window.Wake()
}

// receiveSubscriptions takes the topics the person is subscribed to, as the server keeps them;
// from then on the Topics tab lists only those.
func (chat *screen) receiveSubscriptions(topics []string) {
	chat.mutex.Lock()
	chat.log.SetSubscriptions(chat.me, topics)
	chat.mutex.Unlock()
	window.Wake()
}

// setSubscribed subscribes the person to a topic or unsubscribes them, showing it at once,
// without waiting for the server.
func (chat *screen) setSubscribed(topic string, subscribed bool) {
	chat.log.SetSubscriptions(chat.me, chat.log.WithSubscription(topic, subscribed))
	chat.act("subscriptions", conversation.Action{As: chat.me, Topic: topic, Subscribed: subscribed})
}

// connected remembers whether the server's events are flowing, for the header to say so.
func (chat *screen) connected(online bool) {
	chat.mutex.Lock()
	chat.online = online
	chat.mutex.Unlock()
	window.Wake()
}
