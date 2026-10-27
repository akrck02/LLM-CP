package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/akrck02/LLM-CP/internal/failure"
)

// subscriptions keeps, for each person's name, the topics they are subscribed to. The file is
// the truth: the hub forgets a silent session, so its listens are only a mirror of this.
type subscriptions struct {
	mutex  sync.Mutex
	path   string
	topics map[string][]string // per name, sorted
}

// loadSubscriptions reads the subscriptions kept in a file; a missing file is an empty start.
func loadSubscriptions(path string) (*subscriptions, error) {
	kept := &subscriptions{path: path, topics: map[string][]string{}}
	written, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return kept, nil
	}
	if err != nil {
		return nil, failure.Wrap(SubscriptionsUnreadable, path, err)
	}
	if err := json.Unmarshal(written, &kept.topics); err != nil {
		return nil, failure.Wrap(SubscriptionsUnreadable, path, err)
	}
	return kept, nil
}

// of is the topics a person is subscribed to, sorted; someone new has none.
func (kept *subscriptions) of(name string) []string {
	kept.mutex.Lock()
	defer kept.mutex.Unlock()
	return slices.Clone(kept.topics[name])
}

// set subscribes a person to a topic, or unsubscribes them, and writes the file again.
func (kept *subscriptions) set(name, topic string, subscribed bool) error {
	kept.mutex.Lock()
	defer kept.mutex.Unlock()
	topics := slices.DeleteFunc(slices.Clone(kept.topics[name]), func(each string) bool { return each == topic })
	if subscribed {
		topics = append(topics, topic)
		slices.Sort(topics)
	}
	kept.topics[name] = topics
	if len(topics) == 0 {
		delete(kept.topics, name)
	}
	return kept.save()
}

// save writes every subscription through a temporary file and a rename, so a crash never leaves
// half of it. The caller holds the mutex.
func (kept *subscriptions) save() error {
	written, err := json.MarshalIndent(maps.Clone(kept.topics), "", "  ")
	if err != nil {
		return failure.Wrap(SubscriptionsUnwritable, kept.path, err)
	}
	if err := os.MkdirAll(filepath.Dir(kept.path), 0o700); err != nil {
		return failure.Wrap(SubscriptionsUnwritable, kept.path, err)
	}
	temporary := kept.path + ".new"
	if err := os.WriteFile(temporary, written, 0o600); err != nil {
		return failure.Wrap(SubscriptionsUnwritable, kept.path, err)
	}
	if err := os.Rename(temporary, kept.path); err != nil {
		return failure.Wrap(SubscriptionsUnwritable, kept.path, err)
	}
	return nil
}
