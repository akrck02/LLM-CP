package hub

import (
	"cmp"
	"slices"
)

// watcherRoom is how many events a watcher can fall behind before the hub lets it go: a person
// looking on must never slow the sessions down. A watcher let go finds its channel closed, and
// watches again to get everything.
const watcherRoom = 256

// Event is what a watcher hears: a message, or the hub's state after it changed. A state event
// holds the whole lists, not what changed: they are small, and a watcher never has to merge.
type Event struct {
	Message *Message // nil for a state event
	State   *State   // nil for a message event
}

// State is everything about the hub a person looking on sees besides the messages.
type State struct {
	Contacts []Contact
	Groups   []Group
	Archived []string
	Kinds    []TopicKind
}

// Watcher receives every message published from the moment it started watching, and the hub's
// state each time it changes.
type Watcher struct {
	Events <-chan Event
	events chan Event
}

// Watch starts watching every topic. With past, it gives first the messages the hub already holds,
// oldest first. Its first event is the hub's state. Call Unwatch when done.
func (hub *Hub) Watch(past bool) (*Watcher, []Message) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	events := make(chan Event, watcherRoom)
	watcher := &Watcher{Events: events, events: events}
	hub.watchers = append(hub.watchers, watcher)
	state := hub.state()
	watcher.events <- Event{State: &state}
	if !past {
		return watcher, nil
	}
	var held []Message
	for _, topic := range hub.topics {
		held = append(held, topic...)
	}
	slices.SortFunc(held, func(first, second Message) int { return cmp.Compare(first.Number, second.Number) })
	return watcher, held
}

// Unwatch stops a watcher; its channel is closed.
func (hub *Hub) Unwatch(watcher *Watcher) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	index := slices.Index(hub.watchers, watcher)
	if index < 0 {
		return
	}
	hub.watchers = slices.Delete(hub.watchers, index, index+1)
	close(watcher.events)
}

// tellWatchers hands a message to every watcher. The caller holds the mutex.
func (hub *Hub) tellWatchers(message Message) {
	hub.tell(Event{Message: &message})
}

// stateChanged hands the hub's new state to every watcher. The caller holds the mutex.
func (hub *Hub) stateChanged() {
	if len(hub.watchers) == 0 {
		return
	}
	state := hub.state()
	hub.tell(Event{State: &state})
}

// tell hands an event to every watcher, and lets go of those too far behind to take it: a
// missed event would leave them with a wrong state for good. The caller holds the mutex.
func (hub *Hub) tell(event Event) {
	hub.watchers = slices.DeleteFunc(hub.watchers, func(watcher *Watcher) bool {
		select {
		case watcher.events <- event:
			return false
		default:
			close(watcher.events)
			return true
		}
	})
}

// state is the hub's state now. The caller holds the mutex.
func (hub *Hub) state() State {
	return State{Contacts: hub.contacts(), Groups: hub.groupList(), Archived: hub.archivedList(), Kinds: hub.kindList()}
}
