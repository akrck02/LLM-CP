package conversation

import (
	"slices"
	"testing"
)

// pairedLog is a log read by alice: her pair with api, a pair between api and docs, an inbox
// message from before pairs, and a topic.
func pairedLog() *Log {
	var log Log
	log.Add(message(1, "@alice", "bob", "from before pairs"))
	log.Add(message(2, "alice+api", "api", "may I change the api?"))
	log.Add(message(3, "alice+api", "alice", "yes"))
	log.Add(message(4, "api+docs", "docs", "the api changed?"))
	log.Add(message(5, "release", "carol", "0.2 is out"))
	log.SetState(State{Groups: []Group{
		{Name: "alice+api", Members: []string{"alice", "api"}},
		{Name: "api+docs", Members: []string{"api", "docs"}},
	}})
	log.SetSubscriptions("alice", []string{"release"})
	return &log
}

// Both sides of a conversation are one chat, named after the one the person talks to.
func TestAPairIsOneConversationNamedAfterTheOther(test *testing.T) {
	log := pairedLog()
	if sessions := topicsOf(log.Sessions("alice")); !slices.Equal(sessions, []string{"alice+api", "@alice"}) {
		test.Fatalf("the sessions are %v", sessions)
	}
	if chat := log.Open("alice+api"); len(chat.Messages) != 2 || Name(chat.Topic, "alice") != "api" {
		test.Fatalf("the pair holds %d messages, named %q", len(chat.Messages), Name(chat.Topic, "alice"))
	}
	if topics := topicsOf(log.Topics()); !slices.Equal(topics, []string{"release"}) {
		test.Fatalf("a pair is no topic: %v", topics)
	}
}

// The hub refuses a third voice in a pair, so the chat offers none: the person may only read it.
func TestAPairWithoutMeIsReadOnly(test *testing.T) {
	log := pairedLog()
	if between := topicsOf(log.BetweenSessions("alice")); !slices.Equal(between, []string{"api+docs"}) {
		test.Fatalf("between sessions: %v", between)
	}
	if !IsReadOnly("api+docs", "alice") || IsReadOnly("alice+api", "alice") || IsReadOnly("release", "alice") {
		test.Fatal("only a pair without the person is read-only")
	}
	if Name("api+docs", "alice") != "api + docs" {
		test.Fatalf("named %q", Name("api+docs", "alice"))
	}
}

// Opening a contact goes to the pair, which the first message makes; before the person's name is
// known, to the contact's inbox, which the hub files in the pair as well.
func TestAContactIsReachedThroughThePair(test *testing.T) {
	if ConversationWith("api", "alice") != "alice+api" || ConversationWith("api", "") != "@api" {
		test.Fatalf("got %q and %q", ConversationWith("api", "alice"), ConversationWith("api", ""))
	}
}
