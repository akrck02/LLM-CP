package main

import (
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/akrck02/LLM-CP/hub"
	"github.com/akrck02/LLM-CP/internal/failure"
	"github.com/akrck02/LLM-CP/protocol"
	"github.com/akrck02/LLM-CP/server"
)

// connect opens a client to the hub on a socket. The connection is made on the first call.
func connect(socket string) (protocol.HubClient, *grpc.ClientConn, error) {
	connection, err := grpc.NewClient("unix://"+socket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, failure.Wrap(HubNotRunning, socket, err)
	}
	return protocol.NewHubClient(connection), connection, nil
}

// fromStatus turns an error from the hub back into an error with its code: the hub's own code,
// or HubNotRunning when nothing answers on the socket. Errors made here pass through unchanged.
func fromStatus(err error, socket string) error {
	if err == nil {
		return nil
	}
	answer, isStatus := status.FromError(err)
	if !isStatus {
		return err
	}
	if answer.Code() == codes.Unavailable {
		return failure.Wrap(HubNotRunning, socket, err)
	}
	code := hub.CodeNamed(server.CodeName(err))
	if code == nil {
		return err
	}
	if answer.Message() == code.Error() {
		return code
	}
	return failure.Wrap(code, strings.TrimPrefix(answer.Message(), code.Name()+": "), nil)
}
