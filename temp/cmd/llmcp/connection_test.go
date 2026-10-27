package main

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/akrck02/LLM-CP/internal/failure"
)

func TestAnErrorFromTheHubComesBackWithItsCode(test *testing.T) {
	local := errors.New("made here")
	cases := []struct {
		name string
		err  error
		want *failure.Code
	}{
		{"nothing answers", status.Error(codes.Unavailable, "connection refused"), HubNotRunning},
		{"a status whose text only looks like a code", status.Error(codes.Internal, "hub.name-taken: looks like one"), nil},
		{"an error made here", MissingSession, MissingSession},
		{"a plain error", local, nil},
	}
	for _, each := range cases {
		got := fromStatus(each.err, "llmcp.sock")
		if failure.CodeOf(got) != each.want {
			test.Errorf("%s: got %v", each.name, got)
		}
	}
	if fromStatus(nil, "llmcp.sock") != nil {
		test.Error("no error became one")
	}
}
