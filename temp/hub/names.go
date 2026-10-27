package hub

import "strings"

// maximumNameLength keeps names short enough to read in a list.
const maximumNameLength = 64

// ownTopicPrefix starts the topic that belongs to one session: "@alice" is alice's inbox.
const ownTopicPrefix = "@"

// OwnTopic is the topic only that session listens to, where anyone can write to it alone.
func OwnTopic(session string) string { return ownTopicPrefix + session }

// validName tells whether a session or topic name (without its '@') is short and plain enough
// to type in a shell without quotes.
func validName(name string) bool {
	if name == "" || len(name) > maximumNameLength {
		return false
	}
	for _, character := range name {
		isPlain := character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' || strings.ContainsRune("-_./", character)
		if !isPlain {
			return false
		}
	}
	return true
}

// pairSeparator joins the two names of a pair. Names cannot hold it, so it is never ambiguous.
const pairSeparator = "+"

// PairOf is the name of the conversation between two sessions: their names sorted and joined by
// '+', so it is the same whoever writes first. PairOf("bob", "alice") is "alice+bob".
func PairOf(first, second string) string {
	if second < first {
		first, second = second, first
	}
	return first + pairSeparator + second
}

// pairMembers are the two sessions of a pair's name; false when the topic is not a pair: two
// different valid names, sorted, joined by '+'.
func pairMembers(topic string) (first, second string, isPair bool) {
	first, second, found := strings.Cut(topic, pairSeparator)
	if !found || !validName(first) || !validName(second) || first >= second {
		return "", "", false
	}
	return first, second, true
}

// IsPair tells whether a topic is the conversation between two sessions.
func IsPair(topic string) bool {
	_, _, isPair := pairMembers(topic)
	return isPair
}

// validTopic tells whether a topic name is valid: a plain name, '@' and a plain name, or a pair.
func validTopic(topic string) bool {
	return validName(strings.TrimPrefix(topic, ownTopicPrefix)) || IsPair(topic)
}
