package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/akrck02/LLM-CP/chat/conversation"
	"github.com/akrck02/LLM-CP/hub"
	"github.com/akrck02/LLM-CP/protocol"
	"github.com/akrck02/LLM-CP/server"
)

// action is what the page asks besides publishing.
type action = conversation.Action

// actionCall is one call to the hub made for the page.
type actionCall func(request context.Context, client protocol.HubClient, asked action) error

// createGroup makes the group the page asked for.
func createGroup(request context.Context, client protocol.HubClient, asked action) error {
	_, err := client.CreateGroup(request, &protocol.GroupRequest{Session: asked.As, Group: asked.Topic, Theme: asked.Theme, Members: asked.Members})
	return err
}

// archiveTopic keeps a chat out of the list.
func archiveTopic(request context.Context, client protocol.HubClient, asked action) error {
	_, err := client.Archive(request, &protocol.TopicRequest{Session: asked.As, Topic: asked.Topic})
	return err
}

// restoreTopic brings an archived chat back.
func restoreTopic(request context.Context, client protocol.HubClient, asked action) error {
	_, err := client.Restore(request, &protocol.TopicRequest{Session: asked.As, Topic: asked.Topic})
	return err
}

// addMembers puts sessions in a group.
func addMembers(request context.Context, client protocol.HubClient, asked action) error {
	for _, member := range asked.Members {
		if _, err := client.AddMember(request, &protocol.MemberRequest{Session: asked.As, Group: asked.Topic, Member: member}); err != nil {
			return err
		}
	}
	return nil
}

// removeMembers takes sessions out of a group.
func removeMembers(request context.Context, client protocol.HubClient, asked action) error {
	for _, member := range asked.Members {
		if _, err := client.RemoveMember(request, &protocol.MemberRequest{Session: asked.As, Group: asked.Topic, Member: member}); err != nil {
			return err
		}
	}
	return nil
}

// serveAction answers a POST of an action as JSON with one call to the hub.
func (sender publisher) serveAction(call actionCall) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(response, "use POST", http.StatusMethodNotAllowed)
			return
		}
		var asked action
		if err := json.NewDecoder(http.MaxBytesReader(response, request.Body, maximumMessageBytes)).Decode(&asked); err != nil {
			http.Error(response, "an action is {as, topic, theme, members}", http.StatusBadRequest)
			return
		}
		if err := sender.withClient(request.Context(), asked.As, func(client protocol.HubClient) error {
			return call(request.Context(), client, asked)
		}); err != nil {
			http.Error(response, err.Error(), http.StatusBadRequest)
			return
		}
		response.WriteHeader(http.StatusNoContent)
	}
}

// withClient makes a call to the hub as a session; one not joined yet, or dropped after a
// silence, joins and tries once more.
func (sender publisher) withClient(request context.Context, as string, call func(protocol.HubClient) error) error {
	client, connection, err := connectTo(sender.socket)
	if err != nil {
		return err
	}
	defer connection.Close()
	err = call(client)
	if server.CodeName(err) != hub.SessionNotFound.Name() {
		return err
	}
	if _, err := client.Join(request, &protocol.SessionRequest{Session: as}); err != nil {
		return err
	}
	return call(client)
}
