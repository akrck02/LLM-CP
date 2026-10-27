package hub

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func TestATopicNobodyDeclaredIsAConversation(test *testing.T) {
	hub, _ := newHub(test, "alice")
	mustPublish(test, hub, "alice", "api", "hello")
	if err := hub.Declare("alice", "api", Conversation); err != nil {
		test.Errorf("declaring what it already is: %v", err)
	}
	if err := hub.Declare("alice", "api", Metrics); !errors.Is(err, KindTaken) {
		test.Errorf("a conversation with messages became metrics: %v", err)
	}
}

func TestATopicKeepsTheKindItWasDeclaredWith(test *testing.T) {
	hub, _ := newHub(test, "alice")
	if err := hub.Declare("alice", "load", Metrics); err != nil {
		test.Fatal(err)
	}
	if err := hub.Declare("alice", "load", Conversation); !errors.Is(err, KindTaken) {
		test.Errorf("the kind changed: %v", err)
	}
	if err := hub.Declare("alice", "load", "files"); !errors.Is(err, UnknownKind) {
		test.Errorf("an unknown kind was taken: %v", err)
	}
}

func TestARemovedTopicCanBeDeclaredAgain(test *testing.T) {
	hub, _ := newHub(test, "alice")
	mustPublish(test, hub, "alice", "load", "a sentence")
	if err := hub.RemoveTopic("alice", "load"); err != nil {
		test.Fatal(err)
	}
	if err := hub.Declare("alice", "load", Metrics); err != nil {
		test.Errorf("a removed topic could not be made again: %v", err)
	}
}

func TestAMetricsTopicRefusesASentence(test *testing.T) {
	hub, moving := newHub(test, "alice")
	if err := hub.Declare("alice", "load", Metrics); err != nil {
		test.Fatal(err)
	}
	for _, text := range []string{"the load is high", `{"name":"","value":1}`, `{"name":"load","value":"high"}`} {
		if _, err := hub.Publish("alice", "load", text); !errors.Is(err, NotAPoint) {
			test.Errorf("%s was taken: %v", text, err)
		}
	}
	message, err := hub.Publish("alice", "load", `{"name":"load.average","value":0.5,"labels":{"machine":"box"}}`)
	if err != nil {
		test.Fatal(err)
	}
	point, err := ParsePoint(message.Text)
	if err != nil || point.At != moving.current.UnixMilli() || point.Labels["machine"] != "box" {
		test.Errorf("the point kept is %s (%v)", message.Text, err)
	}
}

func TestAMetricsTopicForgetsPointsOlderThanAWeek(test *testing.T) {
	moving := &clock{current: time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)}
	hub := New(2*PointsKept, moving.now) // nobody goes silent for long enough to be dropped
	for _, name := range []string{"alice", "bob"} {
		if err := hub.Join(name); err != nil {
			test.Fatal(err)
		}
	}
	if err := hub.Declare("alice", "load", Metrics); err != nil {
		test.Fatal(err)
	}
	mustListen(test, hub, "bob", "load", OnlyNew)
	mustPublish(test, hub, "alice", "load", `{"name":"load","value":1}`)
	moving.current = moving.current.Add(PointsKept + time.Hour)
	mustPublish(test, hub, "alice", "load", `{"name":"load","value":2}`)
	hub.mutex.Lock()
	kept, bobAt := len(hub.topics["load"]), hub.sessions["bob"].cursors["load"]
	hub.mutex.Unlock()
	if kept != 1 || bobAt != 0 {
		test.Errorf("kept %d points; bob reads from %d", kept, bobAt)
	}
}

func TestTheStateSaysEachTopicsKindAndPublishers(test *testing.T) {
	hub, _ := newHub(test, "alice")
	if err := hub.Declare("alice", "load", Metrics); err != nil {
		test.Fatal(err)
	}
	mustPublish(test, hub, "alice", "load", `{"name":"load","value":1}`)
	hub.mutex.Lock()
	kinds := hub.state().Kinds
	hub.mutex.Unlock()
	if len(kinds) != 1 || kinds[0].Kind != Metrics || !slices.Equal(kinds[0].Publishers, []string{"alice"}) {
		test.Errorf("the state says %+v", kinds)
	}
}

func TestAPointWithATitleIsShownByItsTitleAndLabels(test *testing.T) {
	point := Point{Name: "tests.passed", Title: "Tests passed", Labels: map[string]string{"repo": "api", "branch": "main"}}
	if shown := point.Shown(); shown != "Tests passed · main · api" {
		test.Errorf("shown as %q", shown)
	}
	point.Title = ""
	if shown := point.Shown(); shown != "tests.passed branch=main repo=api" {
		test.Errorf("with no title, shown as %q", shown)
	}
}
