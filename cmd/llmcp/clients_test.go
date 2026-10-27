package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestTypingFindsTheSessionByItsClient(test *testing.T) {
	socket := filepath.Join(test.TempDir(), "llmcp.sock")
	if err := rememberClient(socket, "client-1", "alice"); err != nil {
		test.Fatalf("remember: %v", err)
	}
	given, err := parseOptions([]string{"typing", "--client", "client-1", "--socket", socket})
	if err != nil {
		test.Fatalf("parse: %v", err)
	}
	if given.session != "alice" {
		test.Fatalf("the session is %q, want alice", given.session)
	}
}

func TestAnUnknownClientSaysSo(test *testing.T) {
	socket := filepath.Join(test.TempDir(), "llmcp.sock")
	if _, err := clientSession(socket, "nobody"); !errors.Is(err, UnknownClient) {
		test.Fatalf("got %v, want %s", err, UnknownClient.Name())
	}
}

func TestLeavingForgetsTheClientsSession(test *testing.T) {
	socket := filepath.Join(test.TempDir(), "llmcp.sock")
	if err := rememberClient(socket, "client-1", "alice"); err != nil {
		test.Fatalf("remember: %v", err)
	}
	forgetClient(socket, "client-1", "alice")
	if _, err := clientSession(socket, "client-1"); !errors.Is(err, UnknownClient) {
		test.Fatalf("after leaving got %v, want %s", err, UnknownClient.Name())
	}
}

func TestEachCommandLineErrorHasItsOwnCode(test *testing.T) {
	socket := filepath.Join(test.TempDir(), "llmcp.sock")
	if err := os.WriteFile(clientsFolder(socket), []byte("a file where a folder goes"), 0o600); err != nil {
		test.Fatal(err)
	}
	if err := rememberClient(socket, "client-1", "alice"); !errors.Is(err, ClientNotRemembered) {
		test.Fatalf("writing into a file got %v, want %s", err, ClientNotRemembered.Name())
	}
	if _, err := clientSession(socket, "client-2"); !errors.Is(err, UnknownClient) {
		test.Fatalf("an unknown client got %v, want %s", err, UnknownClient.Name())
	}
}

func TestMembersAreReadAsPeopleWriteThem(test *testing.T) {
	if got := splitNames("alice, bob,"); !slices.Equal(got, []string{"alice", "bob"}) {
		test.Fatalf("got %q, want [alice bob]", got)
	}
}
