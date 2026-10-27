package main

import (
	"testing"

	"github.com/akrck02/LLM-CP/chat/conversation"
)

// The Topics tab counts what waits unread where the person looks: a topic they do not follow
// must not light the tab up.
func TestTheTopicsTabCountsOnlyFollowedTopics(test *testing.T) {
	chat := &screen{me: "alice"}
	chat.log.Add(conversation.Message{Number: 1, Topic: "api", From: "bob", Text: "one"})
	chat.log.Add(conversation.Message{Number: 2, Topic: "docs", From: "bob", Text: "two"})
	chat.log.Add(conversation.Message{Number: 3, Topic: "docs", From: "bob", Text: "three"})
	chat.log.SetSubscriptions("alice", []string{"api"})
	if unread := chat.unreadIn(true); unread != 1 {
		test.Errorf("the Topics tab counts %d unread", unread)
	}
}
