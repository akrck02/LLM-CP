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
)

// Codes lists every code, so a program can find one by its name.
var Codes = []*failure.Code{SessionNotFound, NameTaken, InvalidName, TopicReserved, NotListening, EmptyMessage}

// CodeNamed is the code with that name, or nil if there is none.
func CodeNamed(name string) *failure.Code {
	for _, code := range Codes {
		if code.Name() == name {
			return code
		}
	}
	return nil
}
