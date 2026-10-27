package server

import (
	"context"

	"github.com/akrck02/LLM-CP/hub"
	"github.com/akrck02/LLM-CP/protocol"
)

func (served *service) Watch(asked *protocol.WatchRequest, stream protocol.Hub_WatchServer) error {
	watcher, past := served.hub.Watch(asked.Past)
	defer served.hub.Unwatch(watcher)
	for _, message := range past {
		if err := stream.Send(messageEvent(message)); err != nil {
			return err
		}
	}
	for {
		select {
		case event, open := <-watcher.Events:
			if !open {
				return asStatus(hub.WatcherBehind)
			}
			if err := stream.Send(toEvent(event)); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}

func (served *service) CreateGroup(_ context.Context, request *protocol.GroupRequest) (*protocol.Done, error) {
	return &protocol.Done{}, asStatus(served.hub.CreateGroup(request.Session, request.Group, request.Theme, request.Members))
}

func (served *service) AddMember(_ context.Context, request *protocol.MemberRequest) (*protocol.Done, error) {
	return &protocol.Done{}, asStatus(served.hub.AddMember(request.Session, request.Group, request.Member))
}

func (served *service) RemoveMember(_ context.Context, request *protocol.MemberRequest) (*protocol.Done, error) {
	return &protocol.Done{}, asStatus(served.hub.RemoveMember(request.Session, request.Group, request.Member))
}

func (served *service) RemoveGroup(_ context.Context, request *protocol.TopicRequest) (*protocol.Done, error) {
	return &protocol.Done{}, asStatus(served.hub.RemoveGroup(request.Session, request.Topic))
}

func (served *service) Groups(context.Context, *protocol.SessionRequest) (*protocol.GroupList, error) {
	return &protocol.GroupList{Groups: toGroups(served.hub.Groups())}, nil
}

func (served *service) Archive(_ context.Context, request *protocol.TopicRequest) (*protocol.Done, error) {
	return &protocol.Done{}, asStatus(served.hub.Archive(request.Session, request.Topic))
}

func (served *service) Restore(_ context.Context, request *protocol.TopicRequest) (*protocol.Done, error) {
	return &protocol.Done{}, asStatus(served.hub.Restore(request.Session, request.Topic))
}

func (served *service) Typing(_ context.Context, request *protocol.TypingRequest) (*protocol.Done, error) {
	return &protocol.Done{}, asStatus(served.hub.Typing(request.Session, !request.Stopped))
}

func (served *service) Declare(_ context.Context, request *protocol.KindRequest) (*protocol.Done, error) {
	return &protocol.Done{}, asStatus(served.hub.Declare(request.Session, request.Topic, hub.Kind(request.Kind)))
}

func (served *service) RemoveTopic(_ context.Context, request *protocol.TopicRequest) (*protocol.Done, error) {
	return &protocol.Done{}, asStatus(served.hub.RemoveTopic(request.Session, request.Topic))
}

// toEvent is a hub event as it travels.
func toEvent(event hub.Event) *protocol.Event {
	if event.Message != nil {
		return messageEvent(*event.Message)
	}
	state := &protocol.State{Groups: toGroups(event.State.Groups), Archived: event.State.Archived}
	for _, kind := range event.State.Kinds {
		state.Kinds = append(state.Kinds, &protocol.TopicKind{Topic: kind.Topic, Kind: string(kind.Kind), Publishers: kind.Publishers})
	}
	for _, contact := range event.State.Contacts {
		travelling := &protocol.Contact{
			Name: contact.Name, Topics: contact.Topics, Groups: contact.Groups, LastSeen: contact.LastSeen.Unix(),
		}
		if !contact.TypingUntil.IsZero() {
			travelling.TypingUntil = contact.TypingUntil.UnixMilli()
		}
		state.Contacts = append(state.Contacts, travelling)
	}
	return &protocol.Event{Kind: &protocol.Event_State{State: state}}
}

// messageEvent is a message as an event.
func messageEvent(message hub.Message) *protocol.Event {
	return &protocol.Event{Kind: &protocol.Event_Message{Message: toProtocol(message)}}
}

// toGroups is the hub's groups as they travel.
func toGroups(groups []hub.Group) []*protocol.Group {
	travelling := make([]*protocol.Group, 0, len(groups))
	for _, group := range groups {
		travelling = append(travelling, &protocol.Group{Name: group.Name, Theme: group.Theme, Members: group.Members})
	}
	return travelling
}
