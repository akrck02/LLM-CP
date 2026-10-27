package main

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/akrck02/LLM-CP/hub"
	"github.com/akrck02/LLM-CP/protocol"
	"github.com/akrck02/LLM-CP/server"
)

// The person at the chat is who the sessions work for: a group the sessions made among
// themselves must not stop them from writing in it.
func TestThePersonMayWriteInAGroupMadeWithoutThem(test *testing.T) {
	sender, request := startHub(test)
	err := sender.withClient(request, "organizer", func(client protocol.HubClient) error {
		_, err := client.CreateGroup(request, &protocol.GroupRequest{Session: "organizer", Group: "all", Members: []string{"organizer"}})
		return err
	})
	if err != nil {
		test.Fatal(err)
	}
	if err := sender.publish(request, written{As: "alice", Topic: "all", Text: "hello"}); err != nil {
		test.Errorf("alice could not write in the group: %v", err)
	}
}

// startHub serves a hub on a socket in a temporary folder, for the chat server to call.
func startHub(test *testing.T) (publisher, context.Context) {
	test.Helper()
	socket := filepath.Join(test.TempDir(), "llmcp.sock")
	listener, err := server.Listen(socket)
	if err != nil {
		test.Fatal(err)
	}
	hubServer := server.New(hub.New(time.Hour, time.Now))
	go hubServer.Serve(listener) // its error only says the test stopped the server
	test.Cleanup(hubServer.Stop)
	return publisher{socket: socket}, context.Background()
}

func TestMembersCanBeAddedToAGroupAndTakenOut(test *testing.T) {
	sender, request := startHub(test)
	err := sender.withClient(request, "alice", func(client protocol.HubClient) error {
		if _, err := client.CreateGroup(request, &protocol.GroupRequest{Session: "alice", Group: "team", Members: []string{"alice", "bob"}}); err != nil {
			return err
		}
		if err := addMembers(request, client, action{As: "alice", Topic: "team", Members: []string{"carol"}}); err != nil {
			return err
		}
		if err := removeMembers(request, client, action{As: "alice", Topic: "team", Members: []string{"bob"}}); err != nil {
			return err
		}
		groups, err := client.Groups(request, &protocol.SessionRequest{Session: "alice"})
		if err != nil {
			return err
		}
		if members := groups.GetGroups()[0].GetMembers(); !slices.Equal(members, []string{"alice", "carol"}) {
			test.Errorf("the members are %v", members)
		}
		return nil
	})
	if err != nil {
		test.Fatal(err)
	}
}
