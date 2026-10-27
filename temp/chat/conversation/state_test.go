package conversation

import (
	"slices"
	"testing"
	"time"
)

var noon = time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)

// logWith is a log with one message in each of the topics given.
func logWith(topics ...string) *Log {
	log := &Log{}
	for index, topic := range topics {
		log.Add(Message{Number: uint64(index + 1), Topic: topic, From: "alice", Text: "hello"})
	}
	return log
}

// topicsOf are the chats' topics, in order.
func topicsOf(chats []*Chat) []string {
	topics := make([]string, 0, len(chats))
	for _, chat := range chats {
		topics = append(topics, chat.Topic)
	}
	return topics
}

func TestAnArchivedChatKeepsItsMessagesOutOfTheList(test *testing.T) {
	log := logWith("api", "docs")
	log.SetState(State{Archived: []string{"api"}})
	if got := topicsOf(log.Chats()); !slices.Equal(got, []string{"docs"}) {
		test.Fatalf("the list holds %v, want [docs]", got)
	}
	if got := topicsOf(log.Archived()); !slices.Equal(got, []string{"api"}) {
		test.Fatalf("the archive holds %v, want [api]", got)
	}
	log.Add(Message{Number: 3, Topic: "api", From: "bob", Text: "still here"})
	if got := topicsOf(log.Chats()); !slices.Equal(got, []string{"docs"}) {
		test.Fatalf("a new message brought an archived chat back: %v", got)
	}
}

func TestContactsListEverySessionWithItsState(test *testing.T) {
	log := &Log{}
	log.SetState(State{Contacts: []Contact{
		{Name: "alice", Groups: []string{"api"}},
		{Name: "bob", TypingUntil: noon.Add(time.Second).UnixMilli()},
	}})
	contacts := log.Contacts()
	if len(contacts) != 2 || contacts[0].Name != "alice" || contacts[1].Name != "bob" {
		test.Fatalf("contacts are %+v", contacts)
	}
	if contacts[0].IsTyping(noon) || !contacts[1].IsTyping(noon) {
		test.Fatal("only bob should be typing at noon")
	}
	if contacts[1].IsTyping(noon.Add(2 * time.Second)) {
		test.Fatal("bob is still typing after his time ran out")
	}
}

func TestTypingShowsOnlyInTheMembersChats(test *testing.T) {
	log := logWith("api", "docs")
	typingUntil := noon.Add(time.Minute).UnixMilli()
	log.SetState(State{
		Contacts: []Contact{{Name: "bob", Topics: []string{"@bob", "api"}, TypingUntil: typingUntil}},
		Groups:   []Group{{Name: "api", Members: []string{"alice", "bob"}}},
	})
	if got := log.TypingIn("api", noon); !slices.Equal(got, []string{"bob"}) {
		test.Fatalf("typing in api: %v, want [bob]", got)
	}
	if got := log.TypingIn("docs", noon); len(got) != 0 {
		test.Fatalf("bob does not listen to docs, yet shows typing there: %v", got)
	}
	if got := log.TypingIn("@bob", noon); !slices.Equal(got, []string{"bob"}) {
		test.Fatalf("typing in bob's own inbox: %v, want [bob]", got)
	}
}

func TestADeclaredTopicIsAChatWithNobodyListening(test *testing.T) {
	var log Log
	log.SetState(State{Kinds: []TopicKind{{Topic: "plans", Kind: "conversation"}}})
	if got := topicsOf(log.Chats()); !slices.Equal(got, []string{"plans"}) {
		test.Fatalf("the list holds %v, want plans", got)
	}
}

func TestGroupsAndListenedTopicsAreChatsBeforeAnyoneWrites(test *testing.T) {
	log := logWith("api")
	log.SetState(State{
		Contacts: []Contact{{Name: "bob", Topics: []string{"@bob", "docs", "api"}}},
		Groups:   []Group{{Name: "release", Members: []string{"bob"}}},
	})
	if got := topicsOf(log.Chats()); !slices.Equal(got, []string{"api", "docs", "release"}) {
		test.Fatalf("the list holds %v, want api (written) and then docs and release, quiet", got)
	}
}
