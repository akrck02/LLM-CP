package server

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/akrck02/LLM-CP/hub"
	"github.com/akrck02/LLM-CP/protocol"
)

// startHub serves a new hub on a socket in a temporary folder and gives its path and a client.
func startHub(test *testing.T) (string, protocol.HubClient) {
	test.Helper()
	path := filepath.Join(test.TempDir(), "llmcp.sock")
	listener, err := Listen(path)
	if err != nil {
		test.Fatal(err)
	}
	server := New(hub.New(time.Hour, time.Now))
	go server.Serve(listener) // its error only says the test stopped the server
	test.Cleanup(server.Stop)
	connection, err := grpc.NewClient("unix://"+path, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() { connection.Close() })
	return path, protocol.NewHubClient(connection)
}

func TestTheSocketIsReadableOnlyByItsOwner(test *testing.T) {
	path, _ := startHub(test)
	information, err := os.Stat(path)
	if err != nil {
		test.Fatal(err)
	}
	if permissions := information.Mode().Perm(); permissions != ownerOnly {
		test.Fatalf("the socket has mode %v, want %v", permissions, ownerOnly)
	}
}

func TestASecondHubCannotTakeASocketInUse(test *testing.T) {
	path, _ := startHub(test)
	if _, err := Listen(path); !errors.Is(err, SocketInUse) {
		test.Fatalf("got %v, want %s", err, SocketInUse.Name())
	}
}

func TestAMessageTravelsBetweenTwoSessionsOverTheSocket(test *testing.T) {
	_, client := startHub(test)
	request := context.Background()
	for _, name := range []string{"alice", "bob"} {
		if _, err := client.Join(request, &protocol.SessionRequest{Session: name}); err != nil {
			test.Fatal(err)
		}
	}
	if _, err := client.Listen(request, &protocol.ListenRequest{Session: "bob", Topic: "api"}); err != nil {
		test.Fatal(err)
	}
	if _, err := client.Publish(request, &protocol.PublishRequest{Session: "alice", Topic: "api", Text: "hello"}); err != nil {
		test.Fatal(err)
	}
	read, err := client.Read(request, &protocol.ReadRequest{Session: "bob"})
	if err != nil || len(read.Messages) != 1 || read.Messages[0].Text != "hello" || read.Messages[0].From != "alice" {
		test.Fatalf("got %v, %v", read, err)
	}
}

func TestAnErrorArrivesWithItsCode(test *testing.T) {
	_, client := startHub(test)
	_, err := client.Read(context.Background(), &protocol.ReadRequest{Session: "nobody"})
	if name := CodeName(err); name != hub.SessionNotFound.Name() {
		test.Fatalf("the error %v carries the code %q", err, name)
	}
	if message := status.Convert(err).Message(); !strings.HasPrefix(message, hub.SessionNotFound.Name()+":") {
		test.Fatalf("the message %q does not start with its code, for a person", message)
	}
}
