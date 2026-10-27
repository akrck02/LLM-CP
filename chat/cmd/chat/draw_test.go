package main

import (
	"testing"

	"github.com/akrck02/LLM-CP/chat/conversation"
)

func TestALongChatDrawsOnlyItsLastMessages(test *testing.T) {
	messages := make([]conversation.Message, 100)
	for index := range messages {
		messages[index].Number = uint64(index)
	}
	shown, hidden := lastOf(messages, 60)
	if len(shown) != 60 || hidden != 40 || shown[0].Number != 40 {
		test.Errorf("shown %d from #%d, hidden %d", len(shown), shown[0].Number, hidden)
	}
}
