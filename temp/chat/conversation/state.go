package conversation

import (
	"slices"
	"time"
)

// Contact is a session on the hub, as the contacts list shows it.
type Contact struct {
	Name   string   `json:"name"`
	Topics []string `json:"topics"` // what it listens to
	Groups []string `json:"groups"` // the groups it is a member of
	// TypingUntil is when it stops showing as typing, in Unix milliseconds; 0 when it is not.
	TypingUntil int64 `json:"typingUntil"`
}

// IsTyping tells whether the contact shows as typing at a time.
func (contact Contact) IsTyping(now time.Time) bool {
	return now.UnixMilli() < contact.TypingUntil
}

// Group is a topic with a theme and chosen members.
type Group struct {
	Name    string   `json:"name"`
	Theme   string   `json:"theme"`
	Members []string `json:"members"`
}

// State is what the hub says besides its messages, whole each time it changes.
type State struct {
	Contacts []Contact   `json:"contacts"`
	Groups   []Group     `json:"groups"`
	Archived []string    `json:"archived"`
	Kinds    []TopicKind `json:"kinds"`
}

// SetState takes the hub's newest state in place of the one before.
func (log *Log) SetState(state State) { log.state = state }

// Contacts lists the sessions on the hub, as the hub sorted them.
func (log *Log) Contacts() []Contact { return log.state.Contacts }

// Group is the group of a topic, or nil when the topic is not one.
func (log *Log) Group(topic string) *Group {
	for index := range log.state.Groups {
		if log.state.Groups[index].Name == topic {
			return &log.state.Groups[index]
		}
	}
	return nil
}

// Groups lists every group.
func (log *Log) Groups() []Group { return log.state.Groups }

// IsArchived tells whether a topic is kept out of the list of chats.
func (log *Log) IsArchived(topic string) bool { return slices.Contains(log.state.Archived, topic) }

// Archived lists the archived chats, the newest first.
func (log *Log) Archived() []*Chat {
	return slices.DeleteFunc(log.allChats(), func(chat *Chat) bool { return !log.IsArchived(chat.Topic) })
}

// TypingIn names who is typing at a time among those who listen to a topic, so the dots show
// in the chats where the answer may land.
func (log *Log) TypingIn(topic string, now time.Time) []string {
	var typing []string
	for _, contact := range log.state.Contacts {
		if contact.IsTyping(now) && slices.Contains(contact.Topics, topic) {
			typing = append(typing, contact.Name)
		}
	}
	return typing
}

// NextTypingEnd is the soonest time after now when someone stops showing as typing, so the page
// can draw again then; zero when nobody is typing.
func (log *Log) NextTypingEnd(now time.Time) time.Time {
	var soonest time.Time
	for _, contact := range log.state.Contacts {
		ends := time.UnixMilli(contact.TypingUntil)
		if contact.IsTyping(now) && (soonest.IsZero() || ends.Before(soonest)) {
			soonest = ends
		}
	}
	return soonest
}

// Action is what a page asks the server to do besides publishing: make a group (Topic is its
// name), archive a chat or bring it back, subscribe to a topic or not. As is who asks.
type Action struct {
	As         string   `json:"as"`
	Topic      string   `json:"topic"`
	Theme      string   `json:"theme,omitempty"`
	Members    []string `json:"members,omitempty"`
	Subscribed bool     `json:"subscribed,omitempty"`
}

// WithArchived is the state with one more topic archived, to show it before the hub says so.
func (log *Log) WithArchived(topic string) State {
	state := log.state
	if !log.IsArchived(topic) {
		state.Archived = append(slices.Clone(state.Archived), topic)
	}
	return state
}
