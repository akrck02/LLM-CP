package main

import (
	"testing"

	"github.com/akrck02/LLM-CP/chat/conversation"
)

func TestAMetricsTopicIsNeverOpenedAsTheFirstChat(test *testing.T) {
	chat := &screen{}
	chat.log.SetState(conversation.State{Kinds: []conversation.TopicKind{{Topic: "tests.results", Kind: "metrics"}}})
	chat.openFirstChat("tests.results")
	if chat.open != "" {
		test.Fatalf("opened %q, a metrics topic", chat.open)
	}
	chat.openFirstChat("api")
	if chat.open != "api" {
		test.Fatalf("opened %q, want the conversation api", chat.open)
	}
}
