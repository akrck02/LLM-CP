package conversation

import (
	"slices"
	"testing"
)

func message(number uint64, topic, from, text string) Message {
	return Message{Number: number, Topic: topic, From: from, Text: text}
}

func TestTheChatWithTheNewestMessageComesFirst(test *testing.T) {
	var log Log
	log.Add(message(1, "api", "alice", "one"))
	log.Add(message(2, "docs", "bob", "two"))
	log.Add(message(3, "api", "bob", "three"))
	topics := []string{}
	for _, chat := range log.Chats() {
		topics = append(topics, chat.Topic)
	}
	if !slices.Equal(topics, []string{"api", "docs"}) {
		test.Fatalf("got %v", topics)
	}
}

func TestAMessageSentTwiceIsShownOnce(test *testing.T) {
	var log Log
	log.Add(message(1, "api", "alice", "one"))
	log.Add(message(1, "api", "alice", "one"))
	if chat := log.Open("api"); len(chat.Messages) != 1 {
		test.Fatalf("got %d messages", len(chat.Messages))
	}
}

func TestOpeningAChatMarksItsMessagesAsRead(test *testing.T) {
	var log Log
	log.Add(message(1, "api", "alice", "one"))
	log.Add(message(2, "api", "bob", "two"))
	if unread := log.Chats()[0].Unread; unread != 2 {
		test.Fatalf("unread before opening: %d", unread)
	}
	log.Open("api")
	log.Add(message(3, "api", "alice", "three"))
	chat := log.Chats()[0]
	if chat.Unread != 1 || !slices.Equal(chat.Members, []string{"alice", "bob"}) {
		test.Fatalf("got unread %d, members %v", chat.Unread, chat.Members)
	}
}

func TestInitialsStandForAName(test *testing.T) {
	cases := map[string]string{"go-docs": "GD", "alice": "A", "@bob": "B", "projects-general-x": "PG", "": "?"}
	for name, want := range cases {
		if got := Initials(name); got != want {
			test.Errorf("%q: got %q, want %q", name, got, want)
		}
	}
}

func TestANameKeepsItsColour(test *testing.T) {
	if ColorIndex("alice", 7) != ColorIndex("alice", 7) {
		test.Fatal("the same name got two colours")
	}
}

func TestWhatWasReadStaysReadWhenTheMarksAreTakenBack(test *testing.T) {
	var earlier Log
	earlier.Add(Message{Number: 1, Topic: "api", From: "alice"})
	earlier.Add(Message{Number: 2, Topic: "api", From: "bob"})
	earlier.Open("api")

	var later Log
	later.SetReadMarks(earlier.ReadMarks())
	later.Add(Message{Number: 1, Topic: "api", From: "alice"})
	later.Add(Message{Number: 2, Topic: "api", From: "bob"})
	later.Add(Message{Number: 3, Topic: "api", From: "alice"})
	if unread := later.Chats()[0].Unread; unread != 1 {
		test.Errorf("unread = %d, want 1: only the message after the last one read", unread)
	}
}
