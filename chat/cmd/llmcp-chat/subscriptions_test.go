package main

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/akrck02/LLM-CP/internal/failure"
	"github.com/akrck02/LLM-CP/protocol"
)

// The hub forgets a person after a silence or a restart; what they chose to follow must not
// go with it, nor with a restart of the chat server.
func TestASubscriptionIsKeptAcrossARestartOfTheServer(test *testing.T) {
	path := filepath.Join(test.TempDir(), "subscriptions.json")
	kept, err := loadSubscriptions(path)
	if err != nil {
		test.Fatal(err)
	}
	for _, topic := range []string{"release", "api", "docs"} {
		if err := kept.set("alice", topic, true); err != nil {
			test.Fatal(err)
		}
	}
	if err := kept.set("alice", "docs", false); err != nil {
		test.Fatal(err)
	}
	again, err := loadSubscriptions(path)
	if err != nil {
		test.Fatal(err)
	}
	if topics := again.of("alice"); !slices.Equal(topics, []string{"api", "release"}) {
		test.Errorf("after a restart alice is subscribed to %v", topics)
	}
}

// Someone new starts subscribed to nothing, by the author's choice: they add what they follow.
func TestANewNameHasNoSubscriptions(test *testing.T) {
	kept, err := loadSubscriptions(filepath.Join(test.TempDir(), "subscriptions.json"))
	if err != nil {
		test.Fatal(err)
	}
	if topics := kept.of("bob"); len(topics) != 0 {
		test.Errorf("bob starts subscribed to %v", topics)
	}
}

// A broken file stops the server with a code to search for, rather than starting it empty and
// overwriting what people chose.
func TestABrokenFileOfSubscriptionsSaysItCannotBeRead(test *testing.T) {
	path := filepath.Join(test.TempDir(), "subscriptions.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		test.Fatal(err)
	}
	if _, err := loadSubscriptions(path); !failure.Is(err, SubscriptionsUnreadable) {
		test.Errorf("the error is %v", err)
	}
}

// A subscription is a real LLM-CP listen, so the header and the other sessions see it.
func TestSubscribingMakesThePersonListenOnTheHub(test *testing.T) {
	sender, request := startHub(test)
	if err := sender.subscribe(request, subscription{As: "alice", Topic: "api", Subscribed: true}); err != nil {
		test.Fatal(err)
	}
	if topics := listenedBy(test, sender, "alice"); !slices.Contains(topics, "api") {
		test.Errorf("alice listens to %v", topics)
	}
	if err := sender.subscribe(request, subscription{As: "alice", Topic: "api"}); err != nil {
		test.Fatal(err)
	}
	if topics := listenedBy(test, sender, "alice"); slices.Contains(topics, "api") {
		test.Errorf("after unsubscribing alice still listens to %v", topics)
	}
}

// The hub drops a person after half an hour of silence; the next time they write, their
// subscriptions must be there again without them choosing each one anew.
func TestAfterASilenceThePersonListensToTheirTopicsAgain(test *testing.T) {
	sender, request := startHub(test)
	if err := sender.subscribe(request, subscription{As: "alice", Topic: "api", Subscribed: true}); err != nil {
		test.Fatal(err)
	}
	err := sender.withClient(request, "alice", func(client protocol.HubClient) error {
		_, err := client.Leave(request, &protocol.SessionRequest{Session: "alice"})
		return err
	})
	if err != nil {
		test.Fatal(err)
	}
	if err := sender.publish(request, written{As: "alice", Topic: "docs", Text: "back"}); err != nil {
		test.Fatal(err)
	}
	if topics := listenedBy(test, sender, "alice"); !slices.Contains(topics, "api") {
		test.Errorf("after rejoining alice listens to %v", topics)
	}
}

// listenedBy is what a session listens to, as the hub says.
func listenedBy(test *testing.T, sender publisher, name string) []string {
	test.Helper()
	var topics []string
	err := sender.withClient(context.Background(), "observer", func(client protocol.HubClient) error {
		list, err := client.Sessions(context.Background(), &protocol.SessionRequest{Session: "observer"})
		for _, session := range list.GetSessions() {
			if session.GetName() == name {
				topics = session.GetTopics()
			}
		}
		return err
	})
	if err != nil {
		test.Fatal(err)
	}
	return topics
}
