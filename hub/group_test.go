package hub

import (
	"errors"
	"slices"
	"testing"
)

func TestAGroupsMembersListenToItEvenThoseWhoJoinLater(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob")
	if err := hub.CreateGroup("alice", "api", "the tasks API", []string{"alice", "bob", "carol"}); err != nil {
		test.Fatalf("create: %v", err)
	}
	if err := hub.Join("carol"); err != nil {
		test.Fatalf("join: %v", err)
	}
	mustPublish(test, hub, "alice", "api", "hello")
	for _, name := range []string{"bob", "carol"} {
		if got := texts(test, hub, name); !slices.Equal(got, []string{"hello"}) {
			test.Fatalf("%s read %v, want the group's message", name, got)
		}
	}
}

func TestAGroupListsItsThemeAndMembers(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob", "carol")
	if err := hub.CreateGroup("alice", "api", "the tasks API", []string{"bob", "alice"}); err != nil {
		test.Fatalf("create: %v", err)
	}
	if err := hub.AddMember("alice", "api", "carol"); err != nil {
		test.Fatalf("add: %v", err)
	}
	if err := hub.RemoveMember("alice", "api", "bob"); err != nil {
		test.Fatalf("remove: %v", err)
	}
	want := []Group{{Name: "api", Theme: "the tasks API", Members: []string{"alice", "carol"}}}
	got := hub.Groups()
	if len(got) != 1 || got[0].Name != want[0].Name || got[0].Theme != want[0].Theme || !slices.Equal(got[0].Members, want[0].Members) {
		test.Fatalf("groups are %+v, want %+v", got, want)
	}
	if err := hub.RemoveGroup("alice", "api"); err != nil {
		test.Fatalf("remove group: %v", err)
	}
	if len(hub.Groups()) != 0 {
		test.Fatal("a removed group is still listed")
	}
}

func TestOnlyMembersPublishToAGroup(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob")
	if err := hub.CreateGroup("alice", "api", "", []string{"alice"}); err != nil {
		test.Fatalf("create: %v", err)
	}
	_, err := hub.Publish("bob", "api", "let me in")
	if !errors.Is(err, NotAMember) {
		test.Fatalf("a stranger publishing got %v, want %s", err, NotAMember.Name())
	}
}

func TestAGroupCannotTakeATopicAlreadyAGroup(test *testing.T) {
	hub, _ := newHub(test, "alice")
	if err := hub.CreateGroup("alice", "api", "", []string{"alice"}); err != nil {
		test.Fatalf("create: %v", err)
	}
	if err := hub.CreateGroup("alice", "api", "", []string{"alice"}); !errors.Is(err, GroupExists) {
		test.Fatalf("a second group of the same name got %v, want %s", err, GroupExists.Name())
	}
}

func TestARemovedMemberStopsListeningAndPublishing(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob")
	if err := hub.CreateGroup("alice", "api", "", []string{"alice", "bob"}); err != nil {
		test.Fatalf("create: %v", err)
	}
	if err := hub.RemoveMember("alice", "api", "bob"); err != nil {
		test.Fatalf("remove: %v", err)
	}
	mustPublish(test, hub, "alice", "api", "after bob left")
	if got := texts(test, hub, "bob"); len(got) != 0 {
		test.Fatalf("bob still reads the group: %v", got)
	}
	if _, err := hub.Publish("bob", "api", "back"); !errors.Is(err, NotAMember) {
		test.Fatalf("bob publishing got %v, want %s", err, NotAMember.Name())
	}
}

func TestADeletedGroupIsAPlainTopicAgain(test *testing.T) {
	hub, _ := newHub(test, "alice", "bob")
	if err := hub.CreateGroup("alice", "api", "", []string{"alice"}); err != nil {
		test.Fatalf("create: %v", err)
	}
	if err := hub.RemoveGroup("alice", "api"); err != nil {
		test.Fatalf("remove group: %v", err)
	}
	mustPublish(test, hub, "bob", "api", "anyone may write here now")
	if got := texts(test, hub, "alice"); !slices.Equal(got, []string{"anyone may write here now"}) {
		test.Fatalf("alice, still listening, read %v", got)
	}
}
