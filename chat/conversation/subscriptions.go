package conversation

import (
	"slices"
	"strings"
)

// SetSubscriptions says who reads the log and which topics they are subscribed to. From then on
// Topics lists only those and the reader's groups, and MoreTopics the others.
func (log *Log) SetSubscriptions(reader string, topics []string) {
	log.reader, log.subscribed, log.knowsSubscriptions = reader, slices.Clone(topics), true
}

// Subscriptions are the topics the reader is subscribed to. The list is a copy.
func (log *Log) Subscriptions() []string { return slices.Clone(log.subscribed) }

// IsSubscribed tells whether the reader follows a topic: one they subscribed to, or a group they
// are a member of. Before the subscriptions are known, every topic is followed.
func (log *Log) IsSubscribed(topic string) bool {
	if !log.knowsSubscriptions || slices.Contains(log.subscribed, topic) {
		return true
	}
	group := log.Group(topic)
	return group != nil && slices.Contains(group.Members, log.reader)
}

// WithSubscription is the list of subscriptions with one topic added or taken out, to show the
// change before the server says it is kept.
func (log *Log) WithSubscription(topic string, subscribed bool) []string {
	topics := slices.DeleteFunc(slices.Clone(log.subscribed), func(each string) bool { return each == topic })
	if subscribed {
		topics = append(topics, topic)
		slices.Sort(topics)
	}
	return topics
}

// Topics lists the chats not archived that are topics the reader follows, the newest first.
// Inboxes are not topics: a message to someone always shows.
func (log *Log) Topics() []*Chat {
	return slices.DeleteFunc(log.Chats(), func(chat *Chat) bool { return chat.IsInbox() || !log.IsSubscribed(chat.Topic) })
}

// MoreTopics lists the chats not archived that are topics the reader does not follow, by name,
// so one is easy to find and subscribe to.
func (log *Log) MoreTopics() []*Chat {
	chats := slices.DeleteFunc(log.Chats(), func(chat *Chat) bool { return chat.IsInbox() || log.IsSubscribed(chat.Topic) })
	slices.SortFunc(chats, func(first, second *Chat) int { return strings.Compare(first.Topic, second.Topic) })
	return chats
}
