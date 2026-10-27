package conversation

import (
	"slices"
	"strings"
)

// A pair is the conversation between two sessions: a group of those two, named after both,
// sorted and joined by '+' ("alice+api"). The hub makes it with their first message to each
// other, and files every later one there, so both sides read one conversation.

// pairSeparator joins a pair's two names; session names never hold it.
const pairSeparator = "+"

// PairOf is the name of the conversation between two sessions, whoever writes first.
func PairOf(first, second string) string {
	if second < first {
		first, second = second, first
	}
	return first + pairSeparator + second
}

// pairMembers are a pair's two sessions; false when the topic is not a pair.
func pairMembers(topic string) (first, second string, isPair bool) {
	first, second, found := strings.Cut(topic, pairSeparator)
	if !found || first == "" || second == "" || strings.Contains(second, pairSeparator) {
		return "", "", false
	}
	return first, second, true
}

// IsPair tells whether a topic is the conversation between two sessions.
func IsPair(topic string) bool {
	_, _, isPair := pairMembers(topic)
	return isPair
}

// Partner is who the person talks to in a pair they are in; false for any other topic.
func Partner(topic, me string) (string, bool) {
	first, second, isPair := pairMembers(topic)
	switch {
	case !isPair:
		return "", false
	case first == me:
		return second, true
	case second == me:
		return first, true
	}
	return "", false
}

// IsReadOnly tells whether the person may only read a topic: a pair between two other sessions,
// where the hub refuses a third voice.
func IsReadOnly(topic, me string) bool {
	_, isMine := Partner(topic, me)
	return IsPair(topic) && !isMine
}

// Name is how a chat is named on screen for the person: a pair they are in by the other
// session's name, a pair between two others by both ("alice + api"), any other topic by its Title.
func Name(topic, me string) string {
	if partner, isMine := Partner(topic, me); isMine {
		return partner
	}
	if first, second, isPair := pairMembers(topic); isPair {
		return first + " " + pairSeparator + " " + second
	}
	return Title(topic)
}

// ConversationWith is where the person talks to a session: their pair, which the first message
// makes; the session's inbox while the person's name is not known.
func ConversationWith(contact, me string) string {
	if me == "" || me == contact {
		return "@" + contact
	}
	return PairOf(me, contact)
}

// Sessions lists the chats not archived where the person talks to sessions: the pairs they are
// in, and the inboxes from before pairs, the newest first.
func (log *Log) Sessions(me string) []*Chat {
	return slices.DeleteFunc(log.Chats(), func(chat *Chat) bool {
		_, isMine := Partner(chat.Topic, me)
		return !chat.IsInbox() && !isMine
	})
}

// BetweenSessions lists the pairs the person is not in, by name: sessions talking to each other,
// which the person may read.
func (log *Log) BetweenSessions(me string) []*Chat {
	chats := slices.DeleteFunc(log.Chats(), func(chat *Chat) bool { return !IsReadOnly(chat.Topic, me) })
	slices.SortFunc(chats, func(first, second *Chat) int { return strings.Compare(first.Topic, second.Topic) })
	return chats
}
