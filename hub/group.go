package hub

import (
	"slices"
	"strings"

	"github.com/akrck02/LLM-CP/internal/failure"
)

// Group is a topic with a theme and the sessions chosen to talk in it. Its members listen to it
// as soon as they are on the hub, and only they may publish there.
type Group struct {
	Name    string
	Theme   string
	Members []string // by name, sorted; a member may not have joined yet
}

// CreateGroup makes a group of a topic, a theme and its members, and makes each member already on
// the hub listen to it, from now on. The session that creates it need not be a member.
func (hub *Hub) CreateGroup(by, name, theme string, members []string) error {
	if !validName(name) {
		return failure.Wrap(InvalidName, name, nil)
	}
	for _, member := range members {
		if !validName(member) {
			return failure.Wrap(InvalidName, member, nil)
		}
	}
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	if _, err := hub.touch(by); err != nil {
		return err
	}
	if _, exists := hub.groups[name]; exists {
		return failure.Wrap(GroupExists, name, nil)
	}
	group := &Group{Name: name, Theme: strings.TrimSpace(theme)}
	hub.groups[name] = group
	for _, member := range members {
		hub.addMember(group, member)
	}
	hub.stateChanged()
	return hub.keep()
}

// AddMember adds a session to a group; it listens to the group from now on.
func (hub *Hub) AddMember(by, group, member string) error {
	if !validName(member) {
		return failure.Wrap(InvalidName, member, nil)
	}
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	found, err := hub.findChangeableGroup(by, group)
	if err != nil {
		return err
	}
	hub.addMember(found, member)
	hub.stateChanged()
	return hub.keep()
}

// RemoveMember takes a session out of a group; it stops listening to it.
func (hub *Hub) RemoveMember(by, group, member string) error {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	found, err := hub.findChangeableGroup(by, group)
	if err != nil {
		return err
	}
	index := slices.Index(found.Members, member)
	if index < 0 {
		return failure.Wrap(NotAMember, member, nil)
	}
	found.Members = slices.Delete(found.Members, index, index+1)
	if joined, online := hub.sessions[member]; online {
		delete(joined.cursors, group)
	}
	hub.stateChanged()
	return hub.keep()
}

// RemoveGroup turns a group back into a plain topic: its messages stay, and so do its listeners;
// it is no longer kept.
func (hub *Hub) RemoveGroup(by, group string) error {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	if _, err := hub.findChangeableGroup(by, group); err != nil {
		return err
	}
	delete(hub.groups, group)
	hub.stateChanged()
	return hub.keep()
}

// Groups lists every group by name, each a copy the caller may keep.
func (hub *Hub) Groups() []Group {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	return hub.groupList()
}

// groupList lists every group by name, as copies. The caller holds the mutex.
func (hub *Hub) groupList() []Group {
	groups := make([]Group, 0, len(hub.groups))
	for _, group := range hub.groups {
		copied := *group
		copied.Members = slices.Clone(group.Members)
		groups = append(groups, copied)
	}
	slices.SortFunc(groups, func(first, second Group) int { return strings.Compare(first.Name, second.Name) })
	return groups
}

// findGroup marks a session as heard of and finds a group. The caller holds the mutex.
func (hub *Hub) findGroup(by, name string) (*Group, error) {
	if _, err := hub.touch(by); err != nil {
		return nil, err
	}
	group, found := hub.groups[name]
	if !found {
		return nil, failure.Wrap(GroupNotFound, name, nil)
	}
	return group, nil
}

// findChangeableGroup finds a group whose members may change: any but a pair. The caller holds the
// mutex.
func (hub *Hub) findChangeableGroup(by, name string) (*Group, error) {
	group, err := hub.findGroup(by, name)
	if err != nil {
		return nil, err
	}
	if IsPair(name) {
		return nil, failure.Wrap(PairFixed, name, nil)
	}
	return group, nil
}

// pairFor is where a message from a session to a topic goes: to '@name' of another session, or
// to a pair's name, it is their pair, made if it is not there yet (made says so); to anything
// else, the topic itself. The caller holds the mutex.
func (hub *Hub) pairFor(from, topic string) (destination string, made bool) {
	other, isAddress := strings.CutPrefix(topic, ownTopicPrefix)
	switch {
	case isAddress && other != from:
		topic = PairOf(from, other)
	case !IsPair(topic):
		return topic, false
	}
	first, second, _ := pairMembers(topic)
	if from != first && from != second {
		return topic, false // not its pair: mayPublish refuses it
	}
	if _, exists := hub.groups[topic]; exists {
		return topic, false
	}
	pair := &Group{Name: topic}
	hub.groups[topic] = pair
	hub.addMember(pair, first)
	hub.addMember(pair, second)
	return topic, true
}

// addMember puts a session in a group, and makes it listen if it is on the hub. The caller holds
// the mutex.
func (hub *Hub) addMember(group *Group, member string) {
	if !slices.Contains(group.Members, member) {
		group.Members = append(group.Members, member)
		slices.Sort(group.Members)
	}
	if joined, online := hub.sessions[member]; online {
		hub.listenFromNow(joined, group.Name)
	}
}

// listenToGroups makes a session that just joined listen to the groups it is a member of. The
// caller holds the mutex.
func (hub *Hub) listenToGroups(name string, joined *session) {
	for _, group := range hub.groups {
		if slices.Contains(group.Members, name) {
			hub.listenFromNow(joined, group.Name)
		}
	}
}

// listenFromNow makes a session listen to a topic from its next message, unless it already does.
// The caller holds the mutex.
func (hub *Hub) listenFromNow(joined *session, topic string) {
	if _, listening := joined.cursors[topic]; !listening {
		joined.cursors[topic] = len(hub.topics[topic])
	}
}

// mayPublish tells whether a session may publish to a topic: anyone may, except to a group it is
// not a member of, or to a pair that is not its own. The caller holds the mutex.
func (hub *Hub) mayPublish(from, topic string) error {
	group, isGroup := hub.groups[topic]
	if isGroup && !slices.Contains(group.Members, from) || !isGroup && IsPair(topic) {
		return failure.Wrap(NotAMember, from+" in "+topic, nil)
	}
	return nil
}
