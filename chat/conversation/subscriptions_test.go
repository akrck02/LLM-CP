package conversation

import (
	"slices"
	"testing"
)

// subscribedLog is a log read by alice, subscribed to "api" alone, on a hub with three topics,
// a group she is in, a group she is not in, and a message to her.
func subscribedLog() *Log {
	var log Log
	log.Add(message(1, "api", "bob", "one"))
	log.Add(message(2, "docs", "bob", "two"))
	log.Add(message(3, "release", "carol", "three"))
	log.Add(message(4, "@alice", "bob", "four"))
	log.SetState(State{Groups: []Group{
		{Name: "team", Members: []string{"alice", "bob"}},
		{Name: "others", Members: []string{"bob", "carol"}},
	}})
	log.SetSubscriptions("alice", []string{"api"})
	return &log
}

// The Topics section is what the person follows, not everything on the hub: a member always
// sees their group.
func TestOnlySubscribedTopicsAndGroupsAreListed(test *testing.T) {
	if topics := topicsOf(subscribedLog().Topics()); !slices.Equal(topics, []string{"api", "team"}) {
		test.Errorf("the topics are %v", topics)
	}
}

// What the person does not follow is still one tap away, by name.
func TestTheOtherTopicsAreMoreTopics(test *testing.T) {
	if topics := topicsOf(subscribedLog().MoreTopics()); !slices.Equal(topics, []string{"docs", "others", "release"}) {
		test.Errorf("the more topics are %v", topics)
	}
}

// Someone new follows nothing, and a message to them still reaches them.
func TestSomeoneNewSeesNoTopicsButTheirInbox(test *testing.T) {
	log := subscribedLog()
	log.SetSubscriptions("dave", nil)
	if topics := topicsOf(log.Topics()); len(topics) != 0 {
		test.Errorf("dave sees the topics %v", topics)
	}
	inbox := false
	for _, chat := range log.Chats() {
		inbox = inbox || chat.Topic == "@alice"
	}
	if !inbox {
		test.Error("the inbox is not among the chats")
	}
}

// Until the page knows the person's subscriptions, nothing disappears: the chat lists every
// topic, as it did before subscriptions existed.
func TestEveryTopicIsListedUntilTheSubscriptionsAreKnown(test *testing.T) {
	var log Log
	log.Add(message(1, "api", "bob", "one"))
	log.Add(message(2, "docs", "bob", "two"))
	if topics := topicsOf(log.Topics()); !slices.Equal(topics, []string{"docs", "api"}) {
		test.Errorf("the topics are %v", topics)
	}
}
