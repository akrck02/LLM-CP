// Package server serves a hub over gRPC on a Unix socket that only its owner can open.
package server

import (
	"context"
	"errors"
	"net"
	"os"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/akrck02/LLM-CP/hub"
	"github.com/akrck02/LLM-CP/internal/failure"
	"github.com/akrck02/LLM-CP/protocol"
)

// ownerOnly lets only the user who started the hub read and write the socket.
const ownerOnly os.FileMode = 0o600

// Listen opens the socket at a path, owner only. A socket left behind by a hub that ended is
// replaced; one with a hub still behind it is not.
func Listen(path string) (net.Listener, error) {
	if connection, err := net.Dial("unix", path); err == nil {
		connection.Close() // only checking that a hub answers
		return nil, failure.Wrap(SocketInUse, path, nil)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, failure.Wrap(SocketNotOpened, path, err)
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, failure.Wrap(SocketNotOpened, path, err)
	}
	if err := os.Chmod(path, ownerOnly); err != nil {
		listener.Close() // the chmod error is the one worth reporting
		return nil, failure.Wrap(SocketNotOpened, path, err)
	}
	return listener, nil
}

// New makes a gRPC server that answers for a hub.
func New(meetingPoint *hub.Hub) *grpc.Server {
	server := grpc.NewServer()
	protocol.RegisterHubServer(server, &service{hub: meetingPoint})
	return server
}

// service turns gRPC calls into calls to the hub.
type service struct {
	protocol.UnimplementedHubServer
	hub *hub.Hub
}

func (served *service) Join(_ context.Context, request *protocol.SessionRequest) (*protocol.Done, error) {
	return &protocol.Done{}, asStatus(served.hub.Join(request.Session))
}

func (served *service) Leave(_ context.Context, request *protocol.SessionRequest) (*protocol.Done, error) {
	return &protocol.Done{}, asStatus(served.hub.Leave(request.Session))
}

func (served *service) Sessions(_ context.Context, request *protocol.SessionRequest) (*protocol.SessionList, error) {
	sessions, err := served.hub.Sessions(request.Session)
	if err != nil {
		return nil, asStatus(err)
	}
	list := &protocol.SessionList{}
	for _, session := range sessions {
		list.Sessions = append(list.Sessions, &protocol.Session{Name: session.Name, Topics: session.Topics})
	}
	return list, nil
}

func (served *service) Listen(_ context.Context, request *protocol.ListenRequest) (*protocol.Done, error) {
	mode := hub.OnlyNew
	if request.Past {
		mode = hub.AlsoPast
	}
	return &protocol.Done{}, asStatus(served.hub.Listen(request.Session, request.Topic, mode))
}

func (served *service) Unlisten(_ context.Context, request *protocol.TopicRequest) (*protocol.Done, error) {
	return &protocol.Done{}, asStatus(served.hub.Unlisten(request.Session, request.Topic))
}

func (served *service) Publish(_ context.Context, request *protocol.PublishRequest) (*protocol.Message, error) {
	message, err := served.hub.Publish(request.Session, request.Topic, request.Text)
	if err != nil {
		return nil, asStatus(err)
	}
	return toProtocol(message), nil
}

func (served *service) Read(request context.Context, asked *protocol.ReadRequest) (*protocol.MessageList, error) {
	wait := time.Duration(asked.WaitSeconds) * time.Second
	messages, err := served.hub.Read(request, asked.Session, wait)
	if err != nil {
		return nil, asStatus(err)
	}
	list := &protocol.MessageList{}
	for _, message := range messages {
		list.Messages = append(list.Messages, toProtocol(message))
	}
	return list, nil
}

// toProtocol is a hub message as it travels.
func toProtocol(message hub.Message) *protocol.Message {
	return &protocol.Message{
		Number:    message.Number,
		Topic:     message.Topic,
		From:      message.From,
		Text:      message.Text,
		Published: message.Published.Unix(),
	}
}

// codeDomain names where the codes carried in a status come from.
const codeDomain = "llmcp"

// asStatus is a hub error as a gRPC status that carries the error's code name in its details,
// so the client finds the code without reading the message.
func asStatus(err error) error {
	if err == nil {
		return nil
	}
	answer := status.New(codes.FailedPrecondition, err.Error())
	code := failure.CodeOf(err)
	if code == nil {
		return answer.Err()
	}
	detailed, detailsErr := answer.WithDetails(&errdetails.ErrorInfo{Reason: code.Name(), Domain: codeDomain})
	if detailsErr != nil {
		return answer.Err() // the message still starts with the code, for a person to read
	}
	return detailed.Err()
}

// CodeName is the name of the code an error from the hub carries, or "" if it carries none.
func CodeName(err error) string {
	for _, detail := range status.Convert(err).Details() {
		information, isInformation := detail.(*errdetails.ErrorInfo)
		if isInformation && information.Domain == codeDomain {
			return information.Reason
		}
	}
	return ""
}
