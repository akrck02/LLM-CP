package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/akrck02/LLM-CP/hub"
	"github.com/akrck02/LLM-CP/protocol"
	"github.com/akrck02/LLM-CP/server"
)

func TestAPointPublishedIsReadBackWhole(test *testing.T) {
	socket := filepath.Join(test.TempDir(), "llmcp.sock")
	listener, err := server.Listen(socket)
	if err != nil {
		test.Fatal(err)
	}
	served := server.New(hub.New(time.Hour, time.Now))
	go served.Serve(listener) // its error only says the test stopped the server
	test.Cleanup(served.Stop)
	for _, arguments := range [][]string{
		{"join", "bob", "--socket", socket},
		{"listen", "load", "--session", "bob", "--socket", socket},
		{"join", "alice", "--socket", socket},
		{"metric", "load", "load.average", "0.5", "--unit", "jobs", "--label", "machine=box", "--session", "alice", "--socket", socket},
	} {
		if err := run(arguments); err != nil {
			test.Fatalf("%v: %v", arguments, err)
		}
	}
	connection, err := grpc.NewClient("unix://"+socket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		test.Fatal(err)
	}
	defer connection.Close()
	read, err := protocol.NewHubClient(connection).Read(context.Background(), &protocol.ReadRequest{Session: "bob"})
	if err != nil || len(read.GetMessages()) != 1 {
		test.Fatalf("bob read %v (%v)", read, err)
	}
	point, err := hub.ParsePoint(read.GetMessages()[0].GetText())
	if err != nil || point.Value != 0.5 || point.Unit != "jobs" || point.Labels["machine"] != "box" || point.At == 0 {
		test.Errorf("bob read the point %+v (%v)", point, err)
	}
}

func TestAValueThatIsNotANumberIsRefused(test *testing.T) {
	if _, err := pointOf("load", "high", "", nil); err == nil {
		test.Error("a word was taken as a value")
	}
}
