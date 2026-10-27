package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/akrck02/LLM-CP/hub"
	"github.com/akrck02/LLM-CP/internal/failure"
	"github.com/akrck02/LLM-CP/protocol"
	"github.com/akrck02/LLM-CP/server"
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

// subscription is a change the page asks for: one person subscribing to one topic, or not.
type subscription struct {
	As         string `json:"as"`
	Topic      string `json:"topic"`
	Subscribed bool   `json:"subscribed"`
}

// serveSubscriptions answers GET ?as=<name> with the person's topics as a JSON list, making
// their session listen to them again on the way, and takes a POST of one change.
func (sender publisher) serveSubscriptions() http.Handler {
	change := fromOwnPage(http.HandlerFunc(sender.serveSubscriptionChange))
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			sender.serveSubscriptionList(response, request)
		case http.MethodPost:
			change.ServeHTTP(response, request)
		default:
			http.Error(response, "use GET or POST", http.StatusMethodNotAllowed)
		}
	})
}

// serveSubscriptionList gives a person's topics. Opening the page is often the first call after
// the hub forgot them, so the session listens to each topic again, joining first if needed.
func (sender publisher) serveSubscriptionList(response http.ResponseWriter, request *http.Request) {
	if !sameOrigin(request) {
		http.Error(response, "only the chat's own page may do this", http.StatusForbidden)
		return
	}
	as := request.URL.Query().Get("as")
	topics := sender.kept.of(as)
	if err := sender.withClient(request.Context(), as, func(client protocol.HubClient) error {
		return listenTo(request.Context(), client, as, topics)
	}); err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(response).Encode(slices.Concat([]string{}, topics)); err != nil {
		slog.Warn("could not send the subscriptions", "as", as, "error", err)
	}
}

// serveSubscriptionChange subscribes a person to a topic or unsubscribes them: on the hub first,
// so a refused topic is not kept, then in the file.
func (sender publisher) serveSubscriptionChange(response http.ResponseWriter, request *http.Request) {
	var asked subscription
	if err := json.NewDecoder(http.MaxBytesReader(response, request.Body, maximumMessageBytes)).Decode(&asked); err != nil {
		http.Error(response, "a subscription is {as, topic, subscribed}", http.StatusBadRequest)
		return
	}
	if err := sender.subscribe(request.Context(), asked); err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

// subscribe makes the person's session listen to the topic or stop, and keeps the choice.
func (sender publisher) subscribe(request context.Context, asked subscription) error {
	err := sender.withClient(request, asked.As, func(client protocol.HubClient) error {
		if asked.Subscribed {
			return listenTo(request, client, asked.As, []string{asked.Topic})
		}
		_, err := client.Unlisten(request, &protocol.TopicRequest{Session: asked.As, Topic: asked.Topic})
		if server.CodeName(err) == hub.NotListening.Name() {
			return nil // not listening is what was asked: the hub may have forgotten the session
		}
		return err
	})
	if err != nil {
		return err
	}
	return sender.kept.set(asked.As, asked.Topic, asked.Subscribed)
}

// listenTo makes a session listen to topics; listening again to one it listens to does nothing.
func listenTo(request context.Context, client protocol.HubClient, as string, topics []string) error {
	for _, topic := range topics {
		if _, err := client.Listen(request, &protocol.ListenRequest{Session: as, Topic: topic}); err != nil {
			return err
		}
	}
	return nil
}
