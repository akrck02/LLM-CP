package hub

import (
	"slices"
	"strings"
	"time"

	"github.com/akrck02/LLM-CP/internal/failure"
)

// TypingLasts is how long a session shows as typing after it last said so. A session that thinks
// for longer says it again; one that ends without saying it stopped stops showing on its own.
const TypingLasts = 30 * time.Second

// Contact is what the contacts list shows of a session on the hub.
type Contact struct {
	Name   string
	Topics []string // what it listens to, sorted
	Groups []string // the groups it is a member of, sorted
	Typing bool
	// TypingUntil is when it stops showing as typing unless it says so again; zero when it is not.
	// A watcher hides the dots then on its own: the hub sends nothing when the time runs out.
	TypingUntil time.Time
	LastSeen    time.Time
}

// Typing tells the hub that a session started or stopped thinking of an answer.
func (hub *Hub) Typing(name string, typing bool) error {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	joined, err := hub.touch(name)
	if err != nil {
		return err
	}
	joined.typingUntil = time.Time{}
	if typing {
		joined.typingUntil = hub.now().Add(TypingLasts)
	}
	hub.stateChanged()
	return nil
}

// Contacts lists every session on the hub, by name, with what it is doing.
func (hub *Hub) Contacts() []Contact {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	hub.dropSilentSessions()
	return hub.contacts()
}

// contacts lists the sessions on the hub. The caller holds the mutex.
func (hub *Hub) contacts() []Contact {
	now := hub.now()
	contacts := make([]Contact, 0, len(hub.sessions))
	for name, joined := range hub.sessions {
		contacts = append(contacts, Contact{
			Name:        name,
			Topics:      joined.topics(),
			Groups:      hub.groupsOf(name),
			Typing:      now.Before(joined.typingUntil),
			TypingUntil: joined.typingUntil,
			LastSeen:    joined.lastSeen,
		})
	}
	slices.SortFunc(contacts, func(first, second Contact) int { return strings.Compare(first.Name, second.Name) })
	return contacts
}

// Archive keeps a topic out of the list of chats; its messages stay, and new ones leave it there.
func (hub *Hub) Archive(by, topic string) error {
	return hub.setArchived(by, topic, true)
}

// Restore brings an archived topic back to the list of chats.
func (hub *Hub) Restore(by, topic string) error {
	return hub.setArchived(by, topic, false)
}

// setArchived archives a topic or brings it back, for a session on the hub.
func (hub *Hub) setArchived(by, topic string, archived bool) error {
	if !validTopic(topic) {
		return failure.Wrap(InvalidName, topic, nil)
	}
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	if _, err := hub.touch(by); err != nil {
		return err
	}
	if archived {
		hub.archived[topic] = true
	} else {
		delete(hub.archived, topic)
	}
	hub.stateChanged()
	return hub.keep()
}

// Archived lists the archived topics, sorted.
func (hub *Hub) Archived() []string {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	return hub.archivedList()
}

// archivedList lists the archived topics, sorted. The caller holds the mutex.
func (hub *Hub) archivedList() []string {
	topics := make([]string, 0, len(hub.archived))
	for topic := range hub.archived {
		topics = append(topics, topic)
	}
	slices.Sort(topics)
	return topics
}

// groupsOf lists the groups a session is a member of, sorted. The caller holds the mutex.
func (hub *Hub) groupsOf(name string) []string {
	var groups []string
	for _, group := range hub.groups {
		if slices.Contains(group.Members, name) {
			groups = append(groups, group.Name)
		}
	}
	slices.Sort(groups)
	return groups
}

// topics lists what a session listens to, sorted.
func (joined *session) topics() []string {
	topics := make([]string, 0, len(joined.cursors))
	for topic := range joined.cursors {
		topics = append(topics, topic)
	}
	slices.Sort(topics)
	return topics
}
