package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/akrck02/LLM-CP/internal/failure"
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
