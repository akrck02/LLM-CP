package hub

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"
)

// A messaging app keeps one conversation per pair of people: the first message makes it.
func TestTheFirstMessageToSomeoneCreatesThePair(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob")
	message, err := hub.Publish("bob", "@alice", "hello")
	if err != nil {
		test.Fatal(err)
	}
	groups := hub.Groups()
	if message.Topic != "alice+bob" || len(groups) != 1 || !slices.Equal(groups[0].Members, []string{"alice", "bob"}) {
		test.Fatalf("published to %q; groups %+v", message.Topic, groups)
	}
}

// Both halves of the conversation land in one place, which both read: neither sees half of it.
func TestAMessageToAtNameLandsInThePair(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob")
	mustPublish(test, hub, "bob", "@alice", "can I change the api?")
	mustPublish(test, hub, "alice", "@bob", "yes")
	mustPublish(test, hub, "bob", "alice+bob", "thanks")
	if got := texts(test, hub, "alice"); !slices.Equal(got, []string{"can I change the api?", "thanks"}) {
		test.Fatalf("alice read %v", got)
	}
	if got := texts(test, hub, "bob"); !slices.Equal(got, []string{"yes"}) {
		test.Fatalf("bob read %v", got)
	}
	for _, message := range hub.topics["@alice"] {
		test.Fatalf("left in alice's inbox: %q", message.Text)
	}
}

// A third session may read a pair, as the person's chat does, but not write in it.
func TestAThirdSessionCannotPublishInAPair(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob", "carol")
	mustPublish(test, hub, "alice", "@bob", "hello")
	if _, err := hub.Publish("carol", "alice+bob", "me too"); !errors.Is(err, NotAMember) {
		test.Fatalf("carol published in alice+bob: %v", err)
	}
	if _, err := hub.Publish("carol", "alice+dave", "hello"); !errors.Is(err, NotAMember) {
		test.Fatalf("carol made a pair she is not in: %v", err)
	}
}

// Writing to oneself is still writing to one's own inbox.
func TestAMessageToOneselfStaysInTheInbox(test *testing.T) {
	hub, _ := newHub(test, "alice")
	message, err := hub.Publish("alice", "@alice", "remember the docs")
	if err != nil || message.Topic != "@alice" || len(hub.Groups()) != 0 {
		test.Fatalf("published to %q, %v; groups %v", message.Topic, err, hub.Groups())
	}
}

// A pair lasts across hub restarts, like any group, so its two sessions keep finding it.
func TestAPairIsKept(test *testing.T) {
	path := filepath.Join(test.TempDir(), "kept.json")
	before := keptHub(test, path)
	if _, err := before.Publish("alice", "@bob", "hello"); err != nil {
		test.Fatal(err)
	}
	after := keptHub(test, path)
	if groups := after.Groups(); len(groups) != 1 || groups[0].Name != "alice+bob" {
		test.Fatalf("came back %+v", groups)
	}
}

// A pair is always its two sessions: anything else is a group, made on purpose.
func TestAPairCannotChangeItsMembers(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob")
	mustPublish(test, hub, "alice", "@bob", "hello")
	for name, err := range map[string]error{
		"add":    hub.AddMember("alice", "alice+bob", "carol"),
		"remove": hub.RemoveMember("alice", "alice+bob", "bob"),
		"delete": hub.RemoveGroup("alice", "alice+bob"),
	} {
		if !errors.Is(err, PairFixed) {
			test.Errorf("%s: %v, want %v", name, err, PairFixed)
		}
	}
}
