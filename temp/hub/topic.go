package hub

import (
	"slices"

	"github.com/akrck02/LLM-CP/internal/failure"
)

// Kind is what a topic carries: sentences between sessions, or points of measurements.
type Kind string

// The kinds a topic can have. A topic nobody declared is a conversation.
const (
	Conversation Kind = "conversation"
	Metrics      Kind = "metrics"
)

// TopicKind is a declared topic as a person looking on sees it: its kind and who publishes to it.
type TopicKind struct {
	Topic      string
	Kind       Kind
	Publishers []string
}

// Declare gives a topic its kind, for the topic's whole life. Declaring the kind it has already
// is fine; another one is refused, and the topic must be removed to be made again.
func (hub *Hub) Declare(by, topic string, kind Kind) error {
	if !validTopic(topic) {
		return failure.Wrap(InvalidName, topic, nil)
	}
	if kind != Conversation && kind != Metrics {
		return failure.Wrap(UnknownKind, string(kind), nil)
	}
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	if _, err := hub.touch(by); err != nil {
		return err
	}
	if current := hub.kindOf(topic); current != kind && (hub.kinds[topic] != "" || len(hub.topics[topic]) > 0) {
		return failure.Wrap(KindTaken, topic+" is a "+string(current), nil)
	}
	hub.kinds[topic] = kind
	hub.stateChanged()
	return hub.keep()
}

// RemoveTopic deletes a topic: its messages, its kind and every session's place in it.
func (hub *Hub) RemoveTopic(by, topic string) error {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	if _, err := hub.touch(by); err != nil {
		return err
	}
	delete(hub.topics, topic)
	delete(hub.kinds, topic)
	delete(hub.archived, topic)
	for _, joined := range hub.sessions {
		delete(joined.cursors, topic)
	}
	hub.stateChanged()
	return hub.keep()
}

// kindOf is a topic's kind; a conversation unless declared otherwise. The caller holds the mutex.
func (hub *Hub) kindOf(topic string) Kind {
	if kind, declared := hub.kinds[topic]; declared {
		return kind
	}
	return Conversation
}

// kindList is every declared topic with its publishers, in name order. The caller holds the mutex.
func (hub *Hub) kindList() []TopicKind {
	var list []TopicKind
	for topic, kind := range hub.kinds {
		var publishers []string
		for _, message := range hub.topics[topic] {
			if !slices.Contains(publishers, message.From) {
				publishers = append(publishers, message.From)
			}
		}
		slices.Sort(publishers)
		list = append(list, TopicKind{Topic: topic, Kind: kind, Publishers: publishers})
	}
	slices.SortFunc(list, func(first, second TopicKind) int { return compareText(first.Topic, second.Topic) })
	return list
}

func compareText(first, second string) int {
	switch {
	case first < second:
		return -1
	case first > second:
		return 1
	}
	return 0
}

// dropOlderThan forgets a topic's messages published before a moment, and moves every session's
// place in it back by as many. The caller holds the mutex.
func (hub *Hub) dropOlderThan(topic string, oldest int64) {
	messages := hub.topics[topic]
	gone := 0
	for gone < len(messages) && messages[gone].Published.UnixMilli() < oldest {
		gone++
	}
	if gone == 0 {
		return
	}
	hub.topics[topic] = slices.Clone(messages[gone:])
	for _, joined := range hub.sessions {
		if cursor, listening := joined.cursors[topic]; listening {
			joined.cursors[topic] = max(cursor-gone, 0)
		}
	}
}
