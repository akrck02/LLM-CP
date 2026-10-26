package hub

import (
	"cmp"
	"context"
	"slices"
	"time"
)

// Read gives a session the messages waiting in the topics it listens to, oldest first, and marks
// them as read. Its own messages are skipped. If none is waiting, it waits up to the given time
// for one to arrive; a wait of zero returns at once.
func (hub *Hub) Read(request context.Context, name string, wait time.Duration) ([]Message, error) {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	for {
		messages, arrived, err := hub.takeWaiting(name)
		if err != nil || len(messages) > 0 || wait <= 0 {
			return messages, err
		}
		select {
		case <-arrived: // read again: the wake-up may be stale, or another read took the message
		case <-timer.C:
			return nil, nil
		case <-request.Done():
			return nil, nil
		}
	}
}

// takeWaiting collects the unread messages of a session and moves its cursors past them. It also
// gives the channel that tells when a new one arrives.
func (hub *Hub) takeWaiting(name string) ([]Message, <-chan struct{}, error) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	joined, err := hub.touch(name)
	if err != nil {
		return nil, nil, err
	}
	select {
	case <-joined.arrived: // what woke it is about to be read
	default:
	}
	var messages []Message
	for topic, cursor := range joined.cursors {
		held := hub.topics[topic]
		for _, message := range held[cursor:] {
			if message.From != name {
				messages = append(messages, message)
			}
		}
		joined.cursors[topic] = len(held)
	}
	slices.SortFunc(messages, func(first, second Message) int { return cmp.Compare(first.Number, second.Number) })
	return messages, joined.arrived, nil
}
