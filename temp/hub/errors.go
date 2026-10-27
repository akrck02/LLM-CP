package hub

import "github.com/akrck02/LLM-CP/internal/failure"

// The codes of every error the hub returns.
var (
	SessionNotFound = failure.Define("hub.session-not-found", "no session has that name; join first")
	NameTaken       = failure.Define("hub.name-taken", "another session already has that name")
	InvalidName     = failure.Define("hub.invalid-name", "names use letters, digits, '-', '_', '.' and '/', at most 64")
	TopicReserved   = failure.Define("hub.topic-reserved", "a topic starting with '@' belongs to the session of that name")
	NotListening    = failure.Define("hub.not-listening", "the session does not listen to that topic")
	EmptyMessage    = failure.Define("hub.empty-message", "a message needs some text")
	GroupExists     = failure.Define("hub.group-exists", "a group of that name exists already")
	GroupNotFound   = failure.Define("hub.group-not-found", "no group has that name")
	NotAMember      = failure.Define("hub.not-a-member", "only a group's members publish to it")
	WatcherBehind   = failure.Define("hub.watcher-behind", "the watcher fell too far behind; watch again")
	KindTaken       = failure.Define("hub.kind-taken", "the topic has another kind; remove it to make it again")
	UnknownKind     = failure.Define("hub.unknown-kind", "a topic is a conversation or metrics")
	NotAPoint       = failure.Define("hub.not-a-point", `a metrics topic takes {"name", "title", "value", "unit", "labels", "at"}`)
	PairFixed       = failure.Define("hub.pair-fixed", "a pair is always its two sessions: it cannot change members or be removed")
	KeepUnreadable  = failure.Define("hub.keep-unreadable", "the file of kept topics and groups cannot be read")
	KeepUnwritable  = failure.Define("hub.keep-unwritable", "the change was made, but the file of kept topics and groups could not be written")
)

// Codes lists every code, so a program can find one by its name.
var Codes = []*failure.Code{SessionNotFound, NameTaken, InvalidName, TopicReserved, NotListening, EmptyMessage,
	GroupExists, GroupNotFound, NotAMember, WatcherBehind, KindTaken, UnknownKind, NotAPoint,
	PairFixed, KeepUnreadable, KeepUnwritable}

// CodeNamed is the code with that name, or nil if there is none.
func CodeNamed(name string) *failure.Code {
	for _, code := range Codes {
		if code.Name() == name {
			return code
		}
	}
	return nil
}
