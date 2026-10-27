package hub

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func TestAnArchivedTopicStaysArchivedWhenAMessageArrives(test *testing.T) {
	hub, _ := newHub(test, "alice")
	mustPublish(test, hub, "alice", "api", "first")
	mustArchive(test, hub.Archive("alice", "api"))
	mustPublish(test, hub, "alice", "api", "second")
	if got := hub.Archived(); !slices.Equal(got, []string{"api"}) {
		test.Fatalf("archived topics are %v, want [api]", got)
	}
	mustArchive(test, hub.Restore("alice", "api"))
	if got := hub.Archived(); len(got) != 0 {
		test.Fatalf("a restored topic is still archived: %v", got)
	}
}

func TestTypingEndsWhenTheSessionPublishes(test *testing.T) {
	hub, _ := newHub(test, "alice")
	if err := hub.Typing("alice", true); err != nil {
		test.Fatalf("typing: %v", err)
	}
	if !typing(test, hub, "alice") {
		test.Fatal("alice is not typing after saying so")
	}
	mustPublish(test, hub, "alice", "api", "done")
	if typing(test, hub, "alice") {
		test.Fatal("alice is still typing after publishing")
	}
}

func TestTypingEndsOnItsOwnAfterAWhile(test *testing.T) {
	hub, moving := newHub(test, "alice")
	if err := hub.Typing("alice", true); err != nil {
		test.Fatalf("typing: %v", err)
	}
	moving.current = moving.current.Add(TypingLasts + time.Second)
	if typing(test, hub, "alice") {
		test.Fatal("alice is still typing long after her last signal")
	}
}

// typing tells whether a session shows as typing in the hub's contacts.
func typing(test *testing.T, hub *Hub, name string) bool {
	test.Helper()
	for _, contact := range hub.Contacts() {
		if contact.Name == name {
			return contact.Typing
		}
	}
	test.Fatalf("%s is not among the contacts", name)
	return false
}

func TestAWatcherHearsContactsGroupsAndTyping(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob")
	watcher, _ := hub.Watch(false)
	defer hub.Unwatch(watcher)
	if first := <-watcher.Events; first.State == nil || len(first.State.Contacts) != 2 {
		test.Fatalf("the first event is %+v, want the state with two contacts", first)
	}
	if err := hub.CreateGroup("alice", "api", "the tasks API", []string{"alice", "bob"}); err != nil {
		test.Fatalf("create: %v", err)
	}
	if event := <-watcher.Events; event.State == nil || len(event.State.Groups) != 1 {
		test.Fatalf("after a group is made the watcher heard %+v", event)
	}
	if err := hub.Typing("bob", true); err != nil {
		test.Fatalf("typing: %v", err)
	}
	event := <-watcher.Events
	if event.State == nil || !event.State.Contacts[1].Typing || event.State.Contacts[1].TypingUntil.IsZero() {
		test.Fatalf("after bob types the watcher heard %+v", event)
	}
	mustArchive(test, hub.Archive("alice", "api"))
	if event := <-watcher.Events; event.State == nil || !slices.Equal(event.State.Archived, []string{"api"}) {
		test.Fatalf("after archiving the watcher heard %+v", event)
	}
}

// mustArchive stops the test if archiving or restoring failed.
func mustArchive(test *testing.T, err error) {
	test.Helper()
	if err != nil {
		test.Fatalf("archive: %v", err)
	}
}

func TestOnlyASessionOnTheHubArchivesAPlainName(test *testing.T) {
	hub, _ := newHub(test, "alice")
	if err := hub.Archive("nobody", "api"); !errors.Is(err, SessionNotFound) {
		test.Fatalf("a stranger archiving got %v, want %s", err, SessionNotFound.Name())
	}
	if err := hub.Archive("alice", "not a name"); !errors.Is(err, InvalidName) {
		test.Fatalf("archiving a bad name got %v, want %s", err, InvalidName.Name())
	}
}

func TestAWatcherTooFarBehindIsLetGo(test *testing.T) {
	hub, _ := newHub(test, "alice")
	watcher, _ := hub.Watch(false)
	for range watcherRoom + 1 {
		mustPublish(test, hub, "alice", "api", "flood")
	}
	for range watcher.Events { // drains what it holds, then ends: the channel was closed
	}
}

func TestADroppedSessionLeavesTheContacts(test *testing.T) {
	hub, moving := newHub(test, "alice", "bob")
	watcher, _ := hub.Watch(false)
	defer hub.Unwatch(watcher)
	<-watcher.Events // the state it starts with
	moving.current = moving.current.Add(silenceLimit + time.Second)
	if err := hub.Typing("alice", false); !errors.Is(err, SessionNotFound) {
		test.Fatalf("a silent session typing got %v, want %s", err, SessionNotFound.Name())
	}
	if event := <-watcher.Events; event.State == nil || len(event.State.Contacts) != 0 {
		test.Fatalf("after both were dropped the watcher heard %+v", event)
	}
}
