package server

import (
	"context"
	"testing"

	"github.com/akrck02/LLM-CP/protocol"
)

func TestAWatcherIsToldWhichTopicsAreMetrics(test *testing.T) {
	_, client := startHub(test)
	request := context.Background()
	if _, err := client.Join(request, &protocol.SessionRequest{Session: "alice"}); err != nil {
		test.Fatal(err)
	}
	if _, err := client.Declare(request, &protocol.KindRequest{Session: "alice", Topic: "load", Kind: "metrics"}); err != nil {
		test.Fatal(err)
	}
	stream, err := client.Watch(request, &protocol.WatchRequest{})
	if err != nil {
		test.Fatal(err)
	}
	event, err := stream.Recv()
	if err != nil {
		test.Fatal(err)
	}
	kinds := event.GetState().GetKinds()
	if len(kinds) != 1 || kinds[0].GetTopic() != "load" || kinds[0].GetKind() != "metrics" {
		test.Errorf("the watcher was told %v", kinds)
	}
}
