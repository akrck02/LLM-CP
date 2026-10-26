// Package hub holds the rules of llmcp in memory: which sessions have joined, which topics each
// one listens to, and which messages each listener has still to read. It knows nothing of gRPC
// or sockets; the server package carries its calls over the wire.
package hub

import (
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/akrck02/LLM-CP/internal/failure"
)

// Message is one text published to a topic. Numbers grow in publish order across all topics.
type Message struct {
	Number    uint64
	Topic     string
	From      string
	Text      string
	Published time.Time
}

// Mode is how a session starts listening to a topic.
type Mode int

const (
	// OnlyNew reads only the messages published from now on.
	OnlyNew Mode = iota
	// AlsoPast reads too the messages the topic already holds.
	AlsoPast
)

// Session is what others can see of a session: its name and the topics it listens to.
type Session struct {
	Name   string
	Topics []string
}

// Hub is the meeting point of the sessions. Its methods are safe to call at the same time.
type Hub struct {
	mutex        sync.Mutex
	sessions     map[string]*session
	topics       map[string][]Message
	published    uint64
	silenceLimit time.Duration
	now          func() time.Time
}

// session is a joined session: when it was last heard of, and how far it has read each topic.
type session struct {
	lastSeen time.Time
	cursors  map[string]int // topic name → index of the first message not read yet
	arrived  chan struct{}  // signalled when a message reaches one of its topics
}

// New makes an empty hub. A session silent for longer than silenceLimit is dropped; now is the
// clock, time.Now outside tests.
func New(silenceLimit time.Duration, now func() time.Time) *Hub {
	return &Hub{
		sessions:     map[string]*session{},
		topics:       map[string][]Message{},
		silenceLimit: silenceLimit,
		now:          now,
	}
}

// Join adds a session with a name, listening to its own topic. Messages sent there while no
// session had the name are waiting for it; those of an earlier session of that name are gone.
func (hub *Hub) Join(name string) error {
	if !validName(name) {
		return failure.Wrap(InvalidName, name, nil)
	}
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	hub.dropSilentSessions()
	if _, taken := hub.sessions[name]; taken {
		return failure.Wrap(NameTaken, name, nil)
	}
	hub.sessions[name] = &session{
		lastSeen: hub.now(),
		cursors:  map[string]int{OwnTopic(name): 0},
		arrived:  make(chan struct{}, 1),
	}
	return nil
}

// Leave removes a session; its name is free again.
func (hub *Hub) Leave(name string) error {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	if _, err := hub.touch(name); err != nil {
		return err
	}
	hub.forget(name)
	return nil
}

// forget removes a session and its own topic, so whoever takes the name next starts with an
// empty inbox. The caller holds the mutex.
func (hub *Hub) forget(name string) {
	delete(hub.sessions, name)
	delete(hub.topics, OwnTopic(name))
}

// Sessions lists the sessions joined right now, by name, with the topics each one listens to.
func (hub *Hub) Sessions(asking string) ([]Session, error) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	if _, err := hub.touch(asking); err != nil {
		return nil, err
	}
	sessions := make([]Session, 0, len(hub.sessions))
	for name, joined := range hub.sessions {
		topics := make([]string, 0, len(joined.cursors))
		for topic := range joined.cursors {
			topics = append(topics, topic)
		}
		slices.Sort(topics)
		sessions = append(sessions, Session{Name: name, Topics: topics})
	}
	slices.SortFunc(sessions, func(first, second Session) int { return strings.Compare(first.Name, second.Name) })
	return sessions, nil
}

// Listen makes a session listen to a topic, from now on or from the topic's first message.
// Listening again to a topic changes nothing.
func (hub *Hub) Listen(name, topic string, mode Mode) error {
	if !validTopic(topic) {
		return failure.Wrap(InvalidName, topic, nil)
	}
	if strings.HasPrefix(topic, ownTopicPrefix) && topic != OwnTopic(name) {
		return failure.Wrap(TopicReserved, topic, nil)
	}
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	joined, err := hub.touch(name)
	if err != nil {
		return err
	}
	if _, listening := joined.cursors[topic]; listening {
		return nil
	}
	joined.cursors[topic] = 0
	if mode == OnlyNew {
		joined.cursors[topic] = len(hub.topics[topic])
	}
	return nil
}

// Unlisten makes a session stop listening to a topic.
func (hub *Hub) Unlisten(name, topic string) error {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	joined, err := hub.touch(name)
	if err != nil {
		return err
	}
	if _, listening := joined.cursors[topic]; !listening || topic == OwnTopic(name) {
		return failure.Wrap(NotListening, topic, nil)
	}
	delete(joined.cursors, topic)
	return nil
}

// Publish adds a message to a topic, where it waits for every session listening to it.
func (hub *Hub) Publish(from, topic, text string) (Message, error) {
	if !validTopic(topic) {
		return Message{}, failure.Wrap(InvalidName, topic, nil)
	}
	if strings.TrimSpace(text) == "" {
		return Message{}, EmptyMessage
	}
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	if _, err := hub.touch(from); err != nil {
		return Message{}, err
	}
	hub.published++
	message := Message{Number: hub.published, Topic: topic, From: from, Text: text, Published: hub.now()}
	hub.topics[topic] = append(hub.topics[topic], message)
	hub.wakeListeners(topic, from)
	return message, nil
}

// wakeListeners tells every session listening to a topic, except the sender, that a message came.
func (hub *Hub) wakeListeners(topic, from string) {
	for name, joined := range hub.sessions {
		if _, listening := joined.cursors[topic]; !listening || name == from {
			continue
		}
		select {
		case joined.arrived <- struct{}{}:
		default: // it has been told already and has not read yet
		}
	}
}

// touch finds a joined session and marks it as heard of now. The caller holds the mutex.
func (hub *Hub) touch(name string) (*session, error) {
	hub.dropSilentSessions()
	joined, found := hub.sessions[name]
	if !found {
		return nil, failure.Wrap(SessionNotFound, name, nil)
	}
	joined.lastSeen = hub.now()
	return joined, nil
}

// dropSilentSessions removes the sessions not heard of for longer than the silence limit, so a
// session that ended without leaving frees its name. The caller holds the mutex.
func (hub *Hub) dropSilentSessions() {
	now := hub.now()
	for name, joined := range hub.sessions {
		if now.Sub(joined.lastSeen) > hub.silenceLimit {
			hub.forget(name)
		}
	}
}
