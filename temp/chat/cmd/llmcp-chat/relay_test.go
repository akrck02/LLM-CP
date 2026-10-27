package main

import (
	"testing"

	"github.com/akrck02/LLM-CP/chat/conversation"
)

func TestALatePageGetsTheLatestContacts(test *testing.T) {
	relay := newRelay()
	relay.changeState(conversation.State{Contacts: []conversation.Contact{{Name: "alice"}}})
	relay.publish(conversation.Message{Number: 1, Topic: "api", From: "alice", Text: "hello"})
	relay.changeState(conversation.State{Contacts: []conversation.Contact{{Name: "alice"}, {Name: "bob"}}})
	kept, page := relay.join()
	defer relay.leave(page)
	if len(kept) != 2 || kept[0].name != stateEvent {
		test.Fatalf("a late page got %+v, want the state and then the message", kept)
	}
	state := kept[0].data.(conversation.State)
	if len(state.Contacts) != 2 {
		test.Fatalf("a late page got the contacts %+v, want the newest two", state.Contacts)
	}
}
