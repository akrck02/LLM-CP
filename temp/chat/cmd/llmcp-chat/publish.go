package main

import (
	"context"
	"encoding/json"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/akrck02/LLM-CP/hub"
	"github.com/akrck02/LLM-CP/protocol"
	"github.com/akrck02/LLM-CP/server"
)

// maximumMessageBytes keeps one message from the page to a reasonable size.
const maximumMessageBytes = 16 << 10

// written is a message the person writes in the page.
type written struct {
	As    string `json:"as"`
	Topic string `json:"topic"`
	Text  string `json:"text"`
}

// publisher sends what the person writes in the page to the hub, as a session of their name,
// and keeps that session listening to the topics the person is subscribed to.
type publisher struct {
	socket string
	kept   *subscriptions
}

// servePublish takes one message as JSON and publishes it, joining the hub first if needed.
func (sender publisher) servePublish(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(response, "use POST", http.StatusMethodNotAllowed)
		return
	}
	var message written
	if err := json.NewDecoder(http.MaxBytesReader(response, request.Body, maximumMessageBytes)).Decode(&message); err != nil {
		http.Error(response, "a message is {as, topic, text}", http.StatusBadRequest)
		return
	}
	if err := sender.publish(request.Context(), message); err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

// publish sends a message as its writer's session. The person at the chat may write in every
// group: when sessions made one without them, they are added to it first.
func (sender publisher) publish(request context.Context, message written) error {
	return sender.withClient(request, message.As, func(client protocol.HubClient) error {
		asked := &protocol.PublishRequest{Session: message.As, Topic: message.Topic, Text: message.Text}
		_, err := client.Publish(request, asked)
		if server.CodeName(err) != hub.NotAMember.Name() {
			return err
		}
		member := &protocol.MemberRequest{Session: message.As, Group: message.Topic, Member: message.As}
		if _, err := client.AddMember(request, member); err != nil {
			return err
		}
		_, err = client.Publish(request, asked)
		return err
	})
}

// connectTo opens a connection to the hub's socket.
func connectTo(socket string) (protocol.HubClient, *grpc.ClientConn, error) {
	connection, err := grpc.NewClient("unix://"+socket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return protocol.NewHubClient(connection), connection, nil
}
