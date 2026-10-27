package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/akrck02/LLM-CP/protocol"
)

var twoMessages = []*protocol.Message{
	{Number: 1, Topic: "api", From: "alice", Text: "the API changed"},
	{Number: 2, Topic: "@bob", From: "carol", Text: "first line\nsecond line"},
}

func TestReadPrintsOneLinePerMessage(test *testing.T) {
	output := &strings.Builder{}
	if err := newPrinter(output, false).messages(twoMessages); err != nil {
		test.Fatal(err)
	}
	want := "#1 api alice: the API changed\n#2 @bob carol: first line\\nsecond line\n"
	if output.String() != want {
		test.Fatalf("got %q, want %q", output.String(), want)
	}
}

func TestReadWithJSONGivesAListEvenWhenEmpty(test *testing.T) {
	cases := map[string][]*protocol.Message{"empty": nil, "two": twoMessages}
	for name, messages := range cases {
		output := &strings.Builder{}
		if err := newPrinter(output, true).messages(messages); err != nil {
			test.Fatal(err)
		}
		var decoded []map[string]any
		if err := json.Unmarshal([]byte(output.String()), &decoded); err != nil || len(decoded) != len(messages) {
			test.Errorf("%s: got %q, %v", name, output.String(), err)
		}
	}
}

func TestOptionsAreReadWhereverTheyAre(test *testing.T) {
	given, err := parseOptions([]string{"api", "--session", "alice", "hello", "--json", "world"})
	if err != nil {
		test.Fatal(err)
	}
	if given.session != "alice" || !given.json || strings.Join(given.arguments, " ") != "api hello world" {
		test.Fatalf("got %+v", given)
	}
}

func TestTextAfterADoubleDashIsNeverAnOption(test *testing.T) {
	given, err := parseOptions([]string{"publish", "api", "--session", "alice", "--", "use", "-v", "--json"})
	if err != nil {
		test.Fatal(err)
	}
	if given.json || strings.Join(given.arguments, " ") != "publish api use -v --json" {
		test.Fatalf("got %+v", given)
	}
}

func TestAWaitLongerThanTheSilenceLimitIsRefused(test *testing.T) {
	if _, err := parseOptions([]string{"read", "--wait", "4294967296"}); !errors.Is(err, BadOption) {
		test.Fatalf("got %v", err)
	}
}

func TestGroupsPrintOneLineEach(test *testing.T) {
	output := &strings.Builder{}
	groups := []*protocol.Group{
		{Name: "api", Theme: "the tasks API", Members: []string{"alice", "bob"}},
		{Name: "docs", Members: []string{"carol"}},
	}
	if err := newPrinter(output, false).groups(groups); err != nil {
		test.Fatal(err)
	}
	want := "api (alice, bob): the tasks API\ndocs (carol)\n"
	if output.String() != want {
		test.Fatalf("got %q, want %q", output.String(), want)
	}
}
