package hub

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// keptHub is a hub that keeps its declarations in a file, with alice joined.
func keptHub(test *testing.T, path string) *Hub {
	test.Helper()
	hub := New(time.Hour, time.Now)
	if err := hub.KeepIn(path); err != nil {
		test.Fatal(err)
	}
	if err := hub.Join("alice"); err != nil {
		test.Fatal(err)
	}
	return hub
}

func TestAGroupAndADeclaredTopicComeBackAfterARestart(test *testing.T) {
	path := filepath.Join(test.TempDir(), "kept.json")
	before := keptHub(test, path)
	must(test, before.CreateGroup("alice", "team", "the release", []string{"alice", "bob"}))
	must(test, before.Declare("alice", "tests", Metrics))
	must(test, before.Archive("alice", "tests"))

	after := keptHub(test, path)
	groups := after.Groups()
	if len(groups) != 1 || groups[0].Theme != "the release" || len(groups[0].Members) != 2 {
		test.Errorf("the groups came back as %+v", groups)
	}
	state := after.state()
	if len(state.Kinds) != 1 || state.Kinds[0].Topic != "tests" || state.Kinds[0].Kind != Metrics {
		test.Errorf("the declared topics came back as %+v", state.Kinds)
	}
	if archived := after.Archived(); len(archived) != 1 || archived[0] != "tests" {
		test.Errorf("the archived topics came back as %v", archived)
	}
}

func TestAnUndeclaredTopicIsNotKept(test *testing.T) {
	path := filepath.Join(test.TempDir(), "kept.json")
	before := keptHub(test, path)
	must(test, before.Declare("alice", "kept", Conversation))
	if _, err := before.Publish("alice", "passing", "hello"); err != nil {
		test.Fatal(err)
	}
	after := keptHub(test, path)
	if kinds := after.state().Kinds; len(kinds) != 1 || kinds[0].Topic != "kept" {
		test.Errorf("came back %+v, want only the declared topic", kinds)
	}
}

func TestARemovedTopicIsNotBroughtBack(test *testing.T) {
	path := filepath.Join(test.TempDir(), "kept.json")
	before := keptHub(test, path)
	must(test, before.Declare("alice", "tests", Metrics))
	must(test, before.CreateGroup("alice", "team", "", []string{"alice"}))
	must(test, before.RemoveTopic("alice", "tests"))
	must(test, before.RemoveGroup("alice", "team"))
	after := keptHub(test, path)
	if len(after.state().Kinds) != 0 || len(after.Groups()) != 0 {
		test.Errorf("came back %+v and %+v", after.state().Kinds, after.Groups())
	}
}

func TestADeclaredTopicIsListedWithNobodyListening(test *testing.T) {
	hub := keptHub(test, "")
	must(test, hub.Declare("alice", "plans", Conversation))
	must(test, hub.Leave("alice"))
	if kinds := hub.state().Kinds; len(kinds) != 1 || kinds[0].Topic != "plans" {
		test.Errorf("the state lists %+v", kinds)
	}
}

func TestAKeepFileThatIsNotJSONStopsTheHub(test *testing.T) {
	path := filepath.Join(test.TempDir(), "kept.json")
	must(test, os.WriteFile(path, []byte("not json"), 0o600))
	if err := New(time.Hour, time.Now).KeepIn(path); !errors.Is(err, KeepUnreadable) {
		test.Errorf("got %v, want hub.keep-unreadable", err)
	}
}

func must(test *testing.T, err error) {
	test.Helper()
	if err != nil {
		test.Fatal(err)
	}
}
