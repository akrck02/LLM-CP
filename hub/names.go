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

// validTopic tells whether a topic name is valid: a plain name, or '@' and a plain name.
func validTopic(topic string) bool {
	return validName(strings.TrimPrefix(topic, ownTopicPrefix))
}
