package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/akrck02/LLM-CP/chat/conversation"
	"github.com/akrck02/LLM-CP/protocol"
)

// reconnectDelay is how long the relay waits before watching the hub again after losing it.
const reconnectDelay = 2 * time.Second

// pageRoom is how many messages a page can fall behind before the relay lets it go; the page
// reconnects and gets everything again.
const pageRoom = 512

// pageEvent is one server-sent event: a message, or the hub's state under the name "state".
type pageEvent struct {
	name string // "" for a message
	data any
}

// relay keeps every message the hub published and its newest state, and hands each new event
// to every open page.
type relay struct {
	mutex    sync.Mutex
	messages []conversation.Message
	state    *conversation.State
	pages    map[chan pageEvent]bool
}

func newRelay() *relay {
	return &relay{pages: map[chan pageEvent]bool{}}
}

// followHub watches the hub for as long as the program runs, coming back when the hub restarts.
func (relay *relay) followHub(socket string) {
	for {
		err := relay.watch(socket)
		slog.Warn("lost the hub; trying again", "socket", socket, "error", err)
		time.Sleep(reconnectDelay)
	}
}

// watch reads the hub's stream until it ends. A restarted hub numbers its messages from 1 again,
// so what was kept is forgotten when the stream starts.
func (relay *relay) watch(socket string) error {
	connection, err := grpc.NewClient("unix://"+socket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer connection.Close()
	stream, err := protocol.NewHubClient(connection).Watch(context.Background(), &protocol.WatchRequest{Past: true})
	if err != nil {
		return err
	}
	first := true
	for {
		event, err := stream.Recv()
		if err != nil {
			return err
		}
		if first {
			relay.forget()
			first = false
		}
		if message := event.GetMessage(); message != nil {
			relay.publish(fromHub(message))
		}
		if state := event.GetState(); state != nil {
			relay.changeState(stateFromHub(state))
		}
	}
}

// fromHub is a message as the page reads it.
func fromHub(message *protocol.Message) conversation.Message {
	return conversation.Message{
		Number: message.Number, Topic: message.Topic, From: message.From, Text: message.Text,
		Published: time.Unix(message.Published, 0),
	}
}

// stateFromHub is the hub's state as the page reads it.
func stateFromHub(state *protocol.State) conversation.State {
	read := conversation.State{Archived: state.Archived}
	for _, contact := range state.Contacts {
		read.Contacts = append(read.Contacts, conversation.Contact{
			Name: contact.Name, Topics: contact.Topics, Groups: contact.Groups, TypingUntil: contact.TypingUntil,
		})
	}
	for _, kind := range state.Kinds {
		read.Kinds = append(read.Kinds, conversation.TopicKind{Topic: kind.Topic, Kind: kind.Kind, Publishers: kind.Publishers})
	}
	for _, group := range state.Groups {
		read.Groups = append(read.Groups, conversation.Group{Name: group.Name, Theme: group.Theme, Members: group.Members})
	}
	return read
}

// forget drops what was kept, for a hub that started again.
func (relay *relay) forget() {
	relay.mutex.Lock()
	defer relay.mutex.Unlock()
	relay.messages = nil
	relay.state = nil
}

// publish keeps a message and hands it to every open page.
func (relay *relay) publish(message conversation.Message) {
	relay.mutex.Lock()
	defer relay.mutex.Unlock()
	relay.messages = append(relay.messages, message)
	relay.tell(pageEvent{data: message})
}

// changeState keeps the hub's newest state and hands it to every open page.
func (relay *relay) changeState(state conversation.State) {
	relay.mutex.Lock()
	defer relay.mutex.Unlock()
	relay.state = &state
	relay.tell(pageEvent{name: stateEvent, data: state})
}

// stateEvent names the server-sent events that carry the hub's state.
const stateEvent = "state"

// tell hands an event to every page that has room for it. The caller holds the mutex.
func (relay *relay) tell(event pageEvent) {
	for page := range relay.pages {
		select {
		case page <- event:
		default: // a page this far behind is closed below and reconnects
			delete(relay.pages, page)
			close(page)
		}
	}
}

// join gives a new page everything kept so far, the state first, and the channel where the
// next events arrive.
func (relay *relay) join() ([]pageEvent, chan pageEvent) {
	relay.mutex.Lock()
	defer relay.mutex.Unlock()
	page := make(chan pageEvent, pageRoom)
	relay.pages[page] = true
	kept := make([]pageEvent, 0, len(relay.messages)+1)
	if relay.state != nil {
		kept = append(kept, pageEvent{name: stateEvent, data: *relay.state})
	}
	for _, message := range relay.messages {
		kept = append(kept, pageEvent{data: message})
	}
	return kept, page
}

// leave forgets a page that closed.
func (relay *relay) leave(page chan pageEvent) {
	relay.mutex.Lock()
	defer relay.mutex.Unlock()
	if relay.pages[page] {
		delete(relay.pages, page)
		close(page)
	}
}

// serveEvents streams the events to one page as server-sent events, one JSON value each.
func (relay *relay) serveEvents(response http.ResponseWriter, request *http.Request) {
	flusher, canFlush := response.(http.Flusher)
	if !canFlush {
		http.Error(response, "streaming is not supported", http.StatusInternalServerError)
		return
	}
	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache")
	kept, page := relay.join()
	defer relay.leave(page)
	for _, event := range kept {
		if writeEvent(response, event) != nil {
			return
		}
	}
	flusher.Flush()
	for {
		select {
		case event, open := <-page:
			if !open || writeEvent(response, event) != nil {
				return
			}
			flusher.Flush()
		case <-request.Context().Done():
			return
		}
	}
}

// writeEvent writes one event as a server-sent event, named when it is not a message.
func writeEvent(response http.ResponseWriter, event pageEvent) error {
	encoded, err := json.Marshal(event.data)
	if err != nil {
		return err
	}
	if event.name != "" {
		if _, err := fmt.Fprintf(response, "event: %s\n", event.name); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(response, "data: %s\n\n", encoded)
	return err
}
