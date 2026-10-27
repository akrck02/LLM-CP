package main

import (
	"encoding/json"
	"log/slog"
	"maps"

	"github.com/akrck02/go-ui/window"
)

// readPreference is where a name's reading is kept: per name, so two people on one device, or
// one person under two names, each find their own chats read.
func readPreference(name string) string { return "read:" + name }

// keepReading recalls the reading of the name in use the first time it is known, and keeps it
// again whenever a chat opened changes it. Called once a frame.
func (chat *screen) keepReading() {
	if chat.me == "" {
		return
	}
	if chat.readingOf != chat.me {
		chat.recallReading()
		return
	}
	marks := chat.log.ReadMarks()
	if maps.Equal(marks, chat.readingKept) {
		return
	}
	text, err := json.Marshal(marks)
	if err == nil {
		err = window.KeepPreference(readPreference(chat.me), string(text))
	}
	if err != nil {
		slog.Warn("chat: what was read will show as unread next time", "error", err)
	}
	chat.readingKept = marks
}

// recallReading takes the reading kept for the name in use, if any.
func (chat *screen) recallReading() {
	chat.readingOf = chat.me
	marks := map[string]uint64{}
	if text, found := window.Preference(readPreference(chat.me)); found {
		if err := json.Unmarshal([]byte(text), &marks); err != nil {
			slog.Warn("chat: the reading kept could not be read; starting afresh", "error", err)
		}
	}
	chat.log.SetReadMarks(marks)
	chat.readingKept = chat.log.ReadMarks()
}
