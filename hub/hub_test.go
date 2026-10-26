package hub

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

// clock is a time that tests move by hand.
type clock struct{ current time.Time }

func (moving *clock) now() time.Time { return moving.current }

const silenceLimit = 30 * time.Minute

// newHub makes a hub with the given sessions joined, and the clock that drives it.
func newHub(test *testing.T, names ...string) (*Hub, *clock) {
	test.Helper()
	moving := &clock{current: time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)}
	hub := New(silenceLimit, moving.now)
	for _, name := range names {
		if err := hub.Join(name); err != nil {
			test.Fatalf("join %s: %v", name, err)
		}
	}
	return hub, moving
}

// mustPublish publishes or stops the test.
func mustPublish(test *testing.T, hub *Hub, from, topic, text string) {
	test.Helper()
	if _, err := hub.Publish(from, topic, text); err != nil {
		test.Fatalf("publish: %v", err)
	}
}

// mustListen listens or stops the test.
func mustListen(test *testing.T, hub *Hub, name, topic string, mode Mode) {
	test.Helper()
	if err := hub.Listen(name, topic, mode); err != nil {
		test.Fatalf("listen: %v", err)
	}
}

// texts reads what a session has waiting, as texts only.
func texts(test *testing.T, hub *Hub, name string) []string {
	test.Helper()
	messages, err := hub.Read(context.Background(), name, 0)
	if err != nil {
		test.Fatalf("read %s: %v", name, err)
	}
	read := []string{}
	for _, message := range messages {
		read = append(read, message.Text)
	}
	return read
}

func TestASecondSessionCannotTakeANameAlreadyInUse(test *testing.T) {
	hub, _ := newHub(test, "alice")
	if err := hub.Join("alice"); !errors.Is(err, NameTaken) {
		test.Fatalf("got %v, want %s", err, NameTaken.Name())
	}
	if err := hub.Leave("alice"); err != nil {
		test.Fatal(err)
	}
	if err := hub.Join("alice"); err != nil {
		test.Fatalf("the name was not freed on leaving: %v", err)
	}
}

func TestASilentSessionIsDroppedAndFreesItsName(test *testing.T) {
	hub, moving := newHub(test, "alice", "bob")
	moving.current = moving.current.Add(silenceLimit / 2)
	if _, err := hub.Sessions("bob"); err != nil {
		test.Fatal(err)
	}
	moving.current = moving.current.Add(silenceLimit/2 + time.Second)
	if err := hub.Join("alice"); err != nil {
		test.Fatalf("alice was silent too long and should be gone: %v", err)
	}
	if _, err := hub.Sessions("bob"); err != nil {
		test.Fatalf("bob spoke in time and should stay: %v", err)
	}
}

func TestASessionSeesTheOthersAndTheirTopics(test *testing.T) {
	hub, _ := newHub(test, "bob", "alice")
	mustListen(test, hub, "alice", "api", OnlyNew)
	sessions, err := hub.Sessions("bob")
	if err != nil {
		test.Fatal(err)
	}
	want := []Session{
		{Name: "alice", Topics: []string{"@alice", "api"}},
		{Name: "bob", Topics: []string{"@bob"}},
	}
	if !slices.EqualFunc(sessions, want, func(got, wanted Session) bool {
		return got.Name == wanted.Name && slices.Equal(got.Topics, wanted.Topics)
	}) {
		test.Fatalf("got %v, want %v", sessions, want)
	}
}

func TestASessionListensToSeveralTopicsAtOnce(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob")
	mustListen(test, hub, "alice", "api", OnlyNew)
	mustListen(test, hub, "alice", "docs", OnlyNew)
	mustPublish(test, hub, "bob", "api", "one")
	mustPublish(test, hub, "bob", "docs", "two")
	if got := texts(test, hub, "alice"); !slices.Equal(got, []string{"one", "two"}) {
		test.Fatalf("got %v", got)
	}
	if err := hub.Unlisten("alice", "docs"); err != nil {
		test.Fatal(err)
	}
	mustPublish(test, hub, "bob", "docs", "three")
	if got := texts(test, hub, "alice"); len(got) != 0 {
		test.Fatalf("alice stopped listening to docs but got %v", got)
	}
}

func TestAMessageReachesEveryoneListeningToItsTopicAndNobodyElse(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob", "carol")
	mustListen(test, hub, "alice", "api", OnlyNew)
	mustListen(test, hub, "bob", "api", OnlyNew)
	mustPublish(test, hub, "alice", "api", "the API changed")
	cases := map[string]int{"alice": 0, "bob": 1, "carol": 0}
	for name, wanted := range cases {
		if got := texts(test, hub, name); len(got) != wanted {
			test.Errorf("%s got %v, want %d messages", name, got, wanted)
		}
	}
}

func TestAMessageToASessionsOwnTopicReachesOnlyThatSession(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob", "carol")
	if err := hub.Listen("carol", "@bob", AlsoPast); !errors.Is(err, TopicReserved) {
		test.Fatalf("carol listened to bob's topic: %v", err)
	}
	mustPublish(test, hub, "alice", OwnTopic("bob"), "only for bob")
	if got := texts(test, hub, "bob"); !slices.Equal(got, []string{"only for bob"}) {
		test.Fatalf("bob got %v", got)
	}
	if got := texts(test, hub, "carol"); len(got) != 0 {
		test.Fatalf("carol got %v", got)
	}
}

func TestASessionThatListensToThePastReadsWhatWasPublishedBeforeItCame(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob", "carol")
	mustPublish(test, hub, "alice", "plan", "earlier")
	mustListen(test, hub, "bob", "plan", AlsoPast)
	mustListen(test, hub, "carol", "plan", OnlyNew)
	mustPublish(test, hub, "alice", "plan", "later")
	if got := texts(test, hub, "bob"); !slices.Equal(got, []string{"earlier", "later"}) {
		test.Fatalf("bob got %v", got)
	}
	if got := texts(test, hub, "carol"); !slices.Equal(got, []string{"later"}) {
		test.Fatalf("carol got %v", got)
	}
}

func TestMessagesWaitInOrderUntilTheyAreRead(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob")
	mustListen(test, hub, "bob", "api", OnlyNew)
	for _, text := range []string{"one", "two", "three"} {
		mustPublish(test, hub, "alice", "api", text)
	}
	if got := texts(test, hub, "bob"); !slices.Equal(got, []string{"one", "two", "three"}) {
		test.Fatalf("got %v", got)
	}
	if got := texts(test, hub, "bob"); len(got) != 0 {
		test.Fatalf("read messages came back: %v", got)
	}
}

func TestAReadThatWaitsWakesWhenAMessageArrives(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob")
	mustListen(test, hub, "bob", "api", OnlyNew)
	go func() {
		time.Sleep(10 * time.Millisecond)
		if _, err := hub.Publish("alice", "api", "done"); err != nil {
			test.Error(err)
		}
	}()
	messages, err := hub.Read(context.Background(), "bob", 5*time.Second)
	if err != nil || len(messages) != 1 || messages[0].Text != "done" {
		test.Fatalf("got %v, %v", messages, err)
	}
}

func TestBadNamesAndEmptyMessagesAreRefused(test *testing.T) {
	hub, _ := newHub(test, "alice")
	if err := hub.Join("has space"); !errors.Is(err, InvalidName) {
		test.Errorf("join: %v", err)
	}
	if _, err := hub.Publish("alice", "api", "  "); !errors.Is(err, EmptyMessage) {
		test.Errorf("publish: %v", err)
	}
	if _, err := hub.Publish("nobody", "api", "hello"); !errors.Is(err, SessionNotFound) {
		test.Errorf("publish from nobody: %v", err)
	}
}

func TestASessionThatTakesAFreedNameDoesNotReadTheInboxOfTheOneBefore(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob")
	mustPublish(test, hub, "alice", OwnTopic("bob"), "for the first bob")
	if err := hub.Leave("bob"); err != nil {
		test.Fatal(err)
	}
	if err := hub.Join("bob"); err != nil {
		test.Fatal(err)
	}
	if got := texts(test, hub, "bob"); len(got) != 0 {
		test.Fatalf("the new bob read %v", got)
	}
}

func TestASessionCannotStopListeningToItsOwnTopic(test *testing.T) {
	hub, _ := newHub(test, "alice")
	if err := hub.Unlisten("alice", OwnTopic("alice")); !errors.Is(err, NotListening) {
		test.Fatalf("got %v", err)
	}
}

func TestAReadThatWaitsKeepsWaitingAfterAStaleWakeUp(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob")
	mustListen(test, hub, "bob", "docs", OnlyNew)
	mustListen(test, hub, "bob", "api", OnlyNew)
	mustPublish(test, hub, "alice", "docs", "wakes bob")
	if err := hub.Unlisten("bob", "docs"); err != nil {
		test.Fatal(err)
	}
	// "docs" left a wake-up behind with nothing to read; the read must not end there.
	go func() {
		time.Sleep(20 * time.Millisecond)
		if _, err := hub.Publish("alice", "api", "worth reading"); err != nil {
			test.Error(err)
		}
	}()
	messages, err := hub.Read(context.Background(), "bob", 5*time.Second)
	if err != nil || len(messages) != 1 || messages[0].Text != "worth reading" {
		test.Fatalf("got %v, %v", messages, err)
	}
}
