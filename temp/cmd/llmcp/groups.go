package main

import (
	"strings"

	"github.com/akrck02/LLM-CP/internal/failure"
	"github.com/akrck02/LLM-CP/protocol"
)

// group runs one of the group commands: create, add, remove or delete.
func group(sent call) error {
	if err := sent.given.requireArguments(2); err != nil {
		return err
	}
	action, name := sent.given.arguments[0], sent.given.arguments[1]
	switch action {
	case "create":
		return createGroup(sent, name)
	case "add", "remove":
		return changeMember(sent, action, name)
	case "delete":
		asked := &protocol.TopicRequest{Session: sent.session, Topic: name}
		if _, err := sent.client.RemoveGroup(sent.request, asked); err != nil {
			return err
		}
		return sent.answer.done(name + " is a plain topic again")
	}
	return failure.Wrap(UnknownCommand, "group "+action, nil)
}

// createGroup makes a group with the theme and members given as options.
func createGroup(sent call, name string) error {
	members := splitNames(sent.given.members)
	asked := &protocol.GroupRequest{Session: sent.session, Group: name, Theme: sent.given.theme, Members: members}
	if _, err := sent.client.CreateGroup(sent.request, asked); err != nil {
		return err
	}
	return sent.answer.done("group " + name + " created; its members listen to it")
}

// splitNames reads names written with commas, as people write them: "alice, bob," is alice and bob.
func splitNames(written string) []string {
	var names []string
	for _, name := range strings.Split(written, ",") {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// changeMember adds a member to a group, or removes one.
func changeMember(sent call, action, name string) error {
	if err := sent.given.requireArguments(3); err != nil {
		return err
	}
	member := sent.given.arguments[2]
	asked := &protocol.MemberRequest{Session: sent.session, Group: name, Member: member}
	if action == "add" {
		if _, err := sent.client.AddMember(sent.request, asked); err != nil {
			return err
		}
		return sent.answer.done(member + " is in " + name)
	}
	if _, err := sent.client.RemoveMember(sent.request, asked); err != nil {
		return err
	}
	return sent.answer.done(member + " left " + name)
}

// groups lists every group with its theme and members.
func groups(sent call) error {
	list, err := sent.client.Groups(sent.request, &protocol.SessionRequest{Session: sent.session})
	if err != nil {
		return err
	}
	return sent.answer.groups(list.Groups)
}

// archive keeps the topic in the arguments out of the list of chats.
func archive(sent call) error {
	if err := sent.given.requireArguments(1); err != nil {
		return err
	}
	topic := sent.given.arguments[0]
	if _, err := sent.client.Archive(sent.request, &protocol.TopicRequest{Session: sent.session, Topic: topic}); err != nil {
		return err
	}
	return sent.answer.done(topic + " archived")
}

// restore brings the archived topic in the arguments back.
func restore(sent call) error {
	if err := sent.given.requireArguments(1); err != nil {
		return err
	}
	topic := sent.given.arguments[0]
	if _, err := sent.client.Restore(sent.request, &protocol.TopicRequest{Session: sent.session, Topic: topic}); err != nil {
		return err
	}
	return sent.answer.done(topic + " restored")
}

// typing says the session is thinking of an answer, or with --stop that it is not any more.
func typing(sent call) error {
	asked := &protocol.TypingRequest{Session: sent.session, Stopped: sent.given.stop}
	if _, err := sent.client.Typing(sent.request, asked); err != nil {
		return err
	}
	if sent.given.stop {
		return sent.answer.done(sent.session + " stopped typing")
	}
	return sent.answer.done(sent.session + " is typing")
}
