package hub

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/akrck02/LLM-CP/internal/failure"
)

// kept is what outlives the hub: the declared topics and the groups, without their messages.
type kept struct {
	Topics []keptTopic `json:"topics"`
	Groups []keptGroup `json:"groups"`
}

type keptTopic struct {
	Topic    string `json:"topic"`
	Kind     Kind   `json:"kind"`
	Archived bool   `json:"archived,omitempty"`
}

type keptGroup struct {
	Name     string   `json:"name"`
	Theme    string   `json:"theme,omitempty"`
	Members  []string `json:"members"`
	Archived bool     `json:"archived,omitempty"`
}

// KeepIn makes the hub keep its declared topics and its groups in a file: it reads them back now,
// and writes them again after every change to them. A missing file is an empty start.
func (hub *Hub) KeepIn(path string) error {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	written, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return failure.Wrap(KeepUnreadable, path, err)
	}
	if err == nil {
		var read kept
		if err := json.Unmarshal(written, &read); err != nil {
			return failure.Wrap(KeepUnreadable, path, err)
		}
		hub.restore(read)
	}
	hub.keepPath = path
	return nil
}

// restore puts back what was kept. The caller holds the mutex.
func (hub *Hub) restore(read kept) {
	for _, topic := range read.Topics {
		hub.kinds[topic.Topic] = topic.Kind
		if topic.Archived {
			hub.archived[topic.Topic] = true
		}
	}
	for _, group := range read.Groups {
		hub.groups[group.Name] = &Group{Name: group.Name, Theme: group.Theme, Members: group.Members}
		if group.Archived {
			hub.archived[group.Name] = true
		}
	}
}

// keep writes the declared topics and the groups to the hub's file, if it has one, through a
// temporary file and a rename, so a crash never leaves half of it. The caller holds the mutex.
func (hub *Hub) keep() error {
	if hub.keepPath == "" {
		return nil
	}
	var writing kept
	for _, kind := range hub.kindList() {
		writing.Topics = append(writing.Topics, keptTopic{Topic: kind.Topic, Kind: kind.Kind, Archived: hub.archived[kind.Topic]})
	}
	for _, group := range hub.groupList() {
		writing.Groups = append(writing.Groups, keptGroup{Name: group.Name, Theme: group.Theme, Members: group.Members, Archived: hub.archived[group.Name]})
	}
	written, err := json.MarshalIndent(writing, "", "  ")
	if err != nil {
		return failure.Wrap(KeepUnwritable, hub.keepPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(hub.keepPath), 0o700); err != nil {
		return failure.Wrap(KeepUnwritable, hub.keepPath, err)
	}
	temporary := hub.keepPath + ".new"
	if err := os.WriteFile(temporary, written, 0o600); err != nil {
		return failure.Wrap(KeepUnwritable, hub.keepPath, err)
	}
	if err := os.Rename(temporary, hub.keepPath); err != nil {
		return failure.Wrap(KeepUnwritable, hub.keepPath, err)
	}
	return nil
}
